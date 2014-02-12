package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

var f = flag.String("f", "", "Output file, else stdout.")
var p = flag.String("p", "main", "Package.")

var fRE = regexp.MustCompile("[a-zA-Z0-9_]+")

func main() {
	flag.Parse()
	w := os.Stdout
	var err error
	if *f != "" {
		if w, err = os.Create(*f); err != nil {
			log.Fatal(err)
		}
		defer w.Close()
	}
	fmt.Fprintf(w, "package %s\n", *p)
	for _, fname := range flag.Args() {
		b, err := ioutil.ReadFile(fname)
		if err != nil {
			log.Fatal(err)
		}
		v := strings.Join(fRE.FindAllString(fname, -1), "_")
		fmt.Fprintf(w, "\nvar %s = []byte(%q)\n", v, b)
	}
}
