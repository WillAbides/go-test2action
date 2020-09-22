package internal

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

func Test_findTest(t *testing.T) {
	_, _, exLine, ok := runtime.Caller(0)
	require.True(t, ok)
	exFilename := "./internal/internal_test.go"
	exLine--
	rootPath, err := filepath.Abs("..")
	require.NoError(t, err)
	rootPkg := "github.com/willabides/go-test2action"
	pkg := path.Join(rootPkg, "internal")
	loc, err := findTest(pkg, "Test_findTest", rootPath, rootPkg)
	require.NoError(t, err)
	require.NotNil(t, loc)
	require.Equal(t, exLine, loc.Line)
	require.Equal(t, exFilename, loc.File)
}

func TestOutputFailures(t *testing.T) {
	err := os.Rename("testdata/dummytest/dummy_test.go.tmpl", "testdata/dummytest/dummy_test.go")
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Rename("testdata/dummytest/dummy_test.go", "testdata/dummytest/dummy_test.go.tmpl")
		require.NoError(t, err)
	})
	pkg := "github.com/willabides/go-test2action/internal/testdata/dummytest"
	input := execPkgTemplate(t, pkg, "testdata/example.go.tmpl")
	expected := `=== RUN   TestPassing
--- PASS: TestPassing (0.00s)
=== RUN   TestFailing
--- FAIL: TestFailing (0.00s)
=== RUN   TestWithSubs
=== RUN   TestWithSubs/passing
=== RUN   TestWithSubs/failing
=== RUN   TestWithSubs/passing_with_println_output
hello
world
=== RUN   TestWithSubs/failing_with_println_output
hello
world
--- FAIL: TestWithSubs (0.00s)
    --- PASS: TestWithSubs/passing (0.00s)
    --- FAIL: TestWithSubs/failing (0.00s)
    --- PASS: TestWithSubs/passing_with_println_output (0.00s)
    --- FAIL: TestWithSubs/failing_with_println_output (0.00s)
FAIL
FAIL	github.com/willabides/go-test2action/internal/testdata/dummytest	0.008s
::error file=./internal/testdata/dummytest/dummy_test.go,line=8,col=0::=== RUN   TestFailing%0A--- FAIL: TestFailing (0.00s)%0A
::error file=./internal/testdata/dummytest/dummy_test.go,line=12,col=0::=== RUN   TestWithSubs%0A--- FAIL: TestWithSubs (0.00s)%0A
::error file=./internal/testdata/dummytest/dummy_test.go,line=12,col=0::=== RUN   TestWithSubs/failing%0A    --- FAIL: TestWithSubs/failing (0.00s)%0A
::error file=./internal/testdata/dummytest/dummy_test.go,line=12,col=0::=== RUN   TestWithSubs/failing_with_println_output%0Ahello%0Aworld%0A    --- FAIL: TestWithSubs/failing_with_println_output (0.00s)%0A
`
	require.NoError(t, err)
	var outbytes []byte
	outbuf := bytes.NewBuffer(outbytes)
	failureCount := OutputFailures(input, outbuf, "..", "github.com/willabides/go-test2action", true)
	require.Equal(t, 4, failureCount)
	out, err := ioutil.ReadAll(outbuf)
	require.NoError(t, err)
	require.Equal(t, expected, string(out))
}

func execPkgTemplate(t *testing.T, pkg, tmplfile string) io.Reader {
	t.Helper()
	tmpl, err := template.ParseFiles(tmplfile)
	require.NoError(t, err)
	var outBytes bytes.Buffer
	separator := string(os.PathSeparator)
	locpath := filepath.Join(strings.Split(pkg, separator)[3:]...)
	locpath = "." + separator + filepath.Join(locpath, "dummy_test.go")
	err = tmpl.Execute(&outBytes, struct{ Package, LocationPath string }{
		Package:      pkg,
		LocationPath: locpath,
	})
	require.NoError(t, err)
	return &outBytes
}
