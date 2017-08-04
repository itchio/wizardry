package main

import (
	"fmt"
	"os"

	"github.com/fasterthanlime/wizardry/wizardry/wizutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s TARGET\n", os.Args[0])
		os.Exit(1)
	}

	target := os.Args[1]

	r, err := os.Open(target)
	if err != nil {
		panic(err)
	}

	stats, err := r.Stat()
	if err != nil {
		panic(err)
	}

	sr := wizutil.NewSliceReader(r, 0, stats.Size())

	res := Identify(sr, 0)
	fmt.Printf("%s: %s\n", target, wizutil.MergeStrings(res))
}
