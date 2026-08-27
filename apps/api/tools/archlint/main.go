// Command archlint enforces the repository's dependency direction and a small
// set of high-signal request-path invariants without third-party dependencies.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const modulePath = "github.com/mattwebhub/micro1-template/apps/api"

type rule struct {
	id, summary, good, bad string
}

var rules = map[string]rule{
	"GO-ARCH-001": {
		id: "GO-ARCH-001", summary: "domain imports the Go standard library only",
		good: "internal/domain/project.go imports errors and strings", bad: "internal/domain/project.go imports pgx or an internal adapter",
	},
	"GO-ARCH-002": {
		id: "GO-ARCH-002", summary: "internal package imports follow the layer dependency matrix",
		good: "services import domain and ports", bad: "services import HTTP transport or PostgreSQL adapters",
	},
	"GO-CTX-001": {
		id: "GO-CTX-001", summary: "request-path code propagates the request context",
		good: "repository.Get(r.Context(), id)", bad: "repository.Get(context.Background(), id) in transport",
	},
	"GO-HTTP-001": {
		id: "GO-HTTP-001", summary: "HTTP JSON bodies use the bounded shared decoder",
		good: "response.DecodeJSON(w, r, &input, maxBytes)", bad: "json.NewDecoder(r.Body) or io.ReadAll(r.Body) in transport",
	},
}

type violation struct {
	ruleID, file string
	line         int
	message      string
}

func main() {
	root := flag.String("root", ".", "repository root")
	explain := flag.String("explain", "", "print one rule and exit")
	flag.Parse()
	if *explain != "" {
		if err := explainRule(*explain); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}
	violations, err := check(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(violations) == 0 {
		fmt.Println("architecture: all executable rules passed")
		return
	}
	for _, item := range violations {
		fmt.Printf("%s:%d: [%s] %s; see ../../docs/rules/%s.md\n", item.file, item.line, item.ruleID, item.message, item.ruleID)
	}
	os.Exit(1)
}

func explainRule(id string) error {
	item, ok := rules[id]
	if !ok {
		ids := make([]string, 0, len(rules))
		for known := range rules {
			ids = append(ids, known)
		}
		sort.Strings(ids)
		return fmt.Errorf("unknown rule %q; choose one of %s", id, strings.Join(ids, ", "))
	}
	fmt.Printf("%s: %s\nCompliant: %s\nViolation: %s\nVerify: make arch\nDocumentation: ../../docs/rules/%s.md\n", item.id, item.summary, item.good, item.bad, item.id)
	return nil
}

func check(root string) ([]violation, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("archlint: resolve root: %w", err)
	}
	var violations []violation
	err = filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(absoluteRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" || strings.HasPrefix(relative, "tools/archlint/testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		layer := packageLayer(relative)
		if layer == "" {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return fmt.Errorf("archlint: parse %s: %w", relative, err)
		}
		violations = append(violations, checkImports(fileSet, parsed, relative, layer)...)
		violations = append(violations, checkCalls(fileSet, parsed, relative, layer)...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("archlint: walk: %w", err)
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].file == violations[j].file {
			return violations[i].line < violations[j].line
		}
		return violations[i].file < violations[j].file
	})
	return violations, nil
}

func checkImports(fileSet *token.FileSet, file *ast.File, path, sourceLayer string) []violation {
	allowed := map[string]map[string]bool{
		"domain": {}, "config": {}, "observability": {"observability": true},
		"ports":     {"domain": true, "ports": true},
		"services":  {"domain": true, "ports": true, "services": true},
		"adapters":  {"domain": true, "ports": true, "adapters": true},
		"transport": {"domain": true, "services": true, "transport": true},
		"bootstrap": {"domain": true, "ports": true, "services": true, "adapters": true, "transport": true, "observability": true, "config": true, "bootstrap": true},
		"cmd":       {"bootstrap": true},
	}
	var result []violation
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		line := fileSet.Position(spec.Pos()).Line
		if sourceLayer == "domain" && strings.Contains(strings.Split(importPath, "/")[0], ".") {
			result = append(result, violation{"GO-ARCH-001", path, line, fmt.Sprintf("domain imports non-standard package %q; move I/O capability to a port and adapter", importPath)})
			continue
		}
		prefix := modulePath + "/internal/"
		if !strings.HasPrefix(importPath, prefix) {
			continue
		}
		targetLayer := strings.Split(strings.TrimPrefix(importPath, prefix), "/")[0]
		if !allowed[sourceLayer][targetLayer] {
			result = append(result, violation{"GO-ARCH-002", path, line, fmt.Sprintf("%s imports forbidden %s package %q; follow the dependency matrix", sourceLayer, targetLayer, importPath)})
		}
	}
	return result
}

func checkCalls(fileSet *token.FileSet, file *ast.File, path, layer string) []violation {
	if layer != "transport" {
		return nil
	}
	aliases := importAliases(file)
	var result []violation
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		importPath := aliases[identifier.Name]
		line := fileSet.Position(call.Pos()).Line
		if importPath == "context" && (selector.Sel.Name == "Background" || selector.Sel.Name == "TODO") {
			result = append(result, violation{"GO-CTX-001", path, line, "transport creates a root context; propagate r.Context() to request I/O"})
		}
		if strings.Contains(path, "/response/") {
			return true
		}
		if (importPath == "encoding/json" && selector.Sel.Name == "NewDecoder") || (importPath == "io" && selector.Sel.Name == "ReadAll") {
			result = append(result, violation{"GO-HTTP-001", path, line, "transport reads a body directly; use response.DecodeJSON for size and shape enforcement"})
		}
		return true
	})
	return result
}

func importAliases(file *ast.File) map[string]string {
	result := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := strings.Split(path, "/")[len(strings.Split(path, "/"))-1]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		result[name] = path
	}
	return result
}

func packageLayer(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) >= 3 && parts[0] == "cmd" && parts[1] == "api" {
		return "cmd"
	}
	if len(parts) < 3 || parts[0] != "internal" {
		return ""
	}
	switch parts[1] {
	case "domain", "ports", "services", "adapters", "observability", "bootstrap", "config":
		return parts[1]
	case "transport":
		return "transport"
	default:
		return ""
	}
}
