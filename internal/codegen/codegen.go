package codegen

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/sema"
	"github.com/crsolver/cometa/internal/stdlib"
)

type generator struct {
	receiver       string
	returnType     sema.Type
	buffer         bytes.Buffer
	model          *sema.Model
	names          *goNames
	nextName       int
	loopLabel      string
	scopes         []string
	loopScopeDepth int
	defaultFlags   map[*ast.FuncDecl][]string
	lineDirectives bool
	displayPath    func(string) string
	position       ast.Pos // Cometa statement being lowered, for //line directives
	directiveDone  bool    // the directive for the next line was already written
	printing       bool    // imprimir is used, so the print runtime is emitted
}

// Options tunes code generation without changing the language semantics.
type Options struct {
	// LineDirectives emits //line comments so Go build errors and runtime
	// panics report Cometa file and line numbers.
	LineDirectives bool
	// DisplayPath maps the canonical path of a source file to the one written
	// in those directives. Nil keeps the canonical path.
	DisplayPath func(string) string
}

func Generate(filename string, program *ast.Program, model *sema.Model) ([]byte, error) {
	return GenerateWithOptions(filename, program, model, Options{})
}

func GenerateWithOptions(filename string, program *ast.Program, model *sema.Model, options Options) ([]byte, error) {

	g := &generator{model: model, names: newGoNames(model), defaultFlags: map[*ast.FuncDecl][]string{}, lineDirectives: options.LineDirectives, displayPath: options.DisplayPath}
	g.printing = usesPrint(program) || model.FormatsValues
	// What the built-in calls of the program need from Go and from the runtime.
	stringFeatures, sorts, sortsByKey, converts := false, false, false, false
	for _, builtin := range model.Builtins {
		switch builtin.Kind {
		case sema.StringMethod, sema.NumberMethod:
			stringFeatures = true
		case sema.ListMethod:
			sorts = sorts || builtin.Name == "ordenar" || builtin.Name == "ordenar_por"
			sortsByKey = sortsByKey || builtin.Name == "ordenar_por"
		case sema.Conversion:
			converts = true
		}
	}
	for expr := range model.ExprTypes {
		if _, ok := expr.(*ast.InterpolatedStringExpr); ok {
			stringFeatures = true
			break
		}
	}
	register := func(decl *ast.FuncDecl) {
		for _, param := range decl.Params {
			if param.Default != nil {
				g.defaultFlags[decl] = append(g.defaultFlags[decl], g.freshName())
			}
		}
	}
	for _, decl := range program.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			register(d)
		case *ast.TypeDecl:
			for _, method := range d.Methods {
				register(method)
			}
		}
	}
	g.line(0, "package main")
	g.line(0, "")
	called := map[string]bool{}
	for _, f := range model.Stdlib.Calls {
		called[f.Namespace+"."+f.Name] = true
	}
	runtimeSource, runtimeImports, err := stdlib.RuntimeFor(model.Stdlib.Modules, called)
	if err != nil {
		return nil, err
	}
	imports := map[string]bool{}
	for _, path := range runtimeImports {
		imports[path] = true
	}
	if g.printing {
		for _, path := range []string{"fmt", "reflect", "sort", "strconv", "strings", "unicode"} {
			imports[`"`+path+`"`] = true
		}
	}
	if stringFeatures {
		imports[`"strconv"`] = true
		imports[`"strings"`] = true
	}
	if sorts {
		imports[`"sort"`] = true
	}
	if sortsByKey {
		imports[`"fmt"`] = true
	}
	var paths []string
	for path := range imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) == 1 {
		g.line(0, "import %s", paths[0])
		g.line(0, "")
	} else if len(paths) > 1 {
		g.line(0, "import (")
		for _, path := range paths {
			g.line(1, "%s", path)
		}
		g.line(0, ")")
		g.line(0, "")
	}
	if runtimeSource != "" {
		g.write("%s\n", runtimeSource)
		for _, module := range model.Stdlib.Modules {
			if module == "std/pincel/datos" {
				// Saved data belongs to the project, not to the window title,
				// so it can be read before pincel.ejecutar runs.
				g.write("var _hgdatosCarpeta = %q\n\n", dataFolder(filename))
				break
			}
		}
		var nativeNames []string
		for name := range model.Types {
			if native, ok := stdlib.LookupType(name); ok && native.Wrapper != "" {
				continue // an alias of a Go type cannot receive marker methods
			}
			if native := stdlib.GoType(name); native != "" && strings.Contains(runtimeSource, "type "+strings.TrimPrefix(native, "*")+" ") {
				nativeNames = append(nativeNames, name)
			}
		}
		sort.Strings(nativeNames)
		for _, name := range nativeNames {
			g.emitVisibilityMethods(sema.Type{Kind: sema.Named, Name: name})
		}
	}
	if stringFeatures {
		g.write("%s\n", stringRuntime)
	}
	if g.printing {
		g.write("%s\n", printRuntime)
	}
	if converts || model.Stdlib.Used {
		g.write("%s\n", numericRuntime)
	}

	for _, decl := range program.Decls {
		if d, ok := decl.(*ast.InterfaceDecl); ok {
			g.emitInterface(d)
		}
		if enumDecl, ok := decl.(*ast.EnumDecl); ok {
			g.emitEnum(enumDecl)
		}
		if typeDecl, ok := decl.(*ast.TypeDecl); ok {
			g.emitType(typeDecl)
		}
	}
	for _, decl := range program.Decls {
		if global, ok := decl.(*ast.GlobalDecl); ok {
			g.emitGlobal(global)
		}
	}
	for _, decl := range program.Decls {
		switch declaration := decl.(type) {
		case *ast.TypeDecl:
			for _, method := range declaration.Methods {
				g.emitFunction(method)
			}
		case *ast.FuncDecl:
			g.emitFunction(declaration)
		}
	}

	formatted, err := eliminateUnusedLocals(resolveLocations(g.buffer.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("no se pudo formatear el Go generado: %w\n%s", err, g.buffer.String())
	}
	if SelfCheck && !strings.Contains(strings.Join(runtimeImports, " "), "github.com/") {
		if err = Validate(filename, formatted); err != nil {
			return nil, err
		}
	}
	return formatted, nil
}

func (g *generator) emitGlobal(decl *ast.GlobalDecl) {
	info := g.model.Globals[decl.Name]
	if decl.Constant {
		g.line(0, "const %s %s = %s", g.globalName(decl.Name), goType(info.Type), g.constExpr(decl.Value))
		g.line(0, "")
		return
	}
	child := &generator{model: g.model, names: g.names, nextName: g.nextName, defaultFlags: g.defaultFlags, lineDirectives: g.lineDirectives, displayPath: g.displayPath, position: g.position}
	child.line(0, "func() %s {", goType(info.Type))
	value := child.flowExpr(decl.Value, 1)
	child.line(1, "return %s", value)
	child.line(0, "}()")
	g.nextName = child.nextName
	g.line(0, "var %s %s = %s", g.globalName(decl.Name), goType(info.Type), strings.TrimSpace(child.buffer.String()))
	g.line(0, "")
}

func (g *generator) emitType(decl *ast.TypeDecl) {
	g.line(0, "type %s%s struct {", exported(decl.Name), g.typeParams(decl))
	info := g.model.Types[decl.Name]
	for _, field := range decl.Fields {
		if field.Embedded {
			g.line(1, "%s", goType(info.Fields[field.Name].Type))
			continue
		}
		g.line(1, "%s %s", exported(field.Name), goType(info.Fields[field.Name].Type))
	}
	g.line(0, "}")
	g.line(0, "")
	g.emitVisibilityMethods(sema.Type{Kind: sema.Named, Name: decl.Name, Args: g.model.TypeParams[decl]})
}

func (g *generator) emitFunction(decl *ast.FuncDecl) {
	g.receiver = decl.Receiver
	var signature sema.FuncInfo
	if decl.Receiver != "" {
		signature = g.model.Types[decl.Receiver].Methods[decl.Name]
		name := exported(decl.Name)
		if len(g.defaultFlags[decl]) > 0 {
			g.emitDefaultBridge(decl, signature)
			name = defaultMethod(decl)
		}
		g.write("func (_self *%s%s) %s(", exported(decl.Receiver), typeArguments(g.model.TypeParams[g.model.Types[decl.Receiver].Decl]), name)
	} else {
		signature = g.model.Functions[decl.Name]
		g.write("func %s%s(", g.funcName(decl.Name), g.typeParams(decl))
	}
	for _, flag := range g.defaultFlags[decl] {
		g.write("%s bool, ", flag)
	}
	for index, param := range decl.Params {
		if index > 0 {
			g.write(", ")
		}
		paramType := goType(signature.Params[index])
		if param.Variadic {
			paramType = "..." + goType(*signature.Params[index].Elem)
		}
		g.write("%s %s", localName(param.Name), paramType)
	}
	g.write(")")
	if signature.Return.Kind != sema.Void {
		g.write(" %s", goType(signature.Return))
	}
	g.write(" {\n")
	g.returnType = signature.Return
	flagIndex := 0
	for _, param := range decl.Params {
		if param.Default != nil {
			g.line(1, "if %s {", g.defaultFlags[decl][flagIndex])
			g.line(2, "%s = %s", localName(param.Name), g.flowExpr(param.Default, 2))
			g.line(1, "}")
			flagIndex++
		}
	}
	target := ""
	if signature.Return.Kind != sema.Void {
		target = "return"
	}
	g.flowBlock(decl.Body, 1, target)
	g.line(0, "}")
	g.line(0, "")
}

// directive maps the following generated lines back to a Cometa source line.
func (g *generator) directive(pos ast.Pos) {
	if !g.lineDirectives || pos.Filename == "" || pos.Line <= 0 {
		return
	}
	g.position = pos
	g.writeDirective()
	g.directiveDone = true
}

func (g *generator) writeDirective() {
	fmt.Fprintf(&g.buffer, "//line %s:%d\n", g.file(g.position), g.position.Line)
}

// file is the path of pos as line directives spell it.
func (g *generator) file(pos ast.Pos) string {
	if g.displayPath != nil {
		return g.displayPath(pos.Filename)
	}
	return pos.Filename
}

// located marks an expression that can fail at run time (an index or a
// division) with its column. A panic trace only carries file and line, so the
// column travels in the file name of an inline directive, `juego.cometa#17:4`,
// which cmd/cometa reads back. The parentheses keep gofmt from moving a
// directive that starts a statement onto its own line.
//
// What follows the expression must return to the enclosing position, which
// is only known later: the mark left after value names the statement, and an
// enclosing located expression rewrites the marks inside it to itself.
// resolveLocations turns the remaining ones into directives.
func (g *generator) located(pos ast.Pos, value string) string {
	if !g.lineDirectives || value == "" || pos.Filename == "" || pos.Line <= 0 || pos.Column <= 0 || g.position.Line <= 0 {
		return value
	}
	here := fmt.Sprintf("/*line %s#%d:%d*/", g.file(pos), pos.Column, pos.Line)
	value = pendingLocation.ReplaceAllLiteralString(value, here)
	return fmt.Sprintf("(%s%s\x00%s:%d\x01)", here, value, g.file(g.position), g.position.Line)
}

var pendingLocation = regexp.MustCompile("\x00[^\x01]*\x01")

func resolveLocations(source []byte) []byte {
	return pendingLocation.ReplaceAllFunc(source, func(mark []byte) []byte {
		return []byte("/*line " + string(mark[1:len(mark)-1]) + "*/")
	})
}

// emitStringRepeat opens a loop over the code points of a string: the element
// is a one-character string and the index counts characters, not bytes.
func (g *generator) emitStringRepeat(stmt *ast.RepeatStmt, iterable string, indent int) {
	index, letter := "_", g.freshName()
	if stmt.Index != "" {
		index = g.freshName()
	}
	g.line(indent, "for %s, %s := range []rune(%s) {", index, letter, iterable)
	if stmt.Element == "_" {
		g.line(indent+1, "_ = %s", letter)
	} else {
		g.line(indent+1, "%s := string(%s)", localName(stmt.Element), letter)
	}
	if stmt.Index != "" {
		g.line(indent+1, "%s := int64(%s)", localName(stmt.Index), index)
	}
}

// constExpr writes an expression that sema accepted as constant: the value of
// a `const` or a label of `casos`. Go needs these as constant expressions, so
// they cannot go through flow lowering, which stores operands in variables.
func (g *generator) constExpr(expr ast.Expr) string {
	value := g.rawConstExpr(expr)
	if _, ok := g.model.NumericCoercions[expr]; ok {
		return "float64(" + value + ")"
	}
	return value
}

func (g *generator) rawConstExpr(expr ast.Expr) string {
	if value, ok := g.model.Stdlib.Constants[expr]; ok {
		return value
	}
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return literal(e)
	case *ast.IdentExpr:
		return g.globalName(e.Name)
	case *ast.UnaryExpr:
		return "(" + e.Operator + g.constExpr(e.Value) + ")"
	case *ast.BinaryExpr:
		binary := "(" + g.constExpr(e.Left) + " " + e.Operator + " " + g.constExpr(e.Right) + ")"
		if e.Operator == "/" || e.Operator == "%" {
			return g.located(e.Pos, binary)
		}
		return binary
	}
	panic(fmt.Sprintf("unsupported constant expression %T", expr))
}

// literal writes a Cometa literal in Go.
func literal(e *ast.LiteralExpr) string {
	if e.Kind == "bool" {
		if e.Value == "verdadero" {
			return "true"
		}
		return "false"
	}
	return e.Value
}

func (g *generator) interpolated(expression *ast.InterpolatedStringExpr, emit func(ast.Expr) string) string {
	parts := make([]string, 0, len(expression.Parts))
	for _, part := range expression.Parts {
		if part.Expr == nil {
			parts = append(parts, strconv.Quote(part.Text))
			continue
		}
		value := emit(part.Expr)
		if value == "" {
			return ""
		}
		switch g.model.ExprTypes[part.Expr].Kind {
		case sema.Decimal:
			value = "_hsnumero(" + value + ")"
		case sema.Integer:
			value = "strconv.FormatInt(" + value + ", 10)"
		case sema.Boolean:
			value = "_hsbool(" + value + ")"
		case sema.String, sema.Never:
		default:
			value = "_hsformato(reflect.ValueOf(" + value + "), false, 0)"
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return `""`
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

func goType(t sema.Type) string {
	if name := stdlib.GoInterface(t.Name); name != "" && t.Kind == sema.Interface {
		return name
	}
	if name := stdlib.GoType(t.Name); name != "" && t.Kind == sema.Named {
		if stdlib.IsValue(t.Name) {
			return name
		}
		return "*" + name
	}
	switch t.Kind {
	case sema.Void, sema.Never:
		return "struct{}"
	case sema.Optional, sema.Result:
		fields := "struct { tag int"
		if t.Elem.Kind != sema.Void {
			fields += "; payload1 " + goType(*t.Elem)
		}
		if t.Kind == sema.Result {
			fields += "; payload2 " + goType(*t.Err)
		}
		return fields + " }"
	case sema.Decimal:
		return "float64"
	case sema.Integer:
		return "int64"
	case sema.String:
		return "string"
	case sema.Boolean:
		return "bool"
	case sema.TypeParameter:
		return "_T_" + t.Name
	case sema.Interface:
		if t.Name == "" {
			return "any"
		}
		return namedGoType(t)
	case sema.Named, sema.Enum:
		return "*" + namedGoType(t)
	case sema.Slice:
		return "[]" + goType(*t.Elem)
	case sema.Map:
		return "map[" + goType(*t.Key) + "]" + goType(*t.Elem)
	default:
		panic("invalid Go type")
	}
}

func usesPrint(program *ast.Program) bool {
	found := false
	ast.Inspect(program, func(node any) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if ident, ok := call.Callee.(*ast.IdentExpr); ok && ident.Name == "imprimir" {
				found = true
			}
		}
		return !found
	})
	return found
}

// SelfCheck makes Generate type-check its own output with Validate. The test
// suites turn it on, so every program they compile also proves that the Go
// generated for it is valid. It stays off otherwise: the check runs the Go
// toolchain (about a second per program) and `go build` repeats it anyway.
// Programs that import Ebitengine are never checked here.
var SelfCheck bool

// Validate type-checks generated Go that only imports the standard library.
// A failure is a defect of the compiler, never of the Cometa program.
func Validate(filename string, source []byte) error {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, filename+".go", source, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("error interno del compilador (por favor reporta este error): el Go generado no es válido: %w", err)
	}
	var typeErrors []string
	config := types.Config{
		Importer: importer.Default(),
		Error:    func(err error) { typeErrors = append(typeErrors, err.Error()) },
	}
	if _, err = config.Check("main", set, []*goast.File{file}, nil); err != nil {
		return fmt.Errorf("error interno del compilador (por favor reporta este error): el Go generado no pasa la verificación de tipos: %s", strings.Join(typeErrors, "; "))
	}
	return nil
}

func (g *generator) write(format string, args ...any) {
	fmt.Fprintf(&g.buffer, format, args...)
}

func (g *generator) line(indent int, format string, args ...any) {
	// Go advances a //line directive by one per generated line, so a statement
	// lowered to several lines needs the directive repeated to keep its line.
	if g.lineDirectives && g.position.Line > 0 && indent > 0 && !g.directiveDone {
		g.writeDirective()
	}
	g.directiveDone = false
	g.buffer.WriteString(strings.Repeat("\t", indent))
	fmt.Fprintf(&g.buffer, format, args...)
	g.buffer.WriteByte('\n')
}
