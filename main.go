package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fasterthanlime/wizardry/wizardry"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: wizardry RULES TARGET")
	}

	rule := os.Args[1]
	ruleReader, err := os.Open(rule)
	if err != nil {
		panic(err)
	}

	defer ruleReader.Close()

	target := os.Args[2]
	targetReader, err := os.Open(target)
	if err != nil {
		panic(err)
	}

	defer targetReader.Close()

	targetSlice := make([]byte, 1024)
	n, err := io.ReadFull(targetReader, targetSlice)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			// ok then
		} else {
			panic(err)
		}
	}

	result, err := wizardry.Identify(ruleReader, targetSlice[:n])
	if err != nil {
		panic(err)
	}

	fmt.Println(target, strings.Join(result, " "))
}
