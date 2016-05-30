package main

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	fs, _ := ioutil.ReadDir(".")
	out, _ := os.Create("templates.go")
	out.Write([]byte("package main \n\nconst(\n"))
	for _, f := range fs {
		if strings.HasSuffix(f.Name(), ".tmpl") {
			out.Write([]byte(strings.TrimSuffix(f.Name(), ".tmpl") + " = ``"))
			f, _ := os.Open(f.Name())
			io.Copy(out, f)
			out.Write([]byte("`\n"))
		}
	}
	out.Write([]byte(")\n"))
}
