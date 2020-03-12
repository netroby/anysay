package main

import (
	"github.com/netroby/anysay/app/data"
	"log"

	"github.com/shurcooL/vfsgen"
)

func main () {
	err := vfsgen.Generate(data.Assets, vfsgen.Options{
		PackageName:  "main",
		VariableName: "Assets",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
