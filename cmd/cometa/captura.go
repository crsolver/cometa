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

	"cometa/internal/diagnostic"
	"cometa/internal/stdlib"
)

var errCaptureUsage = errors.New("uso: cometa captura <archivo.cometa> [-o captura.png] [--cuadros N,M,...] [--escala 1-64] [--entrada guion.txt]")

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
// _hgguion* constants in runtime.txt (0 hold, 1 release, 2 tap, 3 move).
type scriptEvent struct {
	frame int
	kind  int
	code  string
	x, y  float64
}

// readScript parses an input script. Each line is a tick followed by actions:
//
//	60 +D +Espacio    hold keys from tick 60
//	90 -D             release
//	100 Enter         press for one tick
//	120 raton 160 90  move the pointer (logical screen pixels)
//	121 RatonIzquierdo
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
		text, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		fail := func(format string, args ...any) error {
			return fmt.Errorf("%s:%d: %s", path, line, fmt.Sprintf(format, args...))
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
				return nil, fail("tecla desconocida %q%s (usa los nombres de entrada.Tecla, como D, Enter o Izquierda, o RatonIzquierdo)", action, diagnostic.Hint(action, stdlib.InputNames()))
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
func captureProgram(source []byte, output string, frames []int, scale int, events []scriptEvent) error {
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
			fmt.Fprintf(&init, "\t\t{%d, %d, %s, %g, %g},\n", e.frame, e.kind, code, e.x, e.y)
		}
		init.WriteString("\t}\n")
	}
	init.WriteString("}\n")
	if err = buildProgram(append(source, init.String()...), "", true); err != nil {
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
