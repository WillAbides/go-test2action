package internal

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/willabides/ezactions"
)

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
		var loc *ezactions.CommanderFileLocation
		testFile, testLine, err := findTest(pkg, testName, rootPath, rootPkg)
		if err == nil && testLine != 0 {
			loc = &ezactions.CommanderFileLocation{
				File: testFile,
				Line: testLine,
			}
		}
		commander.SetErrorMessage(resEvent.Output, loc)
	}
	return len(failingTests)
}

func findTest(pkg, testName, rootPath, rootPkg string) (string, int, error) {
	if !strings.HasPrefix(pkg, rootPkg) {
		return "", 0, fmt.Errorf("%s does not contain %s", rootPkg, pkg)
	}
	relPkg := strings.TrimPrefix(pkg, rootPkg)
	relPkg = filepath.FromSlash(relPkg)
	dir := filepath.Join(rootPath, relPkg)
	dirstat, err := os.Stat(dir)
	if err != nil {
		return "", 0, errors.New("failed statting directory: " + err.Error())
	}
	if !dirstat.IsDir() {
		return "", 0, fmt.Errorf("not a directory: %q", dir)
	}
	testName = strings.Split(testName, "/")[0]
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return "", 0, errors.New("failed parsing directory: " + err.Error())
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
		return "", 0, err
	}
	testFile = strings.TrimPrefix(filepath.Join("..", testFile), ".")
	return testFile, testLine, nil
}
