package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/vktec/gleg/gen"
)

const URL = "https://www.khronos.org/registry/OpenGL/xml/gl.xml"

func main() {
	file := flag.String("file", "", "load registry from xml file rather than downloading")
	flag.Parse()
	log.SetFlags(0)

	var r io.ReadCloser
	var err error
	if *file != "" {
		r, err = os.Open(*file)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		res, err := http.Get(URL)
		if err != nil {
			log.Fatal(err)
		}
		r = res.Body
	}
	reg, err := gen.Parse(r)
	if err != nil {
		log.Fatal(err)
	}
	r.Close()

	src, err := gen.Generate(reg)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("gl.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, "//go:generate go run ./cmd/gleg/"); err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write(src); err != nil {
		log.Fatal(err)
	}
}
