package main

import (
	"fmt"
	"io"
	"os"

	"github.com/fasterthanlime/wizardry/wizardry/wizinterpreter"
	"github.com/fasterthanlime/wizardry/wizardry/wizparser"
	"github.com/fasterthanlime/wizardry/wizardry/wizutil"
	"github.com/go-errors/errors"
)

func doIdentify() error {
	magdir := *identifyArgs.magdir

	NoLogf := func(format string, args ...interface{}) {}

	Logf := func(format string, args ...interface{}) {
		fmt.Println(fmt.Sprintf(format, args...))
	}

	pctx := &wizparser.ParseContext{
		Logf: NoLogf,
	}

	if *appArgs.debugParser {
		pctx.Logf = Logf
	}

	book := make(wizparser.Spellbook)
	err := pctx.ParseAll(magdir, book)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	target := *identifyArgs.target
	targetReader, err := os.Open(target)
	if err != nil {
		panic(err)
	}

	defer targetReader.Close()

	var targetSlice []byte
	stat, _ := targetReader.Stat()
	targetSlice = make([]byte, stat.Size())

	n, err := io.ReadFull(targetReader, targetSlice)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	ictx := &wizinterpreter.InterpretContext{
		Logf: NoLogf,
		Book: book,
	}

	if *appArgs.debugInterpreter {
		ictx.Logf = Logf
	}

	result, err := ictx.Identify(targetSlice[:n])
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s\n", target, wizutil.MergeStrings(result))

	return nil
}
