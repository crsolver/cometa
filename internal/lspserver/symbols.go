package lspserver

import (
	"context"
	"strings"

	"github.com/owenrumney/go-lsp/lsp"

	"github.com/crsolver/cometa/internal/ast"
)

func (h *Handler) DocumentSymbol(_ context.Context, params *lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	p, _ := h.project(params.TextDocument.URI, nil)
	if p == nil || p.Root == nil || p.Root.Program == nil {
		return nil, nil
	}
	// The module as written: the analyzed program also holds its dependencies.
	program, model := p.Root.Program, p.Model
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var symbols []lsp.DocumentSymbol
	for _, decl := range program.Decls {
		switch declaration := decl.(type) {
		case *ast.InterfaceDecl:
			symbol := symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindInterface, "interfaz"+typeParamDetail(declaration.TypeParams))
			for _, method := range declaration.Methods {
				symbol.Children = append(symbol.Children, symbolFor(lines, method.Name, method.Pos, lsp.SymbolKindMethod, functionDetail(method)))
			}
			symbols = append(symbols, symbol)
		case *ast.EnumDecl:
			symbol := symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindEnum, "enum"+typeParamDetail(declaration.TypeParams))
			for _, variant := range declaration.Variants {
				detail := "variante"
				if variant.Payload != nil {
					detail = typeRefString(*variant.Payload)
				}
				symbol.Children = append(symbol.Children, symbolFor(lines, variant.Name, variant.Pos, lsp.SymbolKindEnumMember, detail))
			}
			symbols = append(symbols, symbol)
		case *ast.TypeDecl:
			symbol := symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindStruct, "tipo"+typeParamDetail(declaration.TypeParams))
			for _, field := range declaration.Fields {
				symbol.Children = append(symbol.Children, symbolFor(lines, field.Name, field.Pos, lsp.SymbolKindField, typeRefString(field.Type)))
			}
			for _, method := range declaration.Methods {
				symbol.Children = append(symbol.Children, symbolFor(lines, method.Name, method.Pos, lsp.SymbolKindMethod, functionDetail(method)))
			}
			symbols = append(symbols, symbol)
		case *ast.FuncDecl:
			symbols = append(symbols, symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindFunction, functionDetail(declaration)))
		case *ast.GlobalDecl:
			kind := lsp.SymbolKindVariable
			detail := "var"
			if declaration.Constant {
				kind = lsp.SymbolKindConstant
				detail = "const"
			}
			if model != nil && model.Globals[declaration.Name] != nil {
				detail += " " + model.Globals[declaration.Name].Type.String()
			}
			symbols = append(symbols, symbolFor(lines, declaration.Name, declaration.Pos, kind, detail))
		}
	}
	return symbols, nil
}

func symbolFor(lines []string, name string, position ast.Pos, kind lsp.SymbolKind, detail string) lsp.DocumentSymbol {
	lineIndex := max(position.Line-1, 0)
	lineLength := 0
	nameStart := max(position.Column-1, 0)
	if lineIndex < len(lines) {
		lineLength = utf16Length(lines[lineIndex])
		if byteIndex := strings.Index(lines[lineIndex], name); byteIndex >= 0 {
			nameStart = utf16Length(lines[lineIndex][:byteIndex])
		}
	}
	wholeLine := lsp.Range{
		Start: lsp.Position{Line: lineIndex, Character: 0},
		End:   lsp.Position{Line: lineIndex, Character: lineLength},
	}
	selection := lsp.Range{
		Start: lsp.Position{Line: lineIndex, Character: nameStart},
		End:   lsp.Position{Line: lineIndex, Character: nameStart + utf16Length(name)},
	}
	return lsp.DocumentSymbol{Name: name, Detail: detail, Kind: kind, Range: wholeLine, SelectionRange: selection}
}

func typeRefString(ref ast.TypeRef) string {
	if ref.Wrapper != "" {
		payload := typeRefString(*ref.Payload)
		if ref.Payload.Name == "$unidad" {
			payload = ""
		}
		if ref.Payload.Wrapper == "!" {
			payload = "(" + payload + ")"
		}
		if ref.Wrapper == "?" {
			return payload + "?"
		}
		if ref.ErrorType.Name == "cadena" && ref.ErrorType.Wrapper == "" {
			return payload + "!"
		}
		errorType := typeRefString(*ref.ErrorType)
		if ref.ErrorType.Wrapper != "" {
			errorType = "(" + errorType + ")"
		}
		return payload + "!" + errorType
	}
	if ref.Element != nil {
		if ref.Key != nil {
			return "[" + typeRefString(*ref.Key) + ": " + typeRefString(*ref.Element) + "]"
		}
		return "[" + typeRefString(*ref.Element) + "]"
	}
	name := ref.Name
	if len(ref.Args) > 0 {
		var args []string
		for _, arg := range ref.Args {
			args = append(args, typeRefString(arg))
		}
		name += "<" + strings.Join(args, ", ") + ">"
	}
	return name
}

func functionDetail(function *ast.FuncDecl) string {
	var params []string
	for _, param := range function.Params {
		prefix := ""
		if param.Variadic {
			prefix = "..."
		}
		detail := param.Name + " " + prefix + typeRefString(param.Type)
		if param.Default != nil {
			detail += " = " + defaultExprSource(param.Default)
		}
		params = append(params, detail)
	}
	detail := "fn " + function.Name + typeParamDetail(function.TypeParams) + "(" + strings.Join(params, ", ") + ")"
	if function.Public {
		detail = "pub " + detail
	}
	if function.ReturnType != nil {
		detail += " " + typeRefString(*function.ReturnType)
	}
	return detail
}
