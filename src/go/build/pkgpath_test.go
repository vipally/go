// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var (
	fsRoot      = testContext.joinPath(getwd(), "testdata/vroot")
	testContext = defaultContext()
	vroot       = "/" //virtual fs root
	thisDir     = getwd()
	showResult  = false
)

// init test evnironment
// build a virtual file system
func init() {
	if testContext.GOOS == "windows" {
		vroot = `v:`
	}
	testContext.IsDir = func(vdir string) bool {
		dir := full(vdir)
		fi, err := os.Stat(dir)
		ok := err == nil && fi.IsDir()
		//fmt.Println("IsDir", vdir, dir, ok)
		return ok
	}
	testContext.OpenFile = func(vdir string) (io.ReadCloser, error) {
		dir := full(vdir)
		f, err := os.Open(dir)
		if err != nil {
			return nil, err // nil interface
		}
		return f, nil
	}
	testContext.ReadDir = func(vdir string) ([]os.FileInfo, error) {
		dir := full(vdir)
		return ioutil.ReadDir(dir)
	}
	testContext.GOROOT = vdir("__goroot__")
	testContext.GOPATH = fmt.Sprintf("%s%c%s%c%s", vdir("gopath1"), filepath.ListSeparator, vdir("gopath2"), filepath.ListSeparator, vdir("gopath3"))

	//	fmt.Printf("%+v\n", gblSrcs)
	//	fmt.Printf("%+v\n", testContext.GOROOT)
	//	fmt.Printf("%+v\n", goRootSrc)
	//	fmt.Printf("%+v\n", full(goRootSrc))
	//	fmt.Printf("%+v\n", gblSrcs[0])
	//	fmt.Printf("%+v\n", full(gblSrcs[0]))
	//	fmt.Printf("%+v\n", testContext.SearchLocalRoot(vdir(`localroot1\src\vendor`)))
	//	fmt.Printf("%+v\n", full("v:\\"))
}

func TestSearchLocalRoot(t *testing.T) {
	testCases := [][]string{
		//related, localroot
		[]string{"noroot1/local1", ""},
		[]string{"localroot1", ""},
		[]string{"localroot1/src", "localroot1"},
		[]string{"localroot1/src/local1", "localroot1"},
		[]string{"localroot1/src/vendor", "localroot1"},
		[]string{"localroot1/src/vendor/vendored", "localroot1"},
		[]string{"localroot1/src/vendor/localrootv1", "localroot1"},
		[]string{"localroot1/src/vendor/localrootv1/src", "localroot1/src/vendor/localrootv1"},
		[]string{"localroot1/src/vendor/localrootv1/src/local1", "localroot1/src/vendor/localrootv1"},
		[]string{"localroot1/src/vendor/localrootv1/src/vendor", "localroot1/src/vendor/localrootv1"},
		[]string{"localroot1/src/vendor/localrootv1/src/vendor/localrootv1", "localroot1/src/vendor/localrootv1"},
		[]string{"localroot1/src/vendor/localrootv1/src/vendor/localrootv1/src", "localroot1/src/vendor/localrootv1/src/vendor/localrootv1"},
		[]string{"gopath1/src", "gopath1"},
		[]string{"gopath1/src/local1", "gopath1"},
		[]string{"gopath1/src/noroot1/local1", "gopath1"},
		[]string{"gopath1/src/localroot1", "gopath1"},
		[]string{"gopath1/src/localroot1/src", "gopath1/src/localroot1"},
		[]string{"__goroot__", ""},
		[]string{"__goroot__/src", ""},
		[]string{"__goroot__/src/fmt", ""},
	}
	for _, testCase := range testCases {
		dir, want := vdir(testCase[0]), vdir(testCase[1])
		got := testContext.SearchLocalRoot(dir)
		if !reflect.DeepEqual(want, got) {
			t.Errorf("SearchLocalRoot(%s) fail, want [%s] got [%s]", dir, want, got)
		} else {
			if showResult {
				if len(dir) <= 30 {
					fmt.Printf("SearchLocalRoot[%-30s]=[%-30s] path=[%s]\n", dir, want, full(want))
				} else {
					fmt.Printf("SearchLocalRoot[%s]\n              =[%s]\n          path=[%s]\n", dir, want, full(want))
				}
			}
		}
	}
}

func TestFormatImportPath(t *testing.T) {
	type _Want = FormatImport

	type _Case struct {
		id       int
		imported string
		dir      string
		wantErr  error
		want     *_Want
	}
	testCases := []*_Case{
		&_Case{1, `./doesnotexist`, `__goroot__/src/go/build`, fmt.Errorf(`import "./doesnotexist": cannot find package at v:\__goroot__\src\go\build\doesnotexist`), &_Want{}},
		&_Case{2, `x/(y)/z`, `noroot1`, fmt.Errorf(`import "x/(y)/z": invalid character U+0028 '('`), &_Want{}},
		&_Case{3, `x/Programme Files/y`, `noroot1`, fmt.Errorf(`import "x/Programme Files/y": invalid character U+0020 ' '`), &_Want{}},
		&_Case{4, `#/#`, `noroot1`, fmt.Errorf(`import "#/#": invalid character U+0023 '#'`), &_Want{}},
		&_Case{5, `##`, `noroot1`, fmt.Errorf(`import "##": invalid character U+0023 '#'`), &_Want{}},
		&_Case{6, `c:/x/y/z`, `noroot1`, fmt.Errorf(`import "c:/x/y/z": invalid character U+003A ':'`), &_Want{}},
		&_Case{7, `./#/x/y/z`, `noroot1`, fmt.Errorf(`import "./#/x/y/z": invalid character U+0023 '#'`), &_Want{}},
		&_Case{8, `x\y\z`, `noroot1`, fmt.Errorf(`import "x\\y\\z": invalid character U+005C '\'`), &_Want{}},
		&_Case{9, `...`, `noroot1`, fmt.Errorf(`import "...": invalid import path`), &_Want{}},
		&_Case{10, `#/./x/y/z`, `noroot1`, fmt.Errorf(`import "#/./x/y/z": invalid import path`), &_Want{}},
		&_Case{11, `.../x/y/z`, `noroot1`, fmt.Errorf(`import ".../x/y/z": invalid import path`), &_Want{}},
		&_Case{12, ``, `noroot1`, fmt.Errorf(`import "": invalid import path`), &_Want{}},
		&_Case{13, `.//local1`, `noroot1`, fmt.Errorf(`import ".//local1": invalid import path`), &_Want{}},
		&_Case{14, `/x/y/z`, `noroot1`, fmt.Errorf(`import "/x/y/z": cannot import absolute path`), &_Want{}},
		&_Case{15, `//x/y/z`, `noroot1`, fmt.Errorf(`import "//x/y/z": cannot import absolute path`), &_Want{}},
		&_Case{16, `.`, `notexist`, fmt.Errorf(`import ".": cannot find package at v:\notexist`), &_Want{}},
		&_Case{17, `.`, `__goroot__/src/notexist`, fmt.Errorf(`import ".": cannot find package at v:\__goroot__\src\notexist`), &_Want{}},
		&_Case{18, `.`, `gopath1/src/notexist`, fmt.Errorf(`import ".": cannot find package at v:\gopath1\src\notexist`), &_Want{}},
		&_Case{19, `.`, `noroot1/testdata/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\noroot1\testdata\local1`, FmtImportPath: `v:\noroot1\testdata\local1`, Dir: `v:\noroot1\testdata\local1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleSelf, Formated: true}},
		&_Case{20, `.`, `localroot1/src/testdata/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\testdata\local1`, FmtImportPath: `v:\localroot1\src\testdata\local1`, Dir: `v:\localroot1\src\testdata\local1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleSelf, Formated: true}},
		&_Case{21, `#/x/y/z`, `notexist`, nil, &_Want{OriginImportPath: `#/x/y/z`, ImporterDir: `v:\notexist`, FmtImportPath: `#/x/y/z`, Dir: ``, Root: ``, ConflictDir: ``, Type: PackageUnknown, Style: ImportStyleLocalRoot, Formated: false}},
		&_Case{22, `x/y/z`, `notexist`, nil, &_Want{OriginImportPath: `x/y/z`, ImporterDir: `v:\notexist`, FmtImportPath: `x/y/z`, Dir: ``, Root: ``, ConflictDir: ``, Type: PackageUnknown, Style: ImportStyleGlobal, Formated: false}},
		&_Case{23, `.`, `noroot1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\noroot1`, FmtImportPath: `v:\noroot1`, Dir: `v:\noroot1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleSelf, Formated: true}},
		&_Case{24, `./local1`, `noroot1`, nil, &_Want{OriginImportPath: `./local1`, ImporterDir: `v:\noroot1`, FmtImportPath: `v:\noroot1\local1`, Dir: `v:\noroot1\local1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleRelated, Formated: true}},
		&_Case{25, `..`, `noroot1/local1`, nil, &_Want{OriginImportPath: `..`, ImporterDir: `v:\noroot1\local1`, FmtImportPath: `v:\noroot1`, Dir: `v:\noroot1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleRelated, Formated: true}},
		&_Case{26, `#`, `localroot1/src/local1`, nil, &_Want{OriginImportPath: `#`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `#`, Dir: ``, Root: ``, ConflictDir: ``, Type: PackageUnknown, Style: ImportStyleLocalRoot, Formated: false}},
		&_Case{27, `.`, `localroot1/src/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `#/local1`, Dir: `v:\localroot1\src\local1`, Root: `v:\localroot1`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{28, `.`, `gopath1/src/localroot1/src/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath1\src\localroot1\src\local1`, FmtImportPath: `#/local1`, Dir: `v:\gopath1\src\localroot1\src\local1`, Root: `v:\gopath1\src\localroot1`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{29, `.`, `gopath1/src/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath1\src\local1`, FmtImportPath: `#/local1`, Dir: `v:\gopath1\src\local1`, Root: `v:\gopath1`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{30, `.`, `gopath2/src/local2`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath2\src\local2`, FmtImportPath: `local2`, Dir: `v:\gopath2\src\local2`, Root: `v:\gopath2`, ConflictDir: `v:\gopath1\src\local2`, Type: PackageGoPath, Style: ImportStyleGlobal, Formated: true}},
		&_Case{31, `.`, `gopath2/src/localroot2/src/local2`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath2\src\localroot2\src\local2`, FmtImportPath: `#/local2`, Dir: `v:\gopath2\src\localroot2\src\local2`, Root: `v:\gopath2\src\localroot2`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{32, `.`, `__goroot__/src/fmt`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\__goroot__\src\fmt`, FmtImportPath: `fmt`, Dir: `v:\__goroot__\src\fmt`, Root: `v:\__goroot__`, ConflictDir: ``, Type: PackageGoRoot, Style: ImportStyleGlobal, Formated: true}},
	}
	for i, testCase := range testCases {
		dir := vdir(testCase.dir)
		formated, err := testContext.FormatImportPath(testCase.imported, dir)
		if genCase := false; genCase { //genCase
			if err != nil {
				fmt.Printf("&_Case{%d, `%s`, `%s`, fmt.Errorf(`%s`),&_Want{}},\n", i+1, testCase.imported, testCase.dir, err.Error())
			} else {
				//fmt.Printf("%+v\n", formated)
				g := &formated
				fmt.Printf("&_Case{%d, `%s`, `%s`, nil, &_Want{OriginImportPath:`%s`, ImporterDir:`%s`, FmtImportPath:`%s`, Dir:`%s`, Root:`%s`, ConflictDir:`%s`, Type:%s, Style:%s, Formated:%v}},\n",
					i+1, testCase.imported, testCase.dir,
					g.OriginImportPath, g.ImporterDir, g.FmtImportPath, g.Dir, g.Root, g.ConflictDir, g.Type, g.Style, g.Formated)
			}
		} else { //verify
			//fmt.Printf("%d FormatImportPath(%q, %s)=%+v %v\n", i+1, testCase.imported, dir, formated, err)

			errEq := reflect.DeepEqual(err, testCase.wantErr)
			if testCase.wantErr != nil || !errEq {
				if !errEq {
					t.Errorf("FormatImportPath [%d %q %s] wantErr=[%+v] gotErr: [%+v]", testCase.id, testCase.imported, dir, testCase.wantErr, err)
				}
				continue
			}
			if !reflect.DeepEqual(&formated, testCase.want) {
				t.Errorf("FormatImportPath[%d %q %s] \n    want [%+v]\n     got [%+v]\n", testCase.id, testCase.imported, dir, testCase.want, &formated)
			}
		}
	}
}

func TestFindImport(t *testing.T) {
	type _Want = PackagePath

	type _Case struct {
		id       int
		imported string
		dir      string
		mode     ImportMode
		wantErr  error
		want     *_Want
	}
	testCases := []*_Case{
		&_Case{1, `c:/go/src`, `noroot1`, 0, fmt.Errorf(`import "c:/go/src": invalid character U+003A ':'`), &_Want{}},
		&_Case{2, ``, `noroot1`, 0, fmt.Errorf(`import "": invalid import path`), &_Want{}},
		&_Case{3, `/x/y/z`, `noroot1`, 0, fmt.Errorf(`import "/x/y/z": cannot import absolute path`), &_Want{}},
		&_Case{4, `//x/y/z`, `noroot1`, 0, fmt.Errorf(`import "//x/y/z": cannot import absolute path`), &_Want{}},
		&_Case{5, `.`, `notexist`, 0, fmt.Errorf(`import ".": cannot find package at v:\notexist`), &_Want{}},
		&_Case{6, `.`, `__goroot__/src/notexist`, 0, fmt.Errorf(`import ".": cannot find package at v:\__goroot__\src\notexist`), &_Want{}},
		&_Case{7, `.`, `gopath1/src/notexist`, 0, fmt.Errorf(`import ".": cannot find package at v:\gopath1\src\notexist`), &_Want{}},
		&_Case{8, `#/testdata/local1`, `localroot1/src/testdata/local1`, 0, fmt.Errorf(`import "#/testdata/local1": cannot refer package under testdata`), &_Want{}},
		&_Case{9, `testdata/local1`, `localroot1/src/testdata/local1`, 0, fmt.Errorf(`import "testdata/local1": cannot refer package under testdata`), &_Want{}},
		&_Case{10, `go/build/testdata/vroot/noroot1`, `__goroot__`, 0, fmt.Errorf(`import "go/build/testdata/vroot/noroot1": cannot refer package under testdata`), &_Want{}},
		&_Case{11, `#/fmt`, `__goroot__/src/go`, 0, fmt.Errorf(`import "#/fmt": cannot find local-root(with sub-tree "<root>/src/vendor/") up from v:\__goroot__\src\go`), &_Want{}},
		&_Case{12, `#/fmt`, `localroot1/src/local1`, 0, fmt.Errorf(`import "#/fmt": cannot find sub-package under local-root v:\localroot1`), &_Want{}},
		&_Case{13, `#/xx`, `notexist`, 0, fmt.Errorf(`import "#/xx": cannot find local-root(with sub-tree "<root>/src/vendor/") up from v:\notexist`), &_Want{}},
		&_Case{14, `xx`, `notexist`, 0, fmt.Errorf(`cannot find package "xx" in any of:
	v:\__goroot__\src\xx (from $GOROOT)
	v:\gopath1\src\xx (from $GOPATH)
	v:\gopath2\src\xx
	v:\gopath3\src\xx`), &_Want{}},
		&_Case{15, `xx`, `gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\localrootv1\src\local1`, 0, fmt.Errorf(`cannot find package "xx" in any of:
	v:\gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\localrootv1\src\vendor\xx (vendor tree)
	v:\gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\xx
	v:\gopath1\src\localroot1\src\vendor\xx
	v:\gopath1\src\vendor\xx
	v:\__goroot__\src\xx (from $GOROOT)
	v:\gopath1\src\xx (from $GOPATH)
	v:\gopath2\src\xx
	v:\gopath3\src\xx
	v:\gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\localrootv1\src\xx (from #LocalRoot)`), &_Want{}},
		&_Case{16, `.`, `localroot1/src/testdata/local1`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\testdata\local1`, FmtImportPath: `v:\localroot1\src\testdata\local1`, ImportPath: `v:\localroot1\src\testdata\local1`, Signature: `_\v_\localroot1\src\testdata\local1`, Dir: `v:\localroot1\src\testdata\local1`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: ``, Type: PackageStandAlone, Style: ImportStyleSelf, IsVendor: false}},
		&_Case{17, `.`, `__goroot__/src/go/build/testdata/vroot/noroot1`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\__goroot__\src\go\build\testdata\vroot\noroot1`, FmtImportPath: `v:\__goroot__\src\go\build\testdata\vroot\noroot1`, ImportPath: `v:\__goroot__\src\go\build\testdata\vroot\noroot1`, Signature: `_\v_\__goroot__\src\go\build\testdata\vroot\noroot1`, Dir: `v:\__goroot__\src\go\build\testdata\vroot\noroot1`, LocalRoot: ``, ConflictDir: ``, Root: ``, Type: PackageStandAlone, Style: ImportStyleSelf, IsVendor: false}},
		&_Case{18, `.`, `localroot1/src/sole`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\sole`, FmtImportPath: `#/sole`, ImportPath: `sole`, Signature: `_\v_\localroot1\src\sole`, Dir: `v:\localroot1\src\sole`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{19, `sole`, `localroot1/src/sole`, 0, nil, &_Want{OriginImportPath: `sole`, ImporterDir: `v:\localroot1\src\sole`, FmtImportPath: `sole`, ImportPath: `sole`, Signature: `_\v_\localroot1\src\sole`, Dir: `v:\localroot1\src\sole`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleGlobal, IsVendor: false}},
		&_Case{20, `#/sole`, `localroot1/src/sole`, 0, nil, &_Want{OriginImportPath: `#/sole`, ImporterDir: `v:\localroot1\src\sole`, FmtImportPath: `#/sole`, ImportPath: `sole`, Signature: `_\v_\localroot1\src\sole`, Dir: `v:\localroot1\src\sole`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{21, `vendored`, `localroot1/src/sole`, 0, nil, &_Want{OriginImportPath: `vendored`, ImporterDir: `v:\localroot1\src\sole`, FmtImportPath: `vendored`, ImportPath: `vendor/vendored`, Signature: `_\v_\localroot1\src\vendor\vendored`, Dir: `v:\localroot1\src\vendor\vendored`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleGlobal, IsVendor: true}},
		&_Case{22, `#/vendored`, `localroot1/src/sole`, 0, nil, &_Want{OriginImportPath: `#/vendored`, ImporterDir: `v:\localroot1\src\sole`, FmtImportPath: `#/vendored`, ImportPath: `vendored`, Signature: `_\v_\localroot1\src\vendor\vendored`, Dir: `v:\localroot1\src\vendor\vendored`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{23, `.`, `localroot1/src/localrootv1/src/local1`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\localrootv1\src\local1`, FmtImportPath: `#/local1`, ImportPath: `local1`, Signature: `_\v_\localroot1\src\localrootv1\src\local1`, Dir: `v:\localroot1\src\localrootv1\src\local1`, LocalRoot: `v:\localroot1\src\localrootv1`, ConflictDir: ``, Root: `v:\localroot1\src\localrootv1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{24, `.`, `localroot1/src/vendor/localrootv1/src/local1`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\vendor\localrootv1\src\local1`, FmtImportPath: `#/local1`, ImportPath: `local1`, Signature: `_\v_\localroot1\src\vendor\localrootv1\src\local1`, Dir: `v:\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\localroot1\src\vendor\localrootv1`, ConflictDir: ``, Root: `v:\localroot1\src\vendor\localrootv1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{25, `#/localrootv1/src/local1`, `localroot1/src/local1`, 0, nil, &_Want{OriginImportPath: `#/localrootv1/src/local1`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `#/localrootv1/src/local1`, ImportPath: `localrootv1/src/local1`, Signature: `_\v_\localroot1\src\vendor\localrootv1\src\local1`, Dir: `v:\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{26, `localrootv1/src/local1`, `localroot1/src/local1`, 0, nil, &_Want{OriginImportPath: `localrootv1/src/local1`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `localrootv1/src/local1`, ImportPath: `vendor/localrootv1/src/local1`, Signature: `_\v_\localroot1\src\vendor\localrootv1\src\local1`, Dir: `v:\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Type: PackageLocalRoot, Style: ImportStyleGlobal, IsVendor: true}},
		&_Case{27, `.`, `gopath1/src/localroot1/src/localrootv1/src/local1`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath1\src\localroot1\src\localrootv1\src\local1`, FmtImportPath: `#/local1`, ImportPath: `local1`, Signature: `_\v_\gopath1\src\localroot1\src\localrootv1\src\local1`, Dir: `v:\gopath1\src\localroot1\src\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1\src\localrootv1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1\src\localrootv1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{28, `.`, `gopath1/src/localroot1/src/vendor/localrootv1/src/local1`, 0, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, FmtImportPath: `#/local1`, ImportPath: `local1`, Signature: `_\v_\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, Dir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1\src\vendor\localrootv1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1\src\vendor\localrootv1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{29, `#/localrootv1/src/local1`, `gopath1/src/localroot1/src/local1`, 0, nil, &_Want{OriginImportPath: `#/localrootv1/src/local1`, ImporterDir: `v:\gopath1\src\localroot1\src\local1`, FmtImportPath: `#/localrootv1/src/local1`, ImportPath: `localrootv1/src/local1`, Signature: `_\v_\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, Dir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1`, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, IsVendor: false}},
		&_Case{30, `localrootv1/src/local1`, `gopath1/src/localroot1/src/local1`, 0, nil, &_Want{OriginImportPath: `localrootv1/src/local1`, ImporterDir: `v:\gopath1\src\localroot1\src\local1`, FmtImportPath: `localrootv1/src/local1`, ImportPath: `vendor/localrootv1/src/local1`, Signature: `_\v_\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, Dir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1`, Type: PackageLocalRoot, Style: ImportStyleGlobal, IsVendor: true}},
		&_Case{31, `fmt`, `gopath1/src/local1`, 0, nil, &_Want{OriginImportPath: `fmt`, ImporterDir: `v:\gopath1\src\local1`, FmtImportPath: `fmt`, ImportPath: `fmt`, Signature: `fmt`, Dir: `v:\__goroot__\src\fmt`, LocalRoot: `v:\gopath1`, ConflictDir: ``, Root: `v:\__goroot__`, Type: PackageGoRoot, Style: ImportStyleGlobal, IsVendor: false}},
		&_Case{32, `cmd/compile`, `noroot1`, 0, nil, &_Want{OriginImportPath: `cmd/compile`, ImporterDir: `v:\noroot1`, FmtImportPath: `cmd/compile`, ImportPath: `cmd/compile`, Signature: `cmd/compile`, Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: ``, ConflictDir: ``, Root: `v:\__goroot__`, Type: PackageGoRoot, Style: ImportStyleGlobal, IsVendor: false}},
		&_Case{33, `cmd/compile`, `localroot1`, 0, nil, &_Want{OriginImportPath: `cmd/compile`, ImporterDir: `v:\localroot1`, FmtImportPath: `cmd/compile`, ImportPath: `cmd/compile`, Signature: `cmd/compile`, Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: ``, ConflictDir: ``, Root: `v:\__goroot__`, Type: PackageGoRoot, Style: ImportStyleGlobal, IsVendor: false}},
		&_Case{34, `cmd/compile`, `gopath1/src/local1`, 0, nil, &_Want{OriginImportPath: `cmd/compile`, ImporterDir: `v:\gopath1\src\local1`, FmtImportPath: `cmd/compile`, ImportPath: `cmd/compile`, Signature: `cmd/compile`, Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: `v:\gopath1`, ConflictDir: ``, Root: `v:\__goroot__`, Type: PackageGoRoot, Style: ImportStyleGlobal, IsVendor: false}},
		&_Case{35, `cmd/compile`, `localroot1/src/local1`, 0, nil, &_Want{OriginImportPath: `cmd/compile`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `cmd/compile`, ImportPath: `cmd/compile`, Signature: `cmd/compile`, Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\__goroot__`, Type: PackageGoRoot, Style: ImportStyleGlobal, IsVendor: false}},
	}
	for i, testCase := range testCases {
		var pp PackagePath
		dir := vdir(testCase.dir)
		err := pp.FindImport(&testContext, testCase.imported, dir, testCase.mode)

		if genCase := false; genCase { //genCase
			if err != nil {
				fmt.Printf("&_Case{%d, `%s`, `%s`, %d, fmt.Errorf(`%s`),&_Want{}},\n", i+1, testCase.imported, testCase.dir, testCase.mode, err.Error())
			} else {
				//fmt.Printf("%+v\n", pp)
				g := &pp
				fmt.Printf("&_Case{%d, `%s`, `%s`, %d, nil, &_Want{OriginImportPath:`%s`, ImporterDir:`%s`, FmtImportPath:`%s`, ImportPath:`%s`, Signature:`%s`, Dir:`%s`, LocalRoot:`%s`, ConflictDir:`%s`, Root:`%s`, Type:%s, Style:%s, IsVendor:%v}},\n",
					i+1, testCase.imported, testCase.dir, testCase.mode,
					g.OriginImportPath, g.ImporterDir, g.FmtImportPath, g.ImportPath, g.Signature, g.Dir, g.LocalRoot, g.ConflictDir, g.Root, g.Type, g.Style, g.IsVendor)
			}
		} else { //verify
			errEq := reflect.DeepEqual(err, testCase.wantErr)
			if testCase.wantErr != nil || !errEq {
				if !errEq {
					t.Errorf("FindImport [%d %q %s] wantErr=[%+v] gotErr: [%+v] \n[%+v]", i+1, testCase.imported, dir, testCase.wantErr, err, &pp)
				}
				continue
			}

			if !reflect.DeepEqual(&pp, testCase.want) {
				//fmt.Printf("ImportPath:\"%s\", Dir:`%s`, LocalRoot:`%s` ,ConflictDir:`%s`,Root:`%s`, Signature:`%s`, IsVendor:%v, Type:%v, Style:%v\n", pp.ImportPath, pp.Dir, pp.LocalRoot, pp.ConflictDir, pp.Root, pp.Signature, pp.IsVendor, pp.Type, pp.Style)
				t.Errorf("FindImport[%d %q %s] \n    want [%+v]\n     got [%+v]\n", testCase.id, testCase.imported, dir, testCase.want, &pp)
			} else {
				if showResult {
					t.Logf("%d FindImport(%q, \"%s\")=%+v err=%v\n", testCase.id, testCase.imported, dir, pp, err)
				}
			}
		}
	}
}

func TestTestdataRE(t *testing.T) {
	type _Case struct {
		dir   string
		match bool
	}
	testCases := []*_Case{
		&_Case{"testData", false},
		&_Case{"testdata", true},
		&_Case{"testdata\\", true},
		&_Case{"testdata/", true},
		&_Case{"x/testdata", true},
		&_Case{"x\\testdata", true},
		&_Case{"x\\testdata\\y", true},
	}
	for _, testCase := range testCases {
		match := testdataRE.MatchString(testCase.dir)
		if match != testCase.match {
			t.Errorf("testdataRE.match(\"%s\") fail, want %v got %v", testCase.dir, testCase.match, match)
		}
	}
}

func TestSrcRE(t *testing.T) {
	type _Case struct {
		dir  string
		root string
	}
	testCases := []*_Case{
		&_Case{`/x/Src/y`, ``},
		&_Case{`/x/srcs/y`, ``},
		&_Case{`/src/y`, ``},
		&_Case{`/x/src/y`, `/x`},
		&_Case{`\x/src\y`, `\x`},
		&_Case{`\x/src`, `\x`},
		&_Case{`c:\x\src\y/src\z\`, `c:\x\src\y`},
		&_Case{`c:\x\src\y\src\z\src`, `c:\x\src\y\src\z`},
		&_Case{`c:\x\src\y\src\z`, `c:\x\src\y`},
		&_Case{`c:\x\src\y`, `c:\x`},
		&_Case{`c:\x`, ``},
	}
	for _, testCase := range testCases {
		got := ""
		match := srcRE.FindAllStringSubmatch(testCase.dir, 1)
		if match != nil {
			got = match[0][1]
		}
		if got != testCase.root {
			t.Errorf("srcRE.match(\"%s\") fail, want %v got %v", testCase.dir, testCase.root, got)
		}
	}
}

func full(vdir string) string {
	if vdir == "" {
		return ""
	}
	if sub, ok := testContext.hasSubdir(testContext.GOROOT, vdir); ok {
		return testContext.joinPath(thisDir, "../../..", sub) //real goroot
	}
	return testContext.joinPath(fsRoot, strings.TrimPrefix(vdir, vroot)) //related to fsRoot
}

func vdir(related string) string {
	if related == "" {
		return ""
	}
	return testContext.joinPath(vroot, `/`, related)
}

func tvdir(s string) string {
	return "\t" + vdir(s)
}
