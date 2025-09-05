package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {

	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: tracebench <csv:path> <first:int> <col:string>")
		os.Exit(1)
	}

	// parse args
	file := os.Args[1]
	take := must(strconv.Atoi(os.Args[2]))
	coln := os.Args[3]

	// read input file
	t0 := time.Now()
	frame := ReadQframe(file).Slice(0, take)
	fmt.Printf("read frame with %d lines in %s\n", frame.Len(), time.Since(t0))

	// use a subset of columns
	sel := frame.Select("time", coln)
	// for _, col := range sel.ColumnNames() {
	// 	sel = sel.Apply(qframe.Instruction{Fn: function.FloatI, DstCol: col, SrcCol1: col})
	// }
	fmt.Println(sel)

	// fit the predictors
	xs := sel.MustFloatView("time").Slice()
	ys := sel.MustFloatView(coln).Slice()
	fmt.Println("xs:", xs)
	fmt.Println("ys:", ys)
	TryFitters(xs, ys, xs[len(xs)-1], xs[1]/2)

}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}
