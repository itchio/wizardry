package main

import (
	"fmt"

	"github.com/itchio/wizardry/wizardry/wizcompiler"
	"github.com/itchio/wizardry/wizardry/wizparser"
)

func doCompile() error {
	magdir := *compileArgs.magdir

	NoLogf := func(format string, args ...any) {}

	Logf := func(format string, args ...any) {
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
		return err
	}

	err = wizcompiler.Compile(book, *compileArgs.output, *compileArgs.chatty, *compileArgs.emitComments, *compileArgs.pkg)
	if err != nil {
		return err
	}

	return nil
}
