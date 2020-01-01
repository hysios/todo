package main

import (
	// "flag"
	// "log"
	// "os"

	// "github.com/hysios/todo/parser"
	// "github.com/hysios/todo/printer"
	"github.com/hysios/todo/cmd"
)

// var input = flag.String("input", "", "input todo file")

func main() {
	cmd.Execute()

	// flag.Parse()

	// if len(*input) == 0 {
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	// f, err := os.Open(*input)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// todo, err := parser.Parse(*input, f)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// print := printer.New(todo)
	// print.Print()

	// if buf, err := json.MarshalIndent(todo, "", "  "); err == nil {
	// 	log.Printf("json \n%s", buf)
	// }

}
