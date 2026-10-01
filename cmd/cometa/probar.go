package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/crsolver/cometa/internal/codegen"
	"github.com/crsolver/cometa/internal/compiler"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/ast"
)

// exitCode is returned from run to choose the process exit status.
type exitCode int

func (e exitCode) Error() string { return "código de salida " + strconv.Itoa(int(e)) }

// Exit statuses of `cometa probar`: 0 all passed, 1 some test failed,
// 2 the program did not compile, 3 timeout, 4 internal failure.
const (
	exitTestsFailed = exitCode(1)
	exitCompile     = exitCode(2)
	exitTimeout     = exitCode(3)
	exitInternal    = exitCode(4)
)

type testResult struct {
	Name    string `json:"nombre"`
	Status  string `json:"estado"` // ok, fallo, error, tiempo_agotado, omitida
	Message string `json:"mensaje,omitempty"`
	File    string `json:"archivo,omitempty"`
	Line    int    `json:"linea,omitempty"`
	Millis  int64  `json:"ms"`
}

type diagnosticJSON struct {
	File    string `json:"archivo"`
	Line    int    `json:"linea"`
	Column  int    `json:"columna"`
	Stage   string `json:"etapa"`
	Message string `json:"mensaje"`
}

type testReport struct {
	Status      string           `json:"estado"` // ok, fallo, error_compilacion, tiempo_agotado, error
	Total       int              `json:"total"`
	Passed      int              `json:"pasaron"`
	Failed      int              `json:"fallaron"`
	Tests       []testResult     `json:"pruebas,omitempty"`
	Diagnostics []diagnosticJSON `json:"diagnosticos,omitempty"`
	Message     string           `json:"mensaje,omitempty"`
}

const probarUsage = "uso: cometa probar <pruebas.cometa> [--json] [--tiempo SEGUNDOS] [--filtro texto]"

func runProbar(args []string) error {
	var input, filter string
	asJSON := false
	seconds := 30
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			asJSON = true
		case "--tiempo", "--filtro":
			flag := args[i]
			i++
			if i >= len(args) {
				return fmt.Errorf("%s", probarUsage)
			}
			if flag == "--filtro" {
				filter = args[i]
				break
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return fmt.Errorf("%s", probarUsage)
			}
			seconds = n
		default:
			if strings.HasPrefix(args[i], "-") || input != "" {
				return fmt.Errorf("argumento inesperado %q\n%s", args[i], probarUsage)
			}
			input = args[i]
		}
	}
	if input == "" || filepath.Ext(input) != ".cometa" {
		return fmt.Errorf("%s", probarUsage)
	}
	report, code := probar(input, filter, time.Duration(seconds)*time.Second, asJSON)
	if asJSON {
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(out))
	} else {
		printReport(report)
	}
	if code != 0 {
		return code
	}
	return nil
}

func probar(input, filter string, limit time.Duration, asJSON bool) (report testReport, code exitCode) {
	project, err := compiler.AnalyzeProject(input, nil)
	var generated []byte
	if err == nil {
		generated, err = codegen.GenerateWithOptions(project.Root.Path, project.Program, project.Model, codegen.Options{LineDirectives: true})
	}
	if err != nil {
		return compileFailure(err, filepath.Dir(input)), exitCompile
	}
	var names []string
	for _, decl := range project.Root.Program.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Receiver != "" {
			continue
		}
		if fn.Name == "inicio" {
			return testReport{Status: "error", Message: "el archivo de pruebas no debe declarar inicio"}, exitCompile
		}
		if strings.HasPrefix(fn.Name, "prueba_") {
			if len(fn.Params) != 0 || fn.ReturnType != nil {
				return testReport{Status: "error", Message: fn.Name + " no debe tener parámetros ni resultado"}, exitCompile
			}
			if strings.Contains(fn.Name, filter) {
				names = append(names, fn.Name)
			}
		}
	}
	report.Total = len(names)
	if len(names) == 0 {
		report.Status = "ok"
		report.Message = "no se encontraron funciones prueba_*"
		return report, 0
	}
	dir, err := os.MkdirTemp("", "cometa-probar-")
	if err != nil {
		return testReport{Status: "error", Message: err.Error()}, exitInternal
	}
	defer os.RemoveAll(dir)
	exe := filepath.Join(dir, "pruebas")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if err = goBuild(dir, generated, map[string]string{"harness.go": harnessSource(names, bytes.Contains(generated, []byte("func _hgpruebasBucle(")))}, exe); err != nil {
		return testReport{Status: "error", Message: err.Error()}, exitInternal
	}
	resultsPath := filepath.Join(dir, "resultados.jsonl")
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	child := exec.CommandContext(ctx, exe)
	child.Env = append(os.Environ(), "COMETA_PROBAR_SALIDA="+resultsPath)
	child.Stdout = os.Stdout
	if asJSON {
		child.Stdout = os.Stderr // stdout stays reserved for the JSON report
	}
	crash := &crashFilter{out: os.Stderr}
	child.Stderr = crash
	runErr := child.Run()
	crash.Flush()
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)

	report.Tests = collectResults(resultsPath, names, timedOut, crash, runErr)
	for _, t := range report.Tests {
		switch t.Status {
		case "ok":
			report.Passed++
		case "omitida":
		default:
			report.Failed++
		}
	}
	switch {
	case timedOut:
		report.Status, code = "tiempo_agotado", exitTimeout
	case runErr != nil && report.Failed == 0:
		// The harness died outside any test, e.g. no display for visual tests.
		report.Status, code = "error", exitInternal
		report.Message = "las pruebas no pudieron ejecutarse: " + runErr.Error()
	case report.Failed > 0:
		report.Status, code = "fallo", exitTestsFailed
	default:
		report.Status = "ok"
	}
	return report, code
}

// collectResults folds the harness' event log into one result per test. A test
// that started but never finished either timed out or killed the process.
func collectResults(path string, names []string, timedOut bool, crash *crashFilter, runErr error) []testResult {
	byName := map[string]*testResult{}
	if file, err := os.Open(path); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1<<20), 1<<24)
		for scanner.Scan() {
			var event struct {
				Name   string `json:"nombre"`
				Status string `json:"estado"`
				Msg    string `json:"mensaje"`
				Trace  string `json:"traza"`
				Millis int64  `json:"ms"`
			}
			if json.Unmarshal(scanner.Bytes(), &event) != nil {
				continue
			}
			result := &testResult{Name: event.Name, Status: event.Status, Message: event.Msg, Millis: event.Millis}
			if event.Status == "error" {
				result.Message = translateRuntimeMessage(result.Message)
			}
			if m := cometaLocation.FindStringSubmatch(event.Trace); m != nil && event.Status != "ok" {
				result.File = filepath.Base(m[1])
				result.Line, _ = strconv.Atoi(m[2])
			}
			byName[event.Name] = result
		}
	}
	var out []testResult
	for _, name := range names {
		result := byName[name]
		switch {
		case result == nil:
			out = append(out, testResult{Name: name, Status: "omitida"})
			continue
		case result.Status == "inicio":
			if timedOut {
				result.Status, result.Message = "tiempo_agotado", "la prueba superó el tiempo límite"
			} else {
				result.Status = "error"
				result.Message = "el programa terminó durante la prueba"
				if crash.crashed {
					result.Message = strings.TrimSpace(describeCrash(crash.trace.String()))
				} else if runErr != nil {
					result.Message += ": " + runErr.Error()
				}
			}
		}
		out = append(out, *result)
	}
	return out
}

func compileFailure(err error, base string) testReport {
	report := testReport{Status: "error_compilacion"}
	for _, e := range diagnostic.Flatten(err) {
		if located, ok := e.(diagnostic.Located); ok {
			file, pos, stage, message := located.Diagnostic()
			if file == "" {
				file = pos.Filename
			}
			if abs, e := compiler.CanonicalPath(base); e == nil && !strings.HasPrefix(file, "cometa-std:") {
				if rel, e := filepath.Rel(abs, file); e == nil {
					file = filepath.ToSlash(rel)
				}
			}
			report.Diagnostics = append(report.Diagnostics, diagnosticJSON{File: file, Line: pos.Line, Column: pos.Column, Stage: stage, Message: message})
			continue
		}
		report.Diagnostics = append(report.Diagnostics, diagnosticJSON{Message: e.Error()})
	}
	return report
}

func printReport(report testReport) {
	switch report.Status {
	case "error_compilacion":
		for _, d := range report.Diagnostics {
			if d.File != "" {
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", d.File, d.Line, d.Column, d.Message)
			} else {
				fmt.Fprintln(os.Stderr, d.Message)
			}
		}
		return
	case "error":
		fmt.Fprintln(os.Stderr, report.Message)
		return
	}
	for _, t := range report.Tests {
		where := ""
		if t.Line > 0 {
			where = fmt.Sprintf(" (%s:%d)", t.File, t.Line)
		}
		switch t.Status {
		case "ok":
			fmt.Printf("  ok      %s (%d ms)\n", t.Name, t.Millis)
		case "omitida":
			fmt.Printf("  omitida %s\n", t.Name)
		default:
			fmt.Printf("  FALLO   %s: %s%s\n", t.Name, t.Message, where)
		}
	}
	if report.Message != "" {
		fmt.Println(report.Message)
		return
	}
	fmt.Printf("%d pruebas: %d pasaron, %d fallaron\n", report.Total, report.Passed, report.Failed)
}

// harnessSource returns the Go file that runs each test in isolation and logs
// its outcome. It has no dependency on the standard-library runtime: failed
// assertions are recognised through the PruebaFallo method on their panic value.
// With visual tests (pruebas.avanzar) the tests run inside a hidden game loop,
// which Ebitengine needs before it can paint and read pixels.
func harnessSource(names []string, visual bool) string {
	reset, start := "", "run()"
	if visual {
		reset = "_hgpruebasReiniciar()"
		start = `if err := _hgpruebasBucle(run); err != nil {
		fmt.Fprintln(os.Stderr, "no se pudieron iniciar las pruebas visuales:", err)
		os.Exit(4)
	}`
	}
	var list strings.Builder
	for _, name := range names {
		runes := []rune(name)
		runes[0] = unicode.ToUpper(runes[0])
		fmt.Fprintf(&list, "\t\t\t{%q, %s},\n", name, string(runes))
	}
	return `package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

func main() {
	out, err := os.OpenFile(os.Getenv("COMETA_PROBAR_SALIDA"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(4)
	}
	emit := func(event map[string]any) {
		line, _ := json.Marshal(event)
		out.Write(append(line, '\n'))
		out.Sync()
	}
	run := func() {
		for _, t := range []struct {
			name string
			run  func()
		}{
` + list.String() + `		} {
			emit(map[string]any{"nombre": t.name, "estado": "inicio"})
			` + reset + `
			start := time.Now()
			status, message, trace := _hpRun(t.run)
			emit(map[string]any{"nombre": t.name, "estado": status, "mensaje": message, "traza": trace, "ms": time.Since(start).Milliseconds()})
		}
	}
	` + start + `
}

func _hpRun(f func()) (status, message, trace string) {
	defer func() {
		if r := recover(); r != nil {
			trace = string(debug.Stack())
			if failure, ok := r.(interface{ PruebaFallo() string }); ok {
				status, message = "fallo", failure.PruebaFallo()
				return
			}
			status, message = "error", fmt.Sprint(r)
		}
	}()
	f()
	return "ok", "", ""
}
`
}

