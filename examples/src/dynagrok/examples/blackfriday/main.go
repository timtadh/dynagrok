package main

import (
	"os"
	"io/ioutil"
)

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

func main() {
	bits, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(bluemonday.UGCPolicy().SanitizeBytes(
		blackfriday.MarkdownCommon(bits)))
}
