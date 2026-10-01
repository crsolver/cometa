// Package cometa is the stable Go API of the Cometa compiler, for programs
// outside this module (platform servers, graders, build tools) that would
// otherwise have to run the cometa binary as a process.
//
// Everything under internal/ may change between versions; this package only
// exposes plain values: source paths or in-memory files go in, generated Go
// source and diagnostics come out.
package cometa

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cometa/internal/compiler"
	"cometa/internal/diagnostic"
)

// Diagnostic is one problem found in a Cometa program. File is empty for
// failures that have no source position, such as an unreadable entry file.
// The JSON field names match the output of `cometa probar --json`.
type Diagnostic struct {
	File    string `json:"archivo,omitempty"`
	Line    int    `json:"linea,omitempty"`
	Column  int    `json:"columna,omitempty"`
	Stage   string `json:"etapa,omitempty"` // lexer, parser, sema or frontend
	Message string `json:"mensaje"`
}

func (d Diagnostic) String() string {
	if d.File == "" {
		return d.Message
	}
	return fmt.Sprintf("%s:%d:%d: %s", d.File, d.Line, d.Column, d.Message)
}

// Error is the only error type returned by Check and Compile. It always holds
// at least one diagnostic, sorted by file and position.
type Error struct {
	Diagnostics []Diagnostic
}

func (e *Error) Error() string {
	lines := make([]string, len(e.Diagnostics))
	for i, d := range e.Diagnostics {
		lines[i] = d.String()
	}
	return strings.Join(lines, "\n")
}

// Options configures where sources come from and how Go is generated. The
// zero value reads the entry file and its imports from disk.
type Options struct {
	// Files is an in-memory project keyed by slash-separated relative paths
	// ("principal.cometa", "mundo/mapa.cometa", "arte/heroe.png"). The entry
	// passed to Check or Compile must be one of its keys, and diagnostics
	// report these same keys. When Files is set the disk is never read.
	Files map[string][]byte

	// Loader replaces reading from disk when Files is nil. It receives
	// absolute paths (lowercased on Windows) of imported modules and embedded
	// resources, so an editor can overlay unsaved buffers.
	Loader func(path string) ([]byte, error)

	// LineDirectives emits //line comments so Go build errors and runtime
	// panics of the generated program point at Cometa source lines.
	LineDirectives bool
}

// Check analyzes a program without generating Go. It returns nil or *Error.
func Check(entry string, options Options) error {
	_, err := run(entry, options, false)
	return err
}

// Compile translates a program into the source of a Go main package. It
// returns the generated code, or nil and *Error.
func Compile(entry string, options Options) ([]byte, error) {
	return run(entry, options, true)
}

func run(entry string, options Options, generate bool) ([]byte, error) {
	loader := options.Loader
	var memory *memoryProject
	if options.Files != nil {
		var err error
		if memory, err = newMemoryProject(options.Files); err != nil {
			return nil, &Error{Diagnostics: []Diagnostic{{Message: err.Error()}}}
		}
		if _, ok := memory.keys[memory.path(entry)]; !ok {
			return nil, &Error{Diagnostics: []Diagnostic{{Message: fmt.Sprintf("el archivo de entrada %q no está entre los archivos del proyecto", entry)}}}
		}
		entry = memory.path(entry)
		loader = memory.load
	}
	var (
		generated []byte
		err       error
	)
	switch {
	case !generate:
		_, err = compiler.AnalyzeProject(entry, loader)
	case options.LineDirectives:
		generated, err = compiler.CompileProjectForRun(entry, loader)
	default:
		generated, err = compiler.CompileProject(entry, loader)
	}
	if err != nil {
		return nil, newError(err, memory)
	}
	if memory != nil && options.LineDirectives {
		generated = memory.relativize(generated)
	}
	return generated, nil
}

func newError(err error, memory *memoryProject) *Error {
	out := &Error{}
	for _, e := range diagnostic.Flatten(err) {
		var located diagnostic.Located
		if !errors.As(e, &located) {
			message := e.Error()
			if memory != nil {
				message = string(memory.relativize([]byte(message)))
			}
			out.Diagnostics = append(out.Diagnostics, Diagnostic{Message: message})
			continue
		}
		file, pos, stage, message := located.Diagnostic()
		if file == "" {
			file = pos.Filename
		}
		if memory != nil {
			file = memory.key(file)
			message = string(memory.relativize([]byte(message)))
		}
		out.Diagnostics = append(out.Diagnostics, Diagnostic{File: file, Line: pos.Line, Column: pos.Column, Stage: stage, Message: message})
	}
	return out
}

// memoryProject places in-memory files under a virtual directory that is
// never read, so the compiler can resolve relative imports as usual.
type memoryProject struct {
	root  string            // canonical virtual directory, with trailing separator
	keys  map[string]string // canonical path -> caller's key
	files map[string][]byte // canonical path -> contents
}

func newMemoryProject(files map[string][]byte) (*memoryProject, error) {
	// The directory name becomes part of the data folder of games that save
	// files, so it is fixed rather than random.
	root, err := compiler.CanonicalPath(filepath.Join(os.TempDir(), "cometa-memoria", "proyecto"))
	if err != nil {
		return nil, err
	}
	m := &memoryProject{root: root + string(filepath.Separator), keys: map[string]string{}, files: map[string][]byte{}}
	for key, data := range files {
		path := m.path(key)
		if !strings.HasPrefix(path, m.root) {
			return nil, fmt.Errorf("la ruta %q sale del proyecto", key)
		}
		m.keys[path] = key
		m.files[path] = data
	}
	return m, nil
}

func (m *memoryProject) path(key string) string {
	path, err := compiler.CanonicalPath(filepath.Join(m.root, filepath.FromSlash(key)))
	if err != nil {
		return ""
	}
	return path
}

func (m *memoryProject) load(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, fmt.Errorf("el archivo %s no existe en el proyecto", m.key(path))
	}
	return data, nil
}

// key maps a canonical path back to the caller's spelling of it.
func (m *memoryProject) key(path string) string {
	if key, ok := m.keys[path]; ok {
		return key
	}
	if strings.HasPrefix(path, m.root) {
		return filepath.ToSlash(strings.TrimPrefix(path, m.root))
	}
	return path
}

// relativize strips the virtual directory from generated code and messages.
func (m *memoryProject) relativize(text []byte) []byte {
	return bytes.ReplaceAll(text, []byte(m.root), nil)
}
