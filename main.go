package main

import (
	"fmt"
	"io"
	"os"

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

	// targetSlice := make([]byte, 2048)
	stat, _ := targetReader.Stat()
	targetSlice := make([]byte, stat.Size())
	n, err := io.ReadFull(targetReader, targetSlice)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			// ok then
		} else {
			panic(err)
		}
	}

	Logf := func(format string, args ...interface{}) {
		fmt.Println(fmt.Sprintf(format, args...))
	}

	pctx := &wizardry.ParseContext{
		Logf: func(format string, args ...interface{}) {},
	}

	book := make(wizardry.Spellbook)
	err = pctx.Parse(ruleReader, book)
	if err != nil {
		panic(err)
	}

	ictx := &wizardry.InterpretContext{
		Logf: Logf,
		Book: book,
	}

	result, err := ictx.Identify(targetSlice[:n])
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s\n", target, result)
}
