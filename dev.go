// +build dev

package main

import (
	"path"
	"runtime"
	"github.com/joho/godotenv"
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot get current dir: no caller information")
	}

	projectRoot := path.Clean(path.Dir(filename))

	godotenv.Load(path.Join(projectRoot, ".env"))
	godotenv.Load(path.Join(projectRoot, ".env.dist"))
}
