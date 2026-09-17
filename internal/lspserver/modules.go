package lspserver

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/owenrumney/go-lsp/lsp"
	"hacha/internal/ast"
	"hacha/internal/compiler"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"hacha/internal/sema"
)

func hasImports(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "usar ") {
			return true
		}
	}
	return false
}

func pathFromURI(uri lsp.DocumentURI) (string, error) {
	u, err := url.Parse(string(uri))
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("se requiere una URI file")
	}
	path := u.Path
	if runtime.GOOS == "windows" && len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	if u.Host != "" && u.Host != "localhost" {
		path = "//" + u.Host + path
	}
	return compiler.CanonicalPath(filepath.FromSlash(path))
}

func fileURI(path string) lsp.DocumentURI {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return lsp.DocumentURI((&url.URL{Scheme: "file", Path: path}).String())
}

// File-backed analysis must resolve resources against a decoded filesystem
// path, not the editor's URI spelling (which may encode the drive colon).
func analysisFilename(uri lsp.DocumentURI) string {
	if path, err := pathFromURI(uri); err == nil {
		return path
	}
	return string(uri)
}

func (h *Handler) sourceLoader() (compiler.SourceLoader, map[string]lsp.DocumentURI) {
	sources := map[string][]byte{}
	uris := map[string]lsp.DocumentURI{}
	for _, uri := range h.documents.URIs() {
		if path, err := pathFromURI(uri); err == nil {
			if text, ok := h.documents.Text(uri); ok {
				sources[path] = []byte(text)
				uris[path] = uri
			}
		}
	}
	return func(path string) ([]byte, error) {
		if source, ok := sources[path]; ok {
			return source, nil
		}
		return os.ReadFile(path)
	}, uris
}

func (h *Handler) project(uri lsp.DocumentURI, override *string) (*compiler.Project, error) {
	path, err := pathFromURI(uri)
	if err != nil {
		return nil, err
	}
	loader, _ := h.sourceLoader()
	if override != nil {
		base := loader
		loader = func(name string) ([]byte, error) {
			if name == path {
				return []byte(*override), nil
			}
			return base(name)
		}
	}
	return compiler.AnalyzeProject(path, loader)
}

func errorFile(err error) string {
	switch e := err.(type) {
	case *lexer.Error:
		return e.Filename
	case *parser.Error:
		return e.Filename
	case *sema.Error:
		return e.Filename
	}
	return ""
}

func (h *Handler) DidChangeWatchedFiles(ctx context.Context, params *lsp.DidChangeWatchedFilesParams) error {
	for _, change := range params.Changes {
		if err := h.publishProjectDiagnostics(ctx, change.URI); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) publishProjectDiagnostics(ctx context.Context, changed lsp.DocumentURI) error {
	if h.client == nil {
		return fmt.Errorf("el cliente LSP no está conectado")
	}
	h.projectMu.Lock()
	defer h.projectMu.Unlock()
	if h.projects == nil {
		h.projects = map[lsp.DocumentURI]*compiler.Project{}
		h.reports = map[lsp.DocumentURI]map[lsp.DocumentURI][]lsp.Diagnostic{}
		h.published = map[lsp.DocumentURI]bool{}
	}
	loader, uris := h.sourceLoader()
	changedPath, _ := pathFromURI(changed)
	open := map[lsp.DocumentURI]bool{}
	for _, uri := range h.documents.URIs() {
		open[uri] = true
	}
	for uri := range h.reports {
		if !open[uri] {
			delete(h.reports, uri)
			delete(h.projects, uri)
		}
	}
	for _, uri := range h.documents.URIs() {
		old := h.projects[uri]
		_, exists := h.reports[uri]
		affected := !exists || uri == changed || old != nil && (old.Modules[changedPath] != nil || len(old.Reverse[changedPath]) > 0)
		if !affected {
			continue
		}
		text, _ := h.documents.Text(uri)
		report := map[lsp.DocumentURI][]lsp.Diagnostic{uri: {}}
		var analysisErr error
		if hasImports(text) {
			p, err := h.project(uri, nil)
			h.projects[uri], analysisErr = p, err
		} else {
			delete(h.projects, uri)
			_, _, analysisErr = compiler.Analyze(analysisFilename(uri), []byte(text))
		}
		if analysisErr != nil {
			destination := uri
			if path := errorFile(analysisErr); filepath.IsAbs(path) {
				if original, ok := uris[path]; ok {
					destination = original
				} else {
					destination = fileURI(path)
				}
				if data, err := loader(path); err == nil {
					text = string(data)
				}
			}
			report[destination] = append(report[destination], diagnosticFromError(analysisErr, text))
		}
		h.reports[uri] = report
	}
	aggregate := map[lsp.DocumentURI][]lsp.Diagnostic{changed: {}}
	for uri := range h.published {
		aggregate[uri] = []lsp.Diagnostic{}
	}
	seen := map[string]bool{}
	for _, report := range h.reports {
		for uri, diagnostics := range report {
			if _, ok := aggregate[uri]; !ok {
				aggregate[uri] = []lsp.Diagnostic{}
			}
			for _, diagnostic := range diagnostics {
				key := fmt.Sprintf("%s:%v", uri, diagnostic.Range)
				if !seen[key] {
					seen[key] = true
					aggregate[uri] = append(aggregate[uri], diagnostic)
				}
			}
		}
	}
	var destinations []string
	for uri := range aggregate {
		destinations = append(destinations, string(uri))
	}
	sort.Strings(destinations)
	for _, raw := range destinations {
		uri := lsp.DocumentURI(raw)
		diagnostics := aggregate[uri]
		sort.Slice(diagnostics, func(i, j int) bool {
			a, b := diagnostics[i], diagnostics[j]
			if a.Range.Start.Line != b.Range.Start.Line {
				return a.Range.Start.Line < b.Range.Start.Line
			}
			if a.Range.Start.Character != b.Range.Start.Character {
				return a.Range.Start.Character < b.Range.Start.Character
			}
			return a.Message < b.Message
		})
		params := &lsp.PublishDiagnosticsParams{URI: uri, Diagnostics: diagnostics}
		if version, exists := h.documents.Version(uri); exists {
			params.Version = &version
		}
		if err := h.client.PublishDiagnostics(ctx, params); err != nil {
			return err
		}
		h.published[uri] = true
	}
	return nil
}

func declarationDetail(d *compiler.Declaration) string {
	switch node := d.Node.(type) {
	case *ast.FuncDecl:
		return functionDetail(node)
	case *ast.TypeDecl:
		return "tipo " + node.Name + typeParamDetail(node.TypeParams)
	case *ast.EnumDecl:
		return "enum " + node.Name + typeParamDetail(node.TypeParams)
	case *ast.InterfaceDecl:
		return "interfaz " + node.Name + typeParamDetail(node.TypeParams)
	case *ast.GlobalDecl:
		keyword := "var"
		if node.Constant {
			keyword = "const"
		}
		if node.Type != nil {
			return keyword + " " + node.Name + " " + typeRefString(*node.Type)
		}
		return keyword + " " + node.Name
	}
	return d.Name
}

func (h *Handler) moduleHover(params *lsp.HoverParams, text string) *lsp.Hover {
	p, _ := h.project(params.TextDocument.URI, nil)
	if p == nil || p.Root == nil || p.Root.Program == nil {
		return nil
	}
	tokens, err := lexer.Lex(p.Root.Path, text)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	id, span, ok := identifierAt(tokens, lines, params.Position)
	if !ok {
		return nil
	}
	id.Pos.Filename = p.Root.Path
	detail, docs := "", ""
	if target := p.Root.Imports[id.Lexeme]; target != nil && p.References[id.Pos] != nil {
		return &lsp.Hover{Contents: lsp.NewHoverContents(lsp.Markdown, "módulo `"+id.Lexeme+"`"), Range: &span}
	}
	if d := p.References[id.Pos]; d != nil {
		detail = declarationDetail(d)
		if p.Model != nil {
			if global := p.Model.Globals[d.Symbol]; global != nil {
				detail = globalHover(d.Name, global.Type, global.Constant).detail
			}
			for call, info := range p.Model.Calls {
				if calleePosition(call.Callee) == id.Pos && info.Signature.Decl != nil {
					detail = signatureDetail(info.Signature)
					break
				}
			}
		}
		docs = documentationBefore(strings.Split(string(d.Module.Source), "\n"), d.Pos.Line)
	} else if p.Model != nil {
		if info, found := resolveHover(p.Root.Bound, p.Model, id.Pos, lines); found {
			detail, docs = info.detail, info.documentation
		}
	}
	if p.Model != nil {
		for call, info := range p.Model.Calls {
			if calleePosition(call.Callee) == id.Pos && info.Signature.Decl != nil {
				detail = signatureDetail(info.Signature)
				if module := p.Modules[info.Signature.Decl.Pos.Filename]; module != nil {
					docs = documentationBefore(strings.Split(string(module.Source), "\n"), info.Signature.Decl.NamePos.Line)
				}
			}
		}
	}
	if detail == "" {
		return nil
	}
	value := "```hacha\n" + p.Display(detail) + "\n```"
	if docs != "" {
		value += "\n\n" + docs
	}
	return &lsp.Hover{Contents: lsp.NewHoverContents(lsp.Markdown, value), Range: &span}
}

func (h *Handler) Definition(_ context.Context, params *lsp.DefinitionParams) ([]lsp.Location, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	p, _ := h.project(params.TextDocument.URI, nil)
	if p == nil || p.Root == nil || p.Root.Program == nil {
		return nil, nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, imp := range p.Root.Program.Imports {
		if imp.Pos.Line-1 == params.Position.Line {
			if target := p.Root.Imports[imp.Alias]; target != nil {
				return []lsp.Location{{URI: fileURI(target.Path), Range: lsp.Range{}}}, nil
			}
		}
	}
	tokens, err := lexer.Lex(p.Root.Path, text)
	if err != nil {
		return nil, nil
	}
	id, _, ok := identifierAt(tokens, lines, params.Position)
	if !ok {
		return nil, nil
	}
	id.Pos.Filename = p.Root.Path
	if target := p.Root.Imports[id.Lexeme]; target != nil && p.References[id.Pos] != nil {
		return []lsp.Location{{URI: fileURI(target.Path), Range: lsp.Range{}}}, nil
	}
	if d := p.References[id.Pos]; d != nil {
		return []lsp.Location{declarationLocation(d.Pos, string(d.Module.Source))}, nil
	}
	if p.Model != nil {
		for call, info := range p.Model.Calls {
			if calleePosition(call.Callee) == id.Pos && info.Signature.Decl != nil {
				return h.locations(p, info.Signature.Decl.NamePos), nil
			}
		}
		for node, constructor := range p.Model.Constructors {
			if call, ok := node.(*ast.CallExpr); ok {
				node = call.Callee
			}
			var position ast.Pos
			switch e := node.(type) {
			case *ast.MemberExpr:
				position = e.NamePos
			case *ast.ContextualVariantExpr:
				position = e.NamePos
			}
			if position == id.Pos && constructor.Variant.Decl != nil {
				return h.locations(p, constructor.Variant.Decl.Pos), nil
			}
		}
		for expr := range p.Model.ExprTypes {
			if match, ok := expr.(*ast.MatchExpr); ok {
				if enum := p.Model.EnumFor(p.Model.ExprTypes[match.Value]); enum != nil {
					for _, arm := range match.Arms {
						if arm.NamePos == id.Pos {
							if variant, exists := enum.Variants[arm.Pattern]; exists {
								return h.locations(p, variant.Decl.Pos), nil
							}
						}
					}
				}
			}
			if member, ok := expr.(*ast.MemberExpr); ok && member.NamePos == id.Pos {
				if enum := p.Model.EnumFor(p.Model.ExprTypes[member.Object]); enum != nil {
					if variant, found := enum.Variants[member.Name]; found {
						return h.locations(p, variant.Decl.Pos), nil
					}
				}
				m := p.Model.Members(p.Model.ExprTypes[member.Object])[member.Name]
				if m.Field.Decl != nil {
					return h.locations(p, m.Field.Decl.Pos), nil
				}
				if m.Method.Decl != nil {
					return h.locations(p, m.Method.Decl.NamePos), nil
				}
			}
		}
	}
	return nil, nil
}

func calleePosition(expr ast.Expr) ast.Pos {
	switch e := expr.(type) {
	case *ast.MemberExpr:
		return e.NamePos
	case *ast.ReceiverExpr:
		return e.NamePos
	case *ast.InstantiateExpr:
		return e.Pos
	}
	return expr.Position()
}

func (h *Handler) locations(p *compiler.Project, pos ast.Pos) []lsp.Location {
	if m := p.Modules[pos.Filename]; m != nil {
		return []lsp.Location{declarationLocation(pos, string(m.Source))}
	}
	return nil
}

func declarationLocation(pos ast.Pos, text string) lsp.Location {
	start, end := diagnosticRange(pos, text)
	return lsp.Location{URI: fileURI(pos.Filename), Range: lsp.Range{Start: start, End: end}}
}

var moduleSelector = regexp.MustCompile(`([\p{L}_][\p{L}\p{N}_]*(?:\.[\p{L}_][\p{L}\p{N}_]*)*(?:<[^\n]+>)?)\.[\p{L}\p{N}_]*$`)

func (h *Handler) moduleCompletion(params *lsp.CompletionParams, text string) *lsp.CompletionList {
	empty := &lsp.CompletionList{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	line := params.Position.Line
	if line < 0 || line >= len(lines) {
		return empty
	}
	prefix := utf16Prefix(lines[line], params.Position.Character)
	if !completionCodePosition(prefix) {
		return empty
	}
	match := moduleSelector.FindStringSubmatch(prefix)
	if len(match) == 0 {
		return h.moduleContextualCompletion(params, lines, prefix)
	}
	receiver := match[1]
	indent := lines[line][:len(lines[line])-len(strings.TrimLeft(lines[line], "\t"))]
	replacement := indent + receiver + ".__hacha_probe__"
	if arrow := strings.Index(prefix, "=>"); arrow >= 0 {
		replacement = prefix[:arrow+2] + " " + receiver + ".__hacha_probe__"
	}
	p := h.completionProject(params.TextDocument.URI, lines, line, replacement)
	if p != nil && p.Model != nil {
		var selected ast.Expr
		var resolved sema.Type
		for expr, t := range p.Model.ExprTypes {
			if expr.Position().Filename == p.Root.Path && expr.Position().Line == line+1 {
				if selected == nil || expr.Position().Column > selected.Position().Column {
					selected, resolved = expr, t
				}
			}
		}
		if selected != nil {
			if resolved.Kind == sema.Named || resolved.Kind == sema.Interface || resolved.Kind == sema.TypeParameter {
				return displayItems(p, resolvedMemberItems(p.Model, resolved))
			}
			if resolved.Kind == sema.Enum {
				if _, ok := selected.(*ast.InstantiateExpr); ok {
					return displayItems(p, enumCompletionItems(p.Model.EnumFor(resolved)))
				}
				if id, ok := selected.(*ast.IdentExpr); ok && p.Model.Enums[id.Name] != nil {
					return displayItems(p, enumCompletionItems(p.Model.EnumFor(resolved)))
				}
			}
			return empty
		}
	}
	// Namespace completion needs only the imports and dependency signatures, so it
	// remains available even when the current declaration is unfinished.
	header := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "usar ") {
			header += line + "\n"
		}
	}
	headerProject, _ := h.project(params.TextDocument.URI, &header)
	if headerProject == nil || headerProject.Root == nil {
		return empty
	}
	if target := headerProject.Root.Imports[receiver]; target != nil {
		var names []string
		for name := range target.Declarations {
			if name != "inicio" {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			d := target.Declarations[name]
			kind := lsp.CompletionItemKindClass
			switch d.Node.(type) {
			case *ast.FuncDecl:
				kind = lsp.CompletionItemKindFunction
			case *ast.EnumDecl:
				kind = lsp.CompletionItemKindEnum
			case *ast.InterfaceDecl:
				kind = lsp.CompletionItemKindInterface
			case *ast.GlobalDecl:
				kind = lsp.CompletionItemKindVariable
				if d.Node.(*ast.GlobalDecl).Constant {
					kind = lsp.CompletionItemKindConstant
				}
			}
			detail := declarationDetail(d)
			if headerProject.Model != nil {
				if global := headerProject.Model.Globals[d.Symbol]; global != nil {
					detail = globalHover(d.Name, global.Type, global.Constant).detail
				}
			}
			empty.Items = append(empty.Items, lsp.CompletionItem{Label: name, Kind: &kind, Detail: headerProject.Display(detail)})
		}
		return empty
	}
	// Resolve a qualified enum (including type arguments) through a typed probe.
	probe := header + "fn __hacha_completion__(valor " + receiver + ")\n\timprimir(valor)\n"
	q, _ := h.project(params.TextDocument.URI, &probe)
	if q != nil && q.Model != nil {
		if f, ok := q.Model.Functions["__hacha_completion__"]; ok && len(f.Params) == 1 {
			t := f.Params[0]
			if t.Kind == sema.Enum {
				return displayItems(q, enumCompletionItems(q.Model.EnumFor(t)))
			}
		}
	}
	return empty
}

func displayItems(p *compiler.Project, list *lsp.CompletionList) *lsp.CompletionList {
	for i := range list.Items {
		list.Items[i].Detail = p.Display(list.Items[i].Detail)
	}
	return list
}

func (h *Handler) completionProject(uri lsp.DocumentURI, lines []string, line int, replacement string) *compiler.Project {
	var best *compiler.Project
	tail := append([]string(nil), lines...)
	for i := line + 1; i < len(tail); i++ {
		if strings.TrimSpace(tail[i]) != "" && !strings.HasPrefix(tail[i], "\t") && !strings.HasPrefix(strings.TrimSpace(tail[i]), "//") {
			break
		}
		tail[i] = ""
	}
	for _, input := range [][]string{lines, tail, lines[:line+1]} {
		copy := append([]string(nil), input...)
		copy[line] = replacement
		source := strings.Join(copy, "\n") + "\n"
		p, _ := h.project(uri, &source)
		if p != nil && p.Model != nil && (best == nil || completionScore(p.Model, line) > completionScore(best.Model, line)) {
			best = p
		}
	}
	return best
}

func (h *Handler) moduleContextualCompletion(params *lsp.CompletionParams, lines []string, prefix string) *lsp.CompletionList {
	line := params.Position.Line
	dot := strings.LastIndex(prefix, ".")
	at := strings.LastIndex(prefix, "@")
	if at > dot {
		replacement := prefix[:at] + "imprimir(verdadero)"
		p := h.completionProject(params.TextDocument.URI, lines, line, replacement)
		if p != nil && p.Model != nil {
			if d := enclosingTypeAt(p.Root.Bound, line+1); d != nil {
				return displayItems(p, resolvedMemberItems(p.Model, sema.Type{Kind: sema.Named, Name: d.Name, Args: p.Model.TypeParams[d]}))
			}
		}
	}
	if dot >= 0 {
		p := h.completionProject(params.TextDocument.URI, lines, line, closeCompletionDelimiters(prefix[:dot+1]+"__hacha_probe__"))
		if p != nil && p.Model != nil {
			for expr, t := range p.Model.ExpectedTypes {
				if expr.Position().Filename == p.Root.Path && expr.Position().Line == line+1 {
					if list := gameConstantCompletion(t); list != nil {
						return list
					}
					if info := p.Model.EnumFor(t); info != nil {
						return displayItems(p, enumCompletionItems(info))
					}
				}
			}
		}
	}
	if dot < 0 && at < 0 {
		header := ""
		for _, line := range lines {
			if strings.HasPrefix(line, "usar ") {
				header += line + "\n"
			}
		}
		p, _ := h.project(params.TextDocument.URI, &header)
		if p != nil && p.Root != nil {
			partial := regexp.MustCompile(`[\p{L}\p{N}_]*$`).FindString(prefix)
			var names []string
			for alias := range p.Root.Imports {
				if strings.HasPrefix(alias, partial) {
					names = append(names, alias)
				}
			}
			sort.Strings(names)
			list := &lsp.CompletionList{}
			kind := lsp.CompletionItemKindModule
			for _, alias := range names {
				list.Items = append(list.Items, lsp.CompletionItem{Label: alias, Kind: &kind, Detail: "módulo " + alias})
			}
			return list
		}
	}
	return &lsp.CompletionList{}
}
