package codegen

const stringRuntime = `
func _hsnumero(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
func _hsbool(v bool) string { if v { return "verdadero" }; return "falso" }
func _hsindice(s, buscar string) int {
	i := strings.Index(s, buscar)
	if i < 0 { return -1 }
	return len([]rune(s[:i]))
}
`
