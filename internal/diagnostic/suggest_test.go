package diagnostic

import "testing"

func TestSuggest(t *testing.T) {
	names := []string{"edad", "nombre", "puntos", "\x00oculto"}
	cases := map[string]string{
		"edda":   "edad",
		"nombr":  "nombre",
		"puntso": "puntos",
		"xyz":    "",
		"edad":   "",
	}
	for input, want := range cases {
		if got := Suggest(input, names); got != want {
			t.Errorf("Suggest(%q) = %q, quiero %q", input, got, want)
		}
	}
	if got := Hint("edda", names); got != "; ¿quisiste decir \"edad\"?" {
		t.Errorf("Hint = %q", got)
	}
}
