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
	genCase     = true
)

func init() {
	//testContext.GOOS = "linux"
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
	testContext.RefreshEnv()

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
		&_Case{1, `x/(y)/z`, `noroot1`, fmt.Errorf(`import "x/(y)/z": invalid character U+0028 '('`), &_Want{}},
		&_Case{2, `x/Programme Files/y`, `noroot1`, fmt.Errorf(`import "x/Programme Files/y": invalid character U+0020 ' '`), &_Want{}},
		&_Case{3, `#/#`, `noroot1`, fmt.Errorf(`import "#/#": invalid character U+0023 '#'`), &_Want{}},
		&_Case{4, `##`, `noroot1`, fmt.Errorf(`import "##": invalid character U+0023 '#'`), &_Want{}},
		&_Case{5, `c:/x/y/z`, `noroot1`, fmt.Errorf(`import "c:/x/y/z": invalid character U+003A ':'`), &_Want{}},
		&_Case{6, `./#/x/y/z`, `noroot1`, fmt.Errorf(`import "./#/x/y/z": invalid character U+0023 '#'`), &_Want{}},
		&_Case{7, `x\y\z`, `noroot1`, fmt.Errorf(`import "x\\y\\z": invalid character U+005C '\'`), &_Want{}},
		&_Case{8, `...`, `noroot1`, fmt.Errorf(`import "...": invalid import path`), &_Want{}},
		&_Case{9, `#/./x/y/z`, `noroot1`, fmt.Errorf(`import "#/./x/y/z": invalid import path`), &_Want{}},
		&_Case{10, `.../x/y/z`, `noroot1`, fmt.Errorf(`import ".../x/y/z": invalid import path`), &_Want{}},
		&_Case{11, ``, `noroot1`, fmt.Errorf(`import "": invalid import path`), &_Want{}},
		&_Case{12, `.//local1`, `noroot1`, fmt.Errorf(`import ".//local1": invalid import path`), &_Want{}},
		&_Case{13, `/x/y/z`, `noroot1`, fmt.Errorf(`import "/x/y/z": cannot import absolute path`), &_Want{}},
		&_Case{14, `//x/y/z`, `noroot1`, fmt.Errorf(`import "//x/y/z": cannot import absolute path`), &_Want{}},
		&_Case{15, `.`, `notexist`, fmt.Errorf(`import ".": cannot find package at v:\notexist`), &_Want{}},
		&_Case{16, `.`, `__goroot__/src/notexist`, fmt.Errorf(`import ".": cannot find package at v:\__goroot__\src\notexist`), &_Want{}},
		&_Case{17, `.`, `gopath1/src/notexist`, fmt.Errorf(`import ".": cannot find package at v:\gopath1\src\notexist`), &_Want{}},
		&_Case{18, `.`, `noroot1/testdata/local1`, fmt.Errorf(`import ".": cannot refer package under testdata v:\noroot1\testdata\local1`), &_Want{}},
		&_Case{19, `.`, `localroot1/src/testdata/local1`, fmt.Errorf(`import ".": cannot refer package under testdata v:\localroot1\src\testdata\local1`), &_Want{}},
		&_Case{20, `#/x/y/z`, `notexist`, nil, &_Want{OriginImportPath: `#/x/y/z`, ImporterDir: `v:\notexist`, FmtImportPath: `#/x/y/z`, Dir: ``, Root: ``, ConflictDir: ``, Type: PackageUnknown, Style: ImportStyleLocalRoot, Formated: false}},
		&_Case{21, `x/y/z`, `notexist`, nil, &_Want{OriginImportPath: `x/y/z`, ImporterDir: `v:\notexist`, FmtImportPath: `x/y/z`, Dir: ``, Root: ``, ConflictDir: ``, Type: PackageUnknown, Style: ImportStyleGlobal, Formated: false}},
		&_Case{22, `.`, `noroot1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\noroot1`, FmtImportPath: `v:\noroot1`, Dir: `v:\noroot1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleSelf, Formated: true}},
		&_Case{23, `./local1`, `noroot1`, nil, &_Want{OriginImportPath: `./local1`, ImporterDir: `v:\noroot1`, FmtImportPath: `v:\noroot1\local1`, Dir: `v:\noroot1\local1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleRelated, Formated: true}},
		&_Case{24, `..`, `noroot1/local1`, nil, &_Want{OriginImportPath: `..`, ImporterDir: `v:\noroot1\local1`, FmtImportPath: `v:\noroot1`, Dir: `v:\noroot1`, Root: ``, ConflictDir: ``, Type: PackageStandAlone, Style: ImportStyleRelated, Formated: true}},
		&_Case{25, `#`, `localroot1/src/local1`, nil, &_Want{OriginImportPath: `#`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `#`, Dir: ``, Root: ``, ConflictDir: ``, Type: PackageUnknown, Style: ImportStyleLocalRoot, Formated: false}},
		&_Case{26, `.`, `localroot1/src/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\localroot1\src\local1`, FmtImportPath: `#/local1`, Dir: `v:\localroot1\src\local1`, Root: `v:\localroot1`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{27, `.`, `gopath1/src/localroot1/src/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath1\src\localroot1\src\local1`, FmtImportPath: `#/local1`, Dir: `v:\gopath1\src\localroot1\src\local1`, Root: `v:\gopath1\src\localroot1`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{28, `.`, `gopath1/src/local1`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath1\src\local1`, FmtImportPath: `#/local1`, Dir: `v:\gopath1\src\local1`, Root: `v:\gopath1`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{29, `.`, `gopath2/src/local2`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath2\src\local2`, FmtImportPath: `local2`, Dir: `v:\gopath2\src\local2`, Root: `v:\gopath2`, ConflictDir: `v:\gopath1\src\local2`, Type: PackageGoPath, Style: ImportStyleGlobal, Formated: true}},
		&_Case{30, `.`, `gopath2/src/localroot2/src/local2`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\gopath2\src\localroot2\src\local2`, FmtImportPath: `#/local2`, Dir: `v:\gopath2\src\localroot2\src\local2`, Root: `v:\gopath2\src\localroot2`, ConflictDir: ``, Type: PackageLocalRoot, Style: ImportStyleLocalRoot, Formated: true}},
		&_Case{31, `.`, `__goroot__/src/fmt`, nil, &_Want{OriginImportPath: `.`, ImporterDir: `v:\__goroot__\src\fmt`, FmtImportPath: `fmt`, Dir: `v:\__goroot__\src\fmt`, Root: `v:\__goroot__`, ConflictDir: ``, Type: PackageGoRoot, Style: ImportStyleGlobal, Formated: true}},
	}
	for i, testCase := range testCases {
		dir := vdir(testCase.dir)
		formated, err := testContext.FormatImportPath(testCase.imported, dir)
		if false { //genCase
			if err != nil {
				fmt.Printf("&_Case{%d, `%s`, `%s`, fmt.Errorf(`%s`),&_Want{}},\n", i+1, testCase.imported, testCase.dir, err.Error())
			} else {
				//fmt.Printf("%+v\n", formated)
				g := &formated
				fmt.Printf("&_Case{%d, `%s`, `%s`, nil, &_Want{OriginImportPath:`%s`, ImporterDir:`%s`, FmtImportPath:`%s`, Dir:`%s`, Root:`%s`, ConflictDir:`%s`, Type:%s, Style:%s, Formated:%v}},\n",
					i+1, testCase.imported, testCase.dir,
					g.OriginImportPath, g.ImporterDir, g.FmtImportPath, g.Dir, g.Root, g.ConflictDir, g.Type, g.Style, g.Formated)
			}
		} else {
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

func _TestFindImport(t *testing.T) {
	type _Want = PackagePath

	type _Case struct {
		imported string
		dir      string
		mode     ImportMode
		wantErr  error
		want     *_Want
	}
	testCases := []*_Case{
		&_Case{"c:/go/src", "noroot1", 0, fmt.Errorf(``), &_Want{}},
		&_Case{"", "noroot1", 0, fmt.Errorf(`import "%s": invalid import path`, ""), &_Want{}},
		&_Case{"/x/y/z", "noroot1", 0, fmt.Errorf(`import "%s": cannot import absolute path`, "/x/y/z"), &_Want{}},
		&_Case{"//x/y/z", "noroot1", 0, fmt.Errorf(`import "%s": cannot import absolute path`, "//x/y/z"), &_Want{}},
		&_Case{".", "notexist", 0, fmt.Errorf(`import "%s": cannot find package at v:\notexist`, "."), &_Want{}},
		&_Case{".", "__goroot__/src/notexist", 0, fmt.Errorf(`import "%s": cannot find package at %s`, ".", vdir(`__goroot__\src\notexist`)), &_Want{}},
		&_Case{".", "gopath1/src/notexist", 0, fmt.Errorf(`import "%s": cannot find package at %s`, ".", vdir("gopath1/src/notexist")), &_Want{}},
		&_Case{".", "localroot1/src/testdata/local1", 0, fmt.Errorf(`import ".": cannot refer package under testdata v:\localroot1\src\testdata\local1`), &_Want{}},
		&_Case{"#/testdata/local1", "localroot1/src/testdata/local1", 0, fmt.Errorf(`import "%s": cannot refer package under testdata`, "#/testdata/local1"), &_Want{}},
		&_Case{"testdata/local1", "localroot1/src/testdata/local1", 0, fmt.Errorf(`import "%s": cannot refer package under testdata`, "testdata/local1"), &_Want{}},
		&_Case{".", "__goroot__/src/go/build/testdata/vroot/noroot1", 0, fmt.Errorf(`import ".": cannot refer package under testdata v:\__goroot__\src\go\build\testdata\vroot\noroot1`), &_Want{}},
		&_Case{"go/build/testdata/vroot/noroot1", "__goroot__", 0, fmt.Errorf(`import "go/build/testdata/vroot/noroot1": cannot refer package under testdata`), &_Want{}},
		&_Case{"#/fmt", "__goroot__/src/go", 0, fmt.Errorf(`import "#/fmt": cannot find local-root(with sub-tree "<root>/src/vendor/") up from v:\__goroot__\src\go`), &_Want{}},
		&_Case{"#/fmt", "localroot1/src/local1", 0, fmt.Errorf(`import "#/fmt": cannot find sub-package under local-root v:\localroot1`), &_Want{}},

		&_Case{"#/xx", "notexist", 0, fmt.Errorf(`import "%s": cannot find local-root(with sub-tree "<root>/src/vendor/") up from %s`, "#/xx", vdir("notexist")), &_Want{}},
		&_Case{"xx", "notexist", 0, fmt.Errorf("cannot find package %q in any of:\n%s", "xx", strings.Join([]string{
			tvdir(`__goroot__\src\xx (from $GOROOT)`),
			tvdir(`gopath1\src\xx (from $GOPATH)`),
			tvdir(`gopath2\src\xx`),
			tvdir(`gopath3\src\xx`),
		}, "\n")), &_Want{}},
		&_Case{"xx", `gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\localrootv1\src\local1`, 0, fmt.Errorf("cannot find package %q in any of:\n%s", "xx", strings.Join([]string{
			tvdir(`gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\localrootv1\src\vendor\xx (vendor tree)`),
			tvdir(`gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\xx`),
			tvdir(`gopath1\src\localroot1\src\vendor\xx`),
			tvdir(`gopath1\src\vendor\xx`),
			tvdir(`__goroot__\src\xx (from $GOROOT)`),
			tvdir(`gopath1\src\xx (from $GOPATH)`),
			tvdir(`gopath2\src\xx`),
			tvdir(`gopath3\src\xx`),
			tvdir(`gopath1\src\localroot1\src\vendor\localrootv1\src\vendor\localrootv1\src\xx (from #LocalRoot)`),
		}, "\n")), &_Want{}},

		&_Case{".", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "sole", Dir: vdir(`localroot1\src\sole`), Signature: `_\v_\localroot1\src\sole`, LocalRoot: vdir("localroot1"), Root: vdir("localroot1"), ConflictDir: "", IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"sole", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "sole", Dir: vdir(`localroot1\src\sole`), Signature: `_\v_\localroot1\src\sole`, LocalRoot: `v:\localroot1`, Root: `v:\localroot1`, ConflictDir: "", IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleGlobal}},
		&_Case{"#/sole", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "sole", Dir: `v:\localroot1\src\sole`, LocalRoot: `v:\localroot1`, ConflictDir: "", Root: `v:\localroot1`, Signature: `_\v_\localroot1\src\sole`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"vendored", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "vendor/vendored", Dir: `v:\localroot1\src\vendor\vendored`, LocalRoot: `v:\localroot1`, ConflictDir: "", Root: `v:\localroot1`, Signature: `_\v_\localroot1\src\vendor\vendored`, IsVendor: true, Type: PackageLocalRoot, Style: ImportStyleGlobal}},
		&_Case{"#/vendored", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "vendored", Dir: `v:\localroot1\src\vendor\vendored`, LocalRoot: `v:\localroot1`, ConflictDir: "", Root: `v:\localroot1`, Signature: `_\v_\localroot1\src\vendor\vendored`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{".", "localroot1/src/localrootv1/src/local1", 0, nil, &_Want{ImportPath: "local1", Dir: `v:\localroot1\src\localrootv1\src\local1`, LocalRoot: `v:\localroot1\src\localrootv1`, ConflictDir: "", Root: `v:\localroot1\src\localrootv1`, Signature: `_\v_\localroot1\src\localrootv1\src\local1`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{".", "localroot1/src/vendor/localrootv1/src/local1", 0, nil, &_Want{ImportPath: "local1", Dir: `v:\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\localroot1\src\vendor\localrootv1`, ConflictDir: "", Root: `v:\localroot1\src\vendor\localrootv1`, Signature: `_\v_\localroot1\src\vendor\localrootv1\src\local1`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"#/localrootv1/src/local1", "localroot1/src/local1", 0, nil, &_Want{ImportPath: "localrootv1/src/local1", Dir: `v:\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\localroot1`, ConflictDir: "", Root: `v:\localroot1`, Signature: `_\v_\localroot1\src\vendor\localrootv1\src\local1`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"localrootv1/src/local1", "localroot1/src/local1", 0, nil, &_Want{ImportPath: "vendor/localrootv1/src/local1", Dir: `v:\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\localroot1`, Signature: `_\v_\localroot1\src\vendor\localrootv1\src\local1`, IsVendor: true, Type: PackageLocalRoot, Style: ImportStyleGlobal}},
		&_Case{".", "gopath1/src/localroot1/src/localrootv1/src/local1", 0, nil, &_Want{ImportPath: "local1", Dir: `v:\gopath1\src\localroot1\src\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1\src\localrootv1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1\src\localrootv1`, Signature: `_\v_\gopath1\src\localroot1\src\localrootv1\src\local1`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{".", "gopath1/src/localroot1/src/vendor/localrootv1/src/local1", 0, nil, &_Want{ImportPath: "local1", Dir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1\src\vendor\localrootv1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1\src\vendor\localrootv1`, Signature: `_\v_\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"#/localrootv1/src/local1", "gopath1/src/localroot1/src/local1", 0, nil, &_Want{ImportPath: "localrootv1/src/local1", Dir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1`, Signature: `_\v_\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"localrootv1/src/local1", "gopath1/src/localroot1/src/local1", 0, nil, &_Want{ImportPath: "vendor/localrootv1/src/local1", Dir: `v:\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, LocalRoot: `v:\gopath1\src\localroot1`, ConflictDir: ``, Root: `v:\gopath1\src\localroot1`, Signature: `_\v_\gopath1\src\localroot1\src\vendor\localrootv1\src\local1`, IsVendor: true, Type: PackageLocalRoot, Style: ImportStyleGlobal}},
		&_Case{"fmt", "gopath1/src/local1", 0, nil, &_Want{ImportPath: "fmt", Dir: `v:\__goroot__\src\fmt`, LocalRoot: `v:\gopath1`, ConflictDir: ``, Root: `v:\__goroot__`, Signature: `fmt`, IsVendor: false, Type: PackageGoRoot, Style: ImportStyleGlobal}},
		&_Case{"cmd/compile", "noroot1", 0, nil, &_Want{ImportPath: "cmd/compile", Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: ``, ConflictDir: ``, Root: `v:\__goroot__`, Signature: `cmd/compile`, IsVendor: false, Type: PackageGoRoot, Style: ImportStyleGlobal}},
		&_Case{"cmd/compile", "localroot1", 0, nil, &_Want{ImportPath: "cmd/compile", Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: ``, ConflictDir: ``, Root: `v:\__goroot__`, Signature: `cmd/compile`, IsVendor: false, Type: PackageGoRoot, Style: ImportStyleGlobal}},
		&_Case{"cmd/compile", "gopath1/src/local1", 0, nil, &_Want{ImportPath: "cmd/compile", Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: `v:\gopath1`, ConflictDir: ``, Root: `v:\__goroot__`, Signature: `cmd/compile`, IsVendor: false, Type: PackageGoRoot, Style: ImportStyleGlobal}},
		&_Case{"cmd/compile", "localroot1/src/local1", 0, nil, &_Want{ImportPath: "cmd/compile", Dir: `v:\__goroot__\src\cmd\compile`, LocalRoot: `v:\localroot1`, ConflictDir: ``, Root: `v:\__goroot__`, Signature: `cmd/compile`, IsVendor: false, Type: PackageGoRoot, Style: ImportStyleGlobal}},
	}
	for i, testCase := range testCases {
		var pp PackagePath
		dir := vdir(testCase.dir)
		err := pp.FindImport(&testContext, testCase.imported, dir, testCase.mode)

		errEq := reflect.DeepEqual(err, testCase.wantErr)
		if testCase.wantErr != nil || !errEq {
			if !errEq {
				t.Errorf("FindImport [%d %q %s] wantErr=[%+v] gotErr: [%+v] \n[%+v]", i+1, testCase.imported, dir, testCase.wantErr, err, &pp)
			}
			continue
		}

		if !reflect.DeepEqual(&pp, testCase.want) {
			//fmt.Printf("ImportPath:\"%s\", Dir:`%s`, LocalRoot:`%s` ,ConflictDir:`%s`,Root:`%s`, Signature:`%s`, IsVendor:%v, Type:%v, Style:%v\n", pp.ImportPath, pp.Dir, pp.LocalRoot, pp.ConflictDir, pp.Root, pp.Signature, pp.IsVendor, pp.Type, pp.Style)
			t.Errorf("FindImport[%d %q %s] \n    want [%+v]\n     got [%+v]\n", i+1, testCase.imported, dir, testCase.want, &pp)
		} else {
			if showResult {
				t.Logf("%d FindImport(%q, \"%s\")=%+v err=%v\n", i+1, testCase.imported, dir, pp, err)
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

func setWd(dir string) {
	wd = vdir(dir)
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

func TestEnd(t *testing.T) {
	Default.RefreshEnv()
}
