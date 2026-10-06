package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/stdlib"
)

var errCaptureUsage = errors.New("uso: cometa captura <archivo.cometa> [-o captura.png] [--cuadros N,M,...] [--escala 1-64] [--entrada guion.txt] [--datos carpeta]")

// parseFrames reads "--cuadros 100,300,900" into ascending unique ticks.
func parseFrames(value string) ([]int, error) {
	seen := map[int]bool{}
	var frames []int
	for _, part := range strings.Split(value, ",") {
		frame, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || frame <= 0 {
			return nil, errCaptureUsage
		}
		if !seen[frame] {
			seen[frame] = true
			frames = append(frames, frame)
		}
	}
	sort.Ints(frames)
	return frames, nil
}

// scriptEvent is one line item of a `--entrada` script; kind matches the
// _hgguion* constants in runtime.txt (0 hold, 1 release, 2 tap, 3 move, 4 text,
// 5 wheel, 6 gamepad axis, whose value is x).
type scriptEvent struct {
	frame int
	kind  int
	code  string
	x, y  float64
	text  string
}

// scriptFields splits a script line into words. A double-quoted word keeps its
// spaces and Go escapes ("hola mundo", "a\nb"), and `#` outside quotes
// starts a comment. The second result marks which words were quoted.
func scriptFields(line string) ([]string, []bool, error) {
	var fields []string
	var quoted []bool
	rest := line
	for {
		rest = strings.TrimLeft(rest, " \t\r")
		if rest == "" || rest[0] == '#' {
			return fields, quoted, nil
		}
		if rest[0] == '"' {
			end := 1
			for end < len(rest) && rest[end] != '"' {
				if rest[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(rest) {
				return nil, nil, errors.New("falta cerrar las comillas")
			}
			value, err := strconv.Unquote(rest[:end+1])
			if err != nil {
				return nil, nil, fmt.Errorf("texto entre comillas no válido: %s", rest[:end+1])
			}
			fields, quoted = append(fields, value), append(quoted, true)
			rest = rest[end+1:]
			continue
		}
		end := strings.IndexAny(rest, " \t\r")
		if end < 0 {
			end = len(rest)
		}
		fields, quoted = append(fields, rest[:end]), append(quoted, false)
		rest = rest[end:]
	}
}

// readScript parses an input script. Each line is a tick followed by actions:
//
//	60 +D +Espacio    hold keys from tick 60
//	90 -D             release
//	100 Enter         press for one tick
//	120 raton 160 90  move the pointer (logical screen pixels)
//	121 RatonIzquierdo
//	130 texto "hola"  type characters (they reach ui.campo_texto)
//	140 rueda 0 -1    move the wheel for one tick
//	150 +MandoA       gamepad 0 buttons (BotonMando names with a Mando prefix)
//	160 eje IzquierdoX 1   gamepad 0 stick value, kept until changed
//
// `#` starts a comment. Events are applied in tick order, then line order.
func readScript(path string) ([]scriptEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el guion de entrada: %w", err)
	}
	defer file.Close()
	var events []scriptEvent
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		fail := func(format string, args ...any) error {
			return fmt.Errorf("%s:%d: %s", path, line, fmt.Sprintf(format, args...))
		}
		fields, quoted, err := scriptFields(scanner.Text())
		if err != nil {
			return nil, fail("%v", err)
		}
		if len(fields) == 0 {
			continue
		}
		frame, err := strconv.Atoi(fields[0])
		if err != nil || frame <= 0 {
			return nil, fail("cada línea empieza con el número de cuadro (1 o más), no %q", fields[0])
		}
		if len(fields) == 1 {
			return nil, fail("falta una acción después del cuadro %d (por ejemplo «+D», «-D», «Enter» o «raton 160 90»)", frame)
		}
		for i := 1; i < len(fields); i++ {
			action := fields[i]
			if quoted[i] {
				return nil, fail("el texto entre comillas va después de «texto»: texto %q", action)
			}
			if action == "texto" {
				if i+1 >= len(fields) || !quoted[i+1] {
					return nil, fail("«texto» necesita un texto entre comillas: texto \"hola\"")
				}
				events = append(events, scriptEvent{frame: frame, kind: 4, text: fields[i+1]})
				i++
				continue
			}
			if action == "rueda" {
				if i+2 >= len(fields) {
					return nil, fail("«rueda» necesita dos números: rueda <x> <y>")
				}
				x, errX := strconv.ParseFloat(fields[i+1], 64)
				y, errY := strconv.ParseFloat(fields[i+2], 64)
				if errX != nil || errY != nil {
					return nil, fail("«rueda» necesita dos números: rueda <x> <y>")
				}
				events = append(events, scriptEvent{frame: frame, kind: 5, x: x, y: y})
				i += 2
				continue
			}
			if action == "eje" {
				if i+2 >= len(fields) {
					return nil, fail("«eje» necesita una palanca y un valor entre -1 y 1: eje IzquierdoX 1")
				}
				code, ok := stdlib.AxisCode(fields[i+1])
				value, err := strconv.ParseFloat(fields[i+2], 64)
				if !ok {
					return nil, fail("palanca desconocida %q%s (usa los nombres de entrada.EjeMando, como IzquierdoX)", fields[i+1], diagnostic.Hint(fields[i+1], stdlib.AxisNames()))
				}
				if err != nil || value < -1 || value > 1 {
					return nil, fail("«eje» necesita un valor entre -1 y 1, no %q", fields[i+2])
				}
				events = append(events, scriptEvent{frame: frame, kind: 6, code: code, x: value})
				i += 2
				continue
			}
			if action == "raton" {
				if i+2 >= len(fields) {
					return nil, fail("«raton» necesita dos números: raton <x> <y>")
				}
				x, errX := strconv.ParseFloat(fields[i+1], 64)
				y, errY := strconv.ParseFloat(fields[i+2], 64)
				if errX != nil || errY != nil {
					return nil, fail("«raton» necesita dos números: raton <x> <y>")
				}
				events = append(events, scriptEvent{frame: frame, kind: 3, x: x, y: y})
				i += 2
				continue
			}
			kind := 2
			if strings.HasPrefix(action, "+") {
				kind, action = 0, action[1:]
			} else if strings.HasPrefix(action, "-") {
				kind, action = 1, action[1:]
			}
			code, ok := stdlib.InputCode(action)
			if !ok {
				return nil, fail("tecla desconocida %q%s (usa los nombres de entrada.Tecla, como D, Enter o Izquierda; RatonIzquierdo para el ratón o MandoA para un mando)", action, diagnostic.Hint(action, stdlib.InputNames()))
			}
			events = append(events, scriptEvent{frame: frame, kind: kind, code: code})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("no se pudo leer el guion de entrada: %w", err)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("%s: el guion de entrada no tiene acciones", path)
	}
	sort.SliceStable(events, func(a, b int) bool { return events[a].frame < events[b].frame })
	return events, nil
}

// capturePaths names one PNG per tick: the output itself for a single tick,
// otherwise <output>_<tick>.png.
func capturePaths(output string, frames []int) []string {
	if len(frames) == 1 {
		return []string{output}
	}
	base := strings.TrimSuffix(output, filepath.Ext(output))
	paths := make([]string, len(frames))
	for i, frame := range frames {
		paths[i] = fmt.Sprintf("%s_%d.png", base, frame)
	}
	return paths
}

// captureProgram runs the game with a hidden window, optionally replaying an
// input script, and saves its logical screen after each requested tick.
// std/pincel/datos works on the data folder, or on an empty temporary one.
func captureProgram(source []byte, output string, frames []int, scale int, events []scriptEvent, data string) error {
	if !strings.Contains(string(source), "func _hgcapturar(") {
		return fmt.Errorf("captura requiere un juego iniciado con pincel.ejecutar")
	}
	if len(frames) == 0 {
		// With a script, the default shot shows its outcome: one tick after
		// the last action, so taps have been released.
		frames = []int{1}
		if len(events) > 0 {
			frames[0] = events[len(events)-1].frame + 1
		}
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	paths := capturePaths(output, frames)
	var init strings.Builder
	init.WriteString("\nfunc init() {\n\t_hgcapturaRutas = []string{")
	for i, path := range paths {
		os.Remove(path)
		if i > 0 {
			init.WriteString(", ")
		}
		init.WriteString(strconv.Quote(path))
	}
	init.WriteString("}\n\t_hgcapturaCuadros = []uint64{")
	for i, frame := range frames {
		if i > 0 {
			init.WriteString(", ")
		}
		init.WriteString(strconv.Itoa(frame))
	}
	fmt.Fprintf(&init, "}\n\t_hgcapturaEscala = %d\n", scale)
	if len(events) > 0 {
		init.WriteString("\t_hgguionActivo = true\n\t_hgguion = []_hgEventoGuion{\n")
		for _, e := range events {
			code := e.code
			if code == "" {
				code = "0"
			}
			fmt.Fprintf(&init, "\t\t{%d, %d, %s, %g, %g, %s},\n", e.frame, e.kind, code, e.x, e.y, strconv.Quote(e.text))
		}
		init.WriteString("\t}\n")
	}
	init.WriteString("}\n")
	env, cleanup, err := dataEnv(data)
	defer cleanup()
	if err != nil {
		return err
	}
	if err = buildProgram(append(source, init.String()...), "", true, env); err != nil {
		return err
	}
	for i, path := range paths {
		if _, err = os.Stat(path); err != nil {
			return fmt.Errorf("el juego terminó antes de capturar el cuadro %d", frames[i])
		}
		fmt.Println("captura guardada en", path)
	}
	return nil
}
