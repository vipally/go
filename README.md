# Ally master branch ![Version][version-img] [![Build status][travis-img]][travis-url]
changlist from Official:
- [#2: fmt: add flag "@" to format a verb "v" value with pretty style(indented-multi-lines) string](https://github.com/vipally/go/issues/2)
- [#1: runtime, time: add API BuildTimestamp to report an application's build time](https://github.com/vipally/go/issues/1)
  - [func runtime.BuildTimestamp() int64](https://github.com/vipally/go/blob/ally_master/src/runtime/time.go#L21)
  - [func time.BuildTime() time.Time](https://github.com/vipally/go/blob/ally_master/src/time/time.go#L1517)

******


# The Go Programming Language

Go is an open source programming language that makes it easy to build simple,
reliable, and efficient software.


![Gopher image](doc/gopher/fiveyears.jpg)
*Gopher image by [Renee French][rf], licensed under [Creative Commons 3.0 Attributions license][cc3-by].*

Our canonical Git repository is located at https://go.googlesource.com/go.
There is a mirror of the repository at https://github.com/golang/go.

Unless otherwise noted, the Go source files are distributed under the
BSD-style license found in the LICENSE file.

### Download and Install

#### Binary Distributions

Official binary distributions are available at https://golang.org/dl/.

After downloading a binary release, visit https://golang.org/doc/install
or load doc/install.html in your web browser for installation
instructions.

#### Install From Source

If a binary distribution is not available for your combination of
operating system and architecture, visit
https://golang.org/doc/install/source or load doc/install-source.html
in your web browser for source installation instructions.

### Contributing

Go is the work of hundreds of contributors. We appreciate your help!

To contribute, please read the contribution guidelines:
	https://golang.org/doc/contribute.html

Note that the Go project does not use GitHub pull requests, and that
we use the issue tracker for bug reports and proposals only. See
https://golang.org/wiki/Questions for a list of places to ask
questions about the Go language.

[rf]: https://reneefrench.blogspot.com/
[cc3-by]: https://creativecommons.org/licenses/by/3.0/
[travis-img]: https://travis-ci.org/vipally/go.svg?branch=ally_master
[travis-url]: https://travis-ci.org/vipally/go
[version-img]: https://img.shields.io/badge/version-1.9.0-green.svg
