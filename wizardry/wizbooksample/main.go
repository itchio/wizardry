package main

import (
	"fmt"
	"os"

	"github.com/fasterthanlime/wizardry/wizardry/wizbook"
	"github.com/fasterthanlime/wizardry/wizardry/wizutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: wizbooksample TARGET\n")
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

	res, err := wizbook.Identify(r, stats.Size(), 0)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s\n", target, wizutil.MergeStrings(res))
}
