// Package rt is the Go runtime of the standard library: the implementation of
// every function and type declared in the catalog of package stdlib.
//
// Nothing imports this package. stdlib embeds its files and copies into each
// generated program the declarations that program reaches, so a console
// program never carries Ebitengine. It is a package only so that gofmt, go vet
// and the editor treat the runtime as the Go code it is.
//
// Every name starts with _hg, which programs cannot declare. A catalog entry
// with GoName "limpiar" is implemented by _hglimpiar; a method is a Go method
// of the type, exported.
package rt

// A generated program defines these itself (internal/codegen); they are
// declared here so the package compiles on its own. Nothing in this file is
// copied into a program.

// _hgdatosCarpeta names the folder of std/pincel/datos for the program.
var _hgdatosCarpeta string

// _hentero converts a decimal to an integer, panicking when it does not fit.
func _hentero(v float64) int64 { return int64(v) }
