package main

import (
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/willabides/go-test2action/internal"
)

var cli struct {
	Passthrough bool   `kong:"help='write test output to stdout'"`
	RootPath    string `kong:"default='.',help='root path for test packages'"`
	RootPkg     string `kong:"required,help='the package at root-path'"`
}

func main() {
	ctx := kong.Parse(&cli)
	rootPath, err := filepath.Abs(cli.RootPath)
	ctx.FatalIfErrorf(err)
	failCount := internal.OutputFailures(os.Stdin, os.Stdout, rootPath, cli.RootPkg, cli.Passthrough)
	if failCount != 0 {
		os.Exit(1)
	}
}
