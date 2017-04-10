package main

import (
	"fmt"
	"io/ioutil"
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

	buf, err := ioutil.ReadFile(target)
	if err != nil {
		panic(err)
	}

	res, err := wizbook.Identify(buf, 0)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s\n", target, wizutil.MergeStrings(res))
}
