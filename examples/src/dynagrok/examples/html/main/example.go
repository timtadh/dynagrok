// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This example demonstrates parsing HTML data and walking the resulting tree.
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"strings"

	"dynagrok/examples/html"
)

var (
	w      = &bytes.Buffer{}
	logger = log.New(w, "", 0)
)

func ExampleParse(infile, outfile string) {
	//s := `<p>Links:</p><ul><li><a href="foo">Foo</a><li><a href="/bar/baz">BarBaz</a></ul>`
	s, err := ioutil.ReadFile(infile)
	if err != nil {
		log.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(string(s)))
	if err != nil {
		log.Fatal(err)
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					logger.Printf("%v", a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	ioutil.WriteFile(outfile, []byte(w.String()), 0777)
	// Output:
	// foo
	// /bar/baz
}

func main() {
	in := flag.String("input", "in.html", "A well-formed html file")
	out := flag.String("output", "out.txt", "The links in the html file")
	flag.Parse()
	ExampleParse(*in, *out)
}
