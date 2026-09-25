package compiler

import "testing"

func TestMultilineListLiterals(t *testing.T) {
	want, err := Compile("lista.cometa", []byte("var lista = [\"hola\"]\nfn inicio()\n\timprimir(lista[0])\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, literal := range []string{
		"[\n\t\"hola\",\n]",
		"[\n\t\"hola\"\n]",
		"[\n\t// comentario\n\n\t\"hola\", // comentario\n]",
		"[\"hola\",]",
	} {
		t.Run(literal, func(t *testing.T) {
			got, err := Compile("lista.cometa", []byte("var lista = "+literal+"\nfn inicio()\n\timprimir(lista[0])\n"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("unexpected list output:\n%s", got)
			}
		})
	}
}

func TestNestedMultilineListsRuntime(t *testing.T) {
	runCometa(t, "fn inicio()\n\tvar listas = [\n\t\t[\n\t\t\t\"hola\",\n\t\t\t\"mundo\",\n\t\t],\n\t\t[\"otra\"],\n\t]\n\tvar vacia [cadena] = [\n\t]\n\timprimir(listas[0][1])\n\timprimir(listas[1][0])\n\timprimir(vacia.longitud())\n", "mundo\notra\n0\n")
}

func TestInvalidMultilineLists(t *testing.T) {
	for _, literal := range []string{
		"[\n\t1\n\t2\n]",
		"[\n\t1,\n",
		"[\n1,\n]",
		"[\n\t1,,\n]",
	} {
		if _, err := Compile("lista.cometa", []byte("var lista = "+literal+"\n")); err == nil {
			t.Errorf("expected error for %q", literal)
		}
	}
}
