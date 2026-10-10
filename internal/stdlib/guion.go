package stdlib

import (
	"sort"
	"strings"
)

// InputCode returns the Go integer expression that `cometa captura --entrada`
// stores for a key (an entrada.Tecla constant such as "D" or "Enter") or a
// mouse button ("Raton" plus a BotonRaton constant, e.g. "RatonIzquierdo") or
// a button of gamepad 0 ("Mando" plus a BotonMando constant, e.g. "MandoA").
// The codes match _hgEventoGuion in rt/pincel.go.
func InputCode(name string) (string, bool) {
	if strings.HasPrefix(name, "Raton") {
		if value, ok := Constants[Symbol("BotonRaton")][strings.TrimPrefix(name, "Raton")]; ok {
			return "-1-int(" + unwrapConstant(value) + ")", true
		}
	}
	if strings.HasPrefix(name, "Mando") {
		if value, ok := Constants[Symbol("BotonMando")][strings.TrimPrefix(name, "Mando")]; ok {
			return "-100-int(" + unwrapConstant(value) + ")", true
		}
	}
	if value, ok := Constants[Symbol("Tecla")][name]; ok {
		return "int(" + unwrapConstant(value) + ")", true
	}
	return "", false
}

// InputNames lists every name InputCode accepts, for suggestions.
func InputNames() []string {
	var names []string
	for name := range Constants[Symbol("Tecla")] {
		names = append(names, name)
	}
	for name := range Constants[Symbol("BotonRaton")] {
		names = append(names, "Raton"+name)
	}
	for name := range Constants[Symbol("BotonMando")] {
		names = append(names, "Mando"+name)
	}
	sort.Strings(names)
	return names
}

// AxisCode returns the Go integer expression for a gamepad stick (an
// entrada.EjeMando constant such as "IzquierdoX").
func AxisCode(name string) (string, bool) {
	if value, ok := Constants[Symbol("EjeMando")][name]; ok {
		return "int(" + unwrapConstant(value) + ")", true
	}
	return "", false
}

// AxisNames lists every name AxisCode accepts, for suggestions.
func AxisNames() []string {
	var names []string
	for name := range Constants[Symbol("EjeMando")] {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// unwrapConstant turns "_hgTecla(ebiten.KeyD)" into "ebiten.KeyD", so the code
// does not depend on the entrada types being emitted.
func unwrapConstant(value string) string {
	return strings.TrimSuffix(value[strings.Index(value, "(")+1:], ")")
}
