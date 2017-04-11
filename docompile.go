package main

import (
	"fmt"

	"github.com/fasterthanlime/wizardry/wizardry/wizcompiler"
	"github.com/fasterthanlime/wizardry/wizardry/wizparser"
	"github.com/go-errors/errors"
)

func doCompile() error {
	magdir := *compileArgs.magdir

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

	err = wizcompiler.Compile(book, *compileArgs.chatty)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
