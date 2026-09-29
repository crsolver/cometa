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

// printRuntime formats values for imprimir with Cometa spellings: verdadero/falso,
// quoted strings and comma-separated lists, maps and structures inside collections.
const printRuntime = `
func _hsimprimir(args ...any) {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = _hsformato(reflect.ValueOf(a), false, 0)
	}
	fmt.Println(strings.Join(parts, " "))
}

func _hsformato(v reflect.Value, nested bool, depth int) string {
	if !v.IsValid() {
		return "nada"
	}
	if depth > 6 {
		return "..."
	}
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() { return "verdadero" }
		return "falso"
	case reflect.String:
		if nested { return strconv.Quote(v.String()) }
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Interface:
		if v.IsNil() { return "nada" }
		return _hsformato(v.Elem(), nested, depth)
	case reflect.Slice:
		items := make([]string, v.Len())
		for i := range items {
			items[i] = _hsformato(v.Index(i), true, depth+1)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case reflect.Map:
		if v.Len() == 0 { return "[:]" }
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			a, b := keys[i], keys[j]
			switch a.Kind() {
			case reflect.String:
				return a.String() < b.String()
			case reflect.Bool:
				return !a.Bool() && b.Bool()
			}
			return a.Int() < b.Int()
		})
		items := make([]string, len(keys))
		for i, k := range keys {
			items[i] = _hsformato(k, true, depth+1) + ": " + _hsformato(v.MapIndex(k), true, depth+1)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case reflect.Ptr:
		if !v.IsNil() && v.Elem().Kind() == reflect.Struct {
			return _hsformato(v.Elem(), nested, depth)
		}
	case reflect.Struct:
		t := v.Type()
		if t.Name() == "" && t.NumField() == 2 && t.Field(0).Name == "tag" && t.Field(1).Name == "payload1" {
			if v.Field(0).Int() != 1 { return "Ninguno" }
			return "Alguno(" + _hsformato(v.Field(1), true, depth+1) + ")"
		}
		if t.Name() != "" && t.Field(0).Name != "tag" {
			items := make([]string, 0, t.NumField())
			for i := 0; i < t.NumField(); i++ {
				name := []rune(t.Field(i).Name)
				name[0] = unicode.ToLower(name[0])
				items = append(items, string(name)+": "+_hsformato(v.Field(i), true, depth+1))
			}
			name := t.Name()
			if i := strings.Index(name, "["); i >= 0 { name = name[:i] }
			return name + " {" + strings.Join(items, ", ") + "}"
		}
	}
	return fmt.Sprint(v)
}
`
