# wizardry

[![build status](https://git.itch.ovh/itchio/wizardry/badges/master/build.svg)](https://git.itch.ovh/itchio/wizardry/commits/master)
[![codecov](https://codecov.io/gh/itchio/wizardry/branch/master/graph/badge.svg)](https://codecov.io/gh/itchio/wizardry)
[![Go Report Card](https://goreportcard.com/badge/github.com/itchio/wizardry)](https://goreportcard.com/report/github.com/itchio/wizardry)
[![GoDoc](https://godoc.org/github.com/itchio/wizardry?status.svg)](https://godoc.org/github.com/itchio/wizardry)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/itchio/wizardry/blob/master/LICENSE)

wizardry is a toolkit to deal with libmagic rule files (sources, not compiled)

It contains:

  * A parser, which turn magic rule files into an AST
  * An interpreter, which identifies a target by following
  the rules in the AST
  * A compiler, which generates go code to follow the
  rules in the AST


## License

wizardry is released under the MIT license, see the
`LICENSE` file for details.

