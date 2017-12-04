// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"fmt"
	"io"
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
		imported string
		dir      string
		wantErr  error
		want     *_Want
	}
	testCases := []*_Case{
		&_Case{"", "noroot1", fmt.Errorf(`import "%s": invalid import path`, ""), &_Want{}},
		&_Case{"/x/y/z", "noroot1", fmt.Errorf(`import "%s": cannot import absolute path`, "/x/y/z"), &_Want{}},
		&_Case{"//x/y/z", "noroot1", fmt.Errorf(`import "%s": cannot import absolute path`, "//x/y/z"), &_Want{}},
		&_Case{".", "notexist", fmt.Errorf(`import "%s": cannot find package at v:\notexist`, "."), &_Want{}},
		&_Case{".", "__goroot__/src/notexist", fmt.Errorf(`import "%s": cannot find package at %s`, ".", vdir(`__goroot__\src\notexist`)), &_Want{}},
		&_Case{".", "gopath1/src/notexist", fmt.Errorf(`import "%s": cannot find package at %s`, ".", vdir("gopath1/src/notexist")), &_Want{}},
		&_Case{".", "noroot1/testdata/local1", fmt.Errorf(`import "%s": cannot refer package under testdata %s`, ".", vdir(`noroot1\testdata\local1`)), &_Want{}},
		&_Case{".", "localroot1/src/testdata/local1", fmt.Errorf(`import ".": cannot refer package under testdata v:\localroot1\src\testdata\local1`), &_Want{}},

		&_Case{"#/x/y/z", "notexist", nil, &_Want{ImportPath: "#/x/y/z", Dir: vdir(``), Root: vdir(``), Type: PackageUnknown, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: false}},
		&_Case{"x/y/z", "notexist", nil, &_Want{ImportPath: "x/y/z", Dir: vdir(``), Root: vdir(``), Type: PackageUnknown, Style: ImportStyleGlobal, ConflictDir: "", Formated: false}},
		&_Case{".", "noroot1", nil, &_Want{ImportPath: "", Dir: vdir(`noroot1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleSelf, ConflictDir: "", Formated: true}},
		&_Case{".//local1", "noroot1", nil, &_Want{ImportPath: "", Dir: vdir(`noroot1\local1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleRelated, ConflictDir: "", Formated: true}},
		&_Case{"./local1", "noroot1", nil, &_Want{ImportPath: "", Dir: vdir(`noroot1\local1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleRelated, ConflictDir: "", Formated: true}},
		&_Case{"..", "noroot1/local1", nil, &_Want{ImportPath: "", Dir: vdir(`noroot1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleRelated, ConflictDir: "", Formated: true}},
		&_Case{".", "localroot1/src/local1", nil, &_Want{ImportPath: "#/local1", Dir: vdir(`localroot1\src\local1`), Root: vdir(`localroot1`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "gopath1/src/localroot1/src/local1", nil, &_Want{ImportPath: "#/local1", Dir: vdir(`gopath1\src\localroot1\src\local1`), Root: vdir(`gopath1\src\localroot1`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "gopath1/src/local1", nil, &_Want{ImportPath: "#/local1", Dir: vdir(`gopath1\src\local1`), Root: vdir(`gopath1`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "gopath2/src/local2", nil, &_Want{ImportPath: "local2", Dir: vdir(`gopath2\src\local2`), Root: vdir(`gopath2`), Type: PackageGoPath, Style: ImportStyleGlobal, ConflictDir: vdir(`gopath1\src\local2`), Formated: true}},
		&_Case{".", "gopath2/src/localroot2/src/local2", nil, &_Want{ImportPath: "#/local2", Dir: vdir(`gopath2\src\localroot2\src\local2`), Root: vdir(`gopath2\src\localroot2`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "__goroot__/src/fmt", nil, &_Want{ImportPath: "fmt", Dir: vdir(`__goroot__\src\fmt`), Root: vdir(`__goroot__`), Type: PackageGoRoot, Style: ImportStyleGlobal, ConflictDir: "", Formated: true}},
	}
	for i, testCase := range testCases {
		dir := vdir(testCase.dir)
		formated, err := testContext.FormatImportPath(testCase.imported, dir)

		//fmt.Printf("%d FormatImportPath(%q, %s)=%+v %v\n", i+1, testCase.imported, dir, formated, err)

		errEq := reflect.DeepEqual(err, testCase.wantErr)
		if testCase.wantErr != nil || !errEq {
			if !errEq {
				t.Errorf("FormatImportPath [%d %q %s] wantErr=[%+v] gotErr: [%+v]", i+1, testCase.imported, dir, testCase.wantErr, err)
			}
			continue
		}
		if !reflect.DeepEqual(&formated, testCase.want) {
			t.Errorf("FormatImportPath[%d %q %s] \n    want [%+v]\n     got [%+v]\n", i+1, testCase.imported, dir, testCase.want, &formated)
		}
	}
}

func TestFindImport(t *testing.T) {
	type _Want = PackagePath

	type _Case struct {
		imported string
		dir      string
		mode     ImportMode
		wantErr  error
		want     *_Want
	}
	testCases := []*_Case{
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

		&_Case{".", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "#/sole", Dir: vdir(`localroot1\src\sole`), Signature: `_\v_\localroot1\src\sole`, LocalRoot: vdir("localroot1"), Root: vdir("localroot1"), ConflictDir: "", IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		&_Case{"sole", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "#/sole", Dir: vdir(`localroot1\src\sole`), Signature: `_\v_\localroot1\src\sole`, LocalRoot: `v:\localroot1`, Root: `v:\localroot1`, ConflictDir: "", IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleGlobal}},

		//		&_Case{"#/sole", "localroot1/src/sole", 0, nil, &_Want{ImportPath: "#/sole", Dir: `v:\localroot1\src\sole`, Signature: "?", LocalRoot: `v:\localroot1`, Root: `v:\localroot1`, ConflictDir: "", IsVendor: false, Type: PackageLocalRoot, Style: ImportStyleLocalRoot}},
		//		&_Case{"vendored", "localroot1/src/sole", 0, nil, &_Want{}},
		//		&_Case{"#/vendored", "localroot1/src/sole", 0, nil, &_Want{}},

		//		&_Case{".", "localroot1/src/localrootv1/src/local1", 0, nil, &_Want{}},
		//		&_Case{".", "localroot1/src/vendor/localrootv1/src/local1", 0, nil, &_Want{}},
		//		&_Case{"#/localrootv1/src/local1", "localroot1/src/local1", 0, nil, &_Want{}},
		//		&_Case{"localrootv1/src/local1", "localroot1/src/local1", 0, nil, &_Want{}},

		//		&_Case{".", "gopath1/src/localroot1/src/localrootv1/src/local1", 0, nil, &_Want{}},
		//		&_Case{".", "gopath1/src/localroot1/src/vendor/localrootv1/src/local1", 0, nil, &_Want{}},
		//		&_Case{"#/localrootv1/src/local1", "gopath1/src/localroot1/src/local1", 0, nil, &_Want{}},
		//		&_Case{"localrootv1/src/local1", "gopath1/src/localroot1/src/local1", 0, nil, &_Want{}},

		//		&_Case{"fmt", "gopath1/src/local1", 0, nil, &_Want{}},
		//		&_Case{"cmd/compile", "noroot1", 0, nil, &_Want{}},
		//		&_Case{"cmd/compile", "localroot1", 0, nil, &_Want{}},
		//		&_Case{"cmd/compile", "gopath1/src/local1", 0, nil, &_Want{}},
		//		&_Case{"cmd/compile", "localroot1/src/local1", 0, nil, &_Want{}},
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
			t.Errorf("FindImport[%d %q %s] \n    want [%+v]\n     got [%+v]\n", i+1, testCase.imported, dir, testCase.want, &pp)
		}

		if false {
			fmt.Printf("%d FindImport(%q, %s)=%+v %v\n", i+1, testCase.imported, dir, pp, err)
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
