package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// SPEC 4.7 T27: все статические сообщения приложения переведены на каждый язык.
func TestCatalogsCoverSourceMessages(t *testing.T) {
	messages := sourceMessages(t)
	for _, language := range supportedLanguages {
		if language == English {
			continue
		}
		catalog := messageCatalogs[language]
		for message := range messages {
			if _, ok := catalog[message]; !ok {
				t.Errorf("%s: message is not translated: %q", language, message)
			}
		}
	}
}

func sourceMessages(t *testing.T) map[string]struct{} {
	t.Helper()
	root := filepath.Join("..", "..", "internal")
	word := regexp.MustCompile(`[A-Za-z]{2,}`)
	format := regexp.MustCompile(`%[#0 +\-]*[0-9]*(?:\.[0-9]+)?[a-zA-Z]`)
	messages := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			argument, localized := localizedArgument(call)
			if !localized || argument >= len(call.Args) {
				return true
			}
			literal, ok := call.Args[argument].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			message, err := strconv.Unquote(literal.Value)
			if err != nil || strings.HasPrefix(message, "{") {
				return true
			}
			plain := format.ReplaceAllString(message, "")
			if !word.MatchString(plain) {
				return true
			}
			messages[message] = struct{}{}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return messages
}

func localizedArgument(call *ast.CallExpr) (int, bool) {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		switch function.Name {
		case "warn":
			return 0, true
		case "rule":
			return 1, true
		}
	case *ast.SelectorExpr:
		switch function.Sel.Name {
		case "printf", "warnf":
			return 0, true
		case "Text", "Sprintf", "Errorf", "NewError":
			if pkg, ok := function.X.(*ast.Ident); ok && pkg.Name == "i18n" {
				return 0, true
			}
		case "New":
			if pkg, ok := function.X.(*ast.Ident); ok && pkg.Name == "errs" {
				return 1, true
			}
		case "Usagef", "Invalidf", "NotFoundf", "Conflictf", "Envf", "Lockf", "Cancelledf":
			if pkg, ok := function.X.(*ast.Ident); ok && pkg.Name == "errs" {
				return 0, true
			}
		}
	}
	return 0, false
}
