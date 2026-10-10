package stdlib

import (
	"strings"
	"testing"

	"github.com/crsolver/cometa/internal/ast"
)

func TestCatalogDeclarations(t *testing.T) {
	goNames := map[string]bool{}
	for _, f := range Functions {
		d := f.Declaration()
		if !d.Public || d.Name != FunctionSymbol(f.Namespace, f.Name) || goNames[f.GoName] {
			t.Fatalf("invalid or duplicate intrinsic: %+v", f)
		}
		goNames[f.GoName] = true
	}
	for _, m := range Modules {
		for i := range m.Types {
			typ := &m.Types[i]
			if typ.Interface {
				continue
			}
			d := typ.Declaration()
			if !d.Public || d.Name != typ.Symbol() || len(d.Fields) != len(typ.Fields) {
				t.Fatal(typ.Name)
			}
			for _, field := range d.Fields {
				if !field.Public {
					t.Fatalf("private native field: %s.%s", typ.Name, field.Name)
				}
			}
			for _, method := range typ.Methods {
				if d := method.Declaration(); d.Name != method.Name || d.Receiver != typ.Symbol() {
					t.Fatalf("method %s.%s declared as %s of %s", typ.Name, method.Name, d.Name, d.Receiver)
				}
			}
		}
	}
}

// Every declaration must point at its own line of the reference source: the
// editor opens that place for "go to definition" and reads the comment above
// it as documentation.
func TestDeclarationsPointAtTheReference(t *testing.T) {
	check := func(m *Module, what string, pos ast.Pos, want string) {
		t.Helper()
		if pos.Filename != m.URI() {
			t.Errorf("%s: file %q, want %q", what, pos.Filename, m.URI())
			return
		}
		lines := strings.Split(m.reference().text, "\n")
		if pos.Line < 1 || pos.Line > len(lines) || pos.Column < 1 || pos.Column > len(lines[pos.Line-1])+1 {
			t.Errorf("%s: position %v is outside the reference", what, pos)
			return
		}
		if got := lines[pos.Line-1][pos.Column-1:]; !strings.HasPrefix(got, want) {
			t.Errorf("%s: position %v holds %q, want %q", what, pos, got, want)
		}
	}
	for _, m := range Modules {
		for _, f := range m.Functions {
			d := f.Declaration()
			check(m, m.Name+"."+f.Name, d.NamePos, f.Name+"(")
			for _, p := range d.Params {
				check(m, m.Name+"."+f.Name+" "+p.Name, p.Pos, p.Name+" ")
			}
		}
		for i := range m.Types {
			typ := &m.Types[i]
			if len(typ.Fields) > 0 {
				d := typ.Declaration()
				check(m, typ.Name, d.NamePos, typ.Name)
				for _, field := range d.Fields {
					check(m, typ.Name+"."+field.Name, field.Pos, field.Name+" ")
				}
			}
			// Two types of a module may share a method name; each one has its own line.
			var previous ast.Pos
			for _, method := range typ.Methods {
				d := method.Declaration()
				check(m, typ.Name+"."+method.Name, d.NamePos, method.Name+"(")
				if d.NamePos.Line <= previous.Line {
					t.Errorf("%s.%s is declared at line %d, before the previous method", typ.Name, method.Name, d.NamePos.Line)
				}
				previous = d.NamePos
			}
		}
	}
}

// The constants of a type are listed in its own comment, which is what the
// editor shows for it; they must not end up above the next declaration.
func TestReferenceListsConstantsWithTheirType(t *testing.T) {
	source, ok := Source("std/pincel/retro")
	if !ok {
		t.Fatal("std/pincel/retro has no reference")
	}
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if line == "pub tipo Atlas" && lines[i-1] != "// Valores: .ASCII, .Dungeon" {
			t.Errorf("line above Atlas = %q", lines[i-1])
		}
	}
	for _, m := range Modules {
		lines := strings.Split(m.reference().text, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "// Valores:") && !strings.HasPrefix(lines[i+1], "pub tipo ") {
				t.Errorf("%s: constants are followed by %q, not by their type", m.Name, lines[i+1])
			}
		}
	}
}

func TestLookups(t *testing.T) {
	if got := QualifiedSignature("(pos Vec2, tinte Color = .Blanco) [Imagen]"); got != "(pos mate.Vec2, tinte color.Color = .Blanco) [graficos.Imagen]" {
		t.Errorf("QualifiedSignature = %q", got)
	}
	// A function symbol can start like a type or another function: the longest wins.
	if got := Display("__std_Vec2 __std_mate_pi __std_graficos_limpiar __std_ui_texto __std_Juego"); got != "mate.Vec2 mate.pi graficos.limpiar ui.texto pincel.Juego" {
		t.Errorf("Display = %q", got)
	}
	for symbol, want := range map[string][3]bool{
		Symbol("Vec2"):   {true, true, false},
		Symbol("Imagen"): {true, false, true},
		Symbol("Juego"):  {false, false, false},
		"Vec2":           {false, false, false},
	} {
		if got := [3]bool{IsType(symbol), IsValue(symbol), IsOpaque(symbol)}; got != want {
			t.Errorf("%s: type, value, opaque = %v, want %v", symbol, got, want)
		}
	}
	if GoType(Symbol("Rejilla")) != "_hgRejilla" || GoType(Symbol("Juego")) != "" {
		t.Error("GoType")
	}
	seen := map[string]bool{}
	for _, m := range Modules {
		if seen[m.Path] || seen[m.Name] {
			t.Errorf("module %s (%s) is listed twice", m.Name, m.Path)
		}
		seen[m.Path], seen[m.Name] = true, true
		if found, ok := LookupModule(m.Path); !ok || found != m {
			t.Errorf("LookupModule(%q)", m.Path)
		}
	}
}
