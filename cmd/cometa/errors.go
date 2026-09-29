package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// errReported marks a failure that was already explained to the user.
var errReported = errors.New("error ya informado")

var (
	cometaLocation = regexp.MustCompile(`(\S[^\n\t]*?\.cometa):(\d+)`)
	indexRange     = regexp.MustCompile(`index out of range \[(-?\d+)\] with length (\d+)`)
	indexNegative  = regexp.MustCompile(`index out of range \[(-?\d+)\]`)
	sliceRange     = regexp.MustCompile(`slice bounds out of range`)
)

// crashFilter forwards the game's stderr unchanged until Go starts printing a
// panic; from then on it holds the text so it can be shown in Cometa terms.
type crashFilter struct {
	out     io.Writer
	partial []byte
	crashed bool
	trace   bytes.Buffer
}

func (c *crashFilter) Write(data []byte) (int, error) {
	c.partial = append(c.partial, data...)
	for {
		end := bytes.IndexByte(c.partial, '\n')
		if end < 0 {
			return len(data), nil
		}
		line := c.partial[:end+1]
		c.partial = c.partial[end+1:]
		c.consume(line)
	}
}

func (c *crashFilter) consume(line []byte) {
	if !c.crashed && (bytes.HasPrefix(line, []byte("panic: ")) || bytes.HasPrefix(line, []byte("fatal error: "))) {
		c.crashed = true
	}
	if c.crashed {
		c.trace.Write(line)
		return
	}
	c.out.Write(line)
}

// Flush handles a final line that had no trailing newline.
func (c *crashFilter) Flush() {
	if len(c.partial) > 0 {
		c.consume(c.partial)
		c.partial = nil
	}
}

// Report prints a translated crash message. It returns false when the game
// did not crash with a Go panic.
func (c *crashFilter) Report() bool {
	if !c.crashed {
		return false
	}
	fmt.Fprint(os.Stderr, describeCrash(c.trace.String()))
	return true
}

func describeCrash(trace string) string {
	first, _, _ := strings.Cut(trace, "\n")
	message := strings.TrimPrefix(strings.TrimPrefix(first, "panic: "), "fatal error: ")
	message = strings.TrimSuffix(message, " [recovered]")
	message = translateRuntimeMessage(message)
	var out strings.Builder
	fmt.Fprintf(&out, "error en ejecución: %s\n", message)
	if found := cometaLocation.FindStringSubmatch(trace); found != nil {
		fmt.Fprintf(&out, "  en %s, línea %s\n", displayPath(found[1]), found[2])
		if text := sourceLine(found[1], found[2]); text != "" {
			fmt.Fprintf(&out, "  %s | %s\n", found[2], text)
		}
	} else {
		out.WriteString("  (no se pudo determinar la línea del programa Cometa)\n")
	}
	return out.String()
}

func translateRuntimeMessage(message string) string {
	if found := indexRange.FindStringSubmatch(message); found != nil {
		return fmt.Sprintf("el índice %s está fuera de rango (la lista tiene %s elementos)", found[1], found[2])
	}
	if found := indexNegative.FindStringSubmatch(message); found != nil {
		return fmt.Sprintf("el índice %s está fuera de rango", found[1])
	}
	switch {
	case strings.Contains(message, "integer divide by zero"):
		return "división entre cero"
	case strings.Contains(message, "nil pointer dereference"):
		return "se usó un valor que no existe (error interno; por favor reporta este caso)"
	case strings.Contains(message, "assignment to entry in nil map"):
		return "se escribió en un mapa sin inicializar (error interno; por favor reporta este caso)"
	case sliceRange.MatchString(message):
		return "los límites de la sublista están fuera de rango"
	}
	return strings.TrimPrefix(message, "runtime error: ")
}

// displayPath prefers a path relative to the working directory.
func displayPath(path string) string {
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, filepath.FromSlash(path)); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

func sourceLine(path, line string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var number int
	fmt.Sscanf(line, "%d", &number)
	lines := strings.Split(string(data), "\n")
	if number < 1 || number > len(lines) {
		return ""
	}
	return strings.TrimSpace(strings.TrimRight(lines[number-1], "\r"))
}

// buildFailure explains why `go build` failed in terms a Cometa user can act on.
func buildFailure(output string, err error) error {
	lower := strings.ToLower(output)
	detail := strings.TrimSpace(output)
	switch {
	case containsAny(lower, "dial tcp", "no such host", "proxy.golang.org", "i/o timeout", "tls handshake", "connection refused", "network is unreachable", "temporary failure in name resolution"):
		return fmt.Errorf("no se pudieron descargar las dependencias del juego (Ebitengine).\nLa primera compilación necesita conexión a internet; revisa tu red o proxy y vuelve a intentarlo.\n\nDetalle de Go:\n%s", detail)
	case containsAny(lower, "requires go >=", "go.mod requires go"):
		return fmt.Errorf("la versión de Go instalada es demasiado antigua: Cometa necesita Go 1.25 o superior (https://go.dev/dl/).\n\nDetalle de Go:\n%s", detail)
	case containsAny(lower, "x11/", "pkg-config", "gcc", "cgo", "alsa", "libgl", "cc1"):
		return fmt.Errorf("faltan dependencias nativas del sistema para compilar el juego.\nEn Linux instala las bibliotecas de desarrollo de Ebitengine (https://ebitengine.org/en/documents/install.html).\n\nDetalle de Go:\n%s", detail)
	case cometaLocation.MatchString(output):
		return fmt.Errorf("error interno del compilador (por favor reporta este error): el Go generado no compila.\n%s", detail)
	}
	return fmt.Errorf("no se pudo construir el juego: %w\n%s", err, detail)
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
