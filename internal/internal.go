package internal

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/willabides/ezactions"
)

// OutputFailures is what main calls
func OutputFailures(input io.Reader, output io.Writer, rootPath, rootPkg string, passthrough bool) int {
	commander := &ezactions.WorkflowCommander{
		Printer: func(s string) {
			fmt.Fprint(output, s)
		},
	}
	var passthroughWriter io.Writer
	if passthrough {
		passthroughWriter = output
	}
	events := parseEvents(input, passthroughWriter)
	failingTests := events.withTest().withPackage().byKey().filterByResult("fail")
	for _, key := range failingTests.sortedKeys() {
		events := failingTests[key]
		resEvent := events.result()
		if resEvent == nil {
			continue
		}
		pkg := resEvent.Package
		testName := resEvent.Test
		loc, err := findTest(pkg, testName, rootPath, rootPkg)
		if err != nil {
			loc = nil
		}
		msg := events.output()
		if msg == "" {
			msg = `a test failed with no output ¯\_(ツ)_/¯ `
		}
		msg = url.QueryEscape(msg)
		commander.SetErrorMessage(msg, loc)
	}
	return len(failingTests)
}

func findTest(pkg, testName, rootPath, rootPkg string) (*ezactions.CommanderFileLocation, error) {
	if !strings.HasPrefix(pkg, rootPkg) {
		return nil, fmt.Errorf("%s does not contain %s", rootPkg, pkg)
	}
	relPkg := strings.TrimPrefix(pkg, rootPkg)
	relPkg = filepath.FromSlash(relPkg)
	dir := filepath.Join(rootPath, relPkg)
	dirstat, err := os.Stat(dir)
	if err != nil {
		return nil, errors.New("failed statting directory: " + err.Error())
	}
	if !dirstat.IsDir() {
		return nil, fmt.Errorf("not a directory: %q", dir)
	}
	testName = strings.Split(testName, "/")[0]
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, errors.New("failed parsing directory: " + err.Error())
	}
	var testFile string
	var testLine int
	for _, pkg := range pkgs {
		ast.Inspect(pkg, func(n ast.Node) bool {
			decl, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			if decl.Name.Name == testName {
				p := fset.Position(decl.Pos())
				testFile = p.Filename
				testLine = p.Line
			}
			return true
		})
		if testFile != "" {
			break
		}
	}
	testFile, err = filepath.Rel(rootPath, testFile)
	if err != nil {
		return nil, err
	}
	testFile = strings.TrimPrefix(filepath.Join("..", testFile), ".")
	return &ezactions.CommanderFileLocation{
		File: testFile,
		Line: testLine,
	}, nil
}
