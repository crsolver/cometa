package main

import (
	"github.com/crsolver/cometa/internal/stdlib"
	"testing"
)

func TestBuildCoversCatalog(t *testing.T) {
	modules := Build()
	count := 0
	for _, m := range modules {
		if len(m.Funciones) == 0 {
			t.Errorf("módulo %s sin funciones", m.Nombre)
		}
		if m.Ruta == "" {
			t.Errorf("módulo %s sin ruta", m.Nombre)
		}
		count += len(m.Funciones)
	}
	if count != len(stdlib.Functions) {
		t.Fatalf("funciones documentadas = %d, catálogo = %d", count, len(stdlib.Functions))
	}
	types := 0
	for _, m := range modules {
		types += len(m.Tipos)
	}
	if types == 0 {
		t.Fatal("no se generó ningún tipo")
	}
}
