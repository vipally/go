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
	thidDir     = getwd()
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
		wantErr  bool
		want     *_Want
	}
	testCases := []*_Case{
		&_Case{"", "noroot1", true, &_Want{}},
		&_Case{"/x/y/z", "noroot1", true, &_Want{}},
		&_Case{"//x/y/z", "noroot1", true, &_Want{}},
		&_Case{".", "notexist", true, &_Want{}},
		&_Case{".", "__goroot__/src/notexist", true, &_Want{}},
		&_Case{".", "gopath1/src/notexist", true, &_Want{}},

		&_Case{"#/x/y/z", "notexist", false, &_Want{ImportPath: "#/x/y/z", Dir: vdir(``), Root: vdir(``), Type: PackageUnknown, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: false}},
		&_Case{"x/y/z", "notexist", false, &_Want{ImportPath: "x/y/z", Dir: vdir(``), Root: vdir(``), Type: PackageUnknown, Style: ImportStyleGlobal, ConflictDir: "", Formated: false}},
		&_Case{".", "noroot1", false, &_Want{ImportPath: "", Dir: vdir(`noroot1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleSelf, ConflictDir: "", Formated: true}},
		&_Case{".//local1", "noroot1", false, &_Want{ImportPath: "", Dir: vdir(`noroot1\local1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleRelated, ConflictDir: "", Formated: true}},
		&_Case{"./local1", "noroot1", false, &_Want{ImportPath: "", Dir: vdir(`noroot1\local1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleRelated, ConflictDir: "", Formated: true}},
		&_Case{"..", "noroot1/local1", false, &_Want{ImportPath: "", Dir: vdir(`noroot1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleRelated, ConflictDir: "", Formated: true}},
		&_Case{".", "noroot1/testdata/local1", false, &_Want{ImportPath: "", Dir: vdir(`noroot1\testdata\local1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleSelf, ConflictDir: "", Formated: true}},
		&_Case{".", "localroot1/src/testdata/local1", false, &_Want{ImportPath: "", Dir: vdir(`localroot1\src\testdata\local1`), Root: vdir(``), Type: PackageStandAlone, Style: ImportStyleSelf, ConflictDir: "", Formated: true}},
		&_Case{".", "localroot1/src/local1", false, &_Want{ImportPath: "#/local1", Dir: vdir(`localroot1\src\local1`), Root: vdir(`localroot1`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "gopath1/src/localroot1/src/local1", false, &_Want{ImportPath: "#/local1", Dir: vdir(`gopath1\src\localroot1\src\local1`), Root: vdir(`gopath1\src\localroot1`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "gopath1/src/local1", false, &_Want{ImportPath: "#/local1", Dir: vdir(`gopath1\src\local1`), Root: vdir(`gopath1`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "gopath2/src/local2", false, &_Want{ImportPath: "local2", Dir: vdir(`gopath2\src\local2`), Root: vdir(`gopath2`), Type: PackageGoPath, Style: ImportStyleGlobal, ConflictDir: vdir(`gopath1\src\local2`), Formated: true}},
		&_Case{".", "gopath2/src/localroot2/src/local2", false, &_Want{ImportPath: "#/local2", Dir: vdir(`gopath2\src\localroot2\src\local2`), Root: vdir(`gopath2\src\localroot2`), Type: PackageLocalRoot, Style: ImportStyleLocalRoot, ConflictDir: "", Formated: true}},
		&_Case{".", "__goroot__/src/fmt", false, &_Want{ImportPath: "fmt", Dir: vdir(`__goroot__\src\fmt`), Root: vdir(`__goroot__`), Type: PackageGoRoot, Style: ImportStyleGlobal, ConflictDir: "", Formated: true}},
	}
	for i, testCase := range testCases {
		dir := vdir(testCase.dir)
		formated, err := testContext.FormatImportPath(testCase.imported, dir)
		gotErr := err != nil

		//fmt.Printf("%d FormatImportPath(%q, %s)=%+v %v\n", i, testCase.imported, dir, formated, err)

		if testCase.wantErr || gotErr != testCase.wantErr {
			if gotErr != testCase.wantErr {
				t.Errorf("FormatImportPath [%d %q %s] wantErr=%v gotErr: [%+v]", i+1, testCase.imported, dir, testCase.wantErr, err)
			}
			continue
		}
		if !reflect.DeepEqual(&formated, testCase.want) {
			fmt.Printf("FormatImportPath[%d %q %s] \n    want [%+v]\n     got [%+v]\n", i+1, testCase.imported, dir, testCase.want, &formated)
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
		return testContext.joinPath(thidDir, "../../..", sub) //real goroot
	}
	return testContext.joinPath(fsRoot, strings.TrimPrefix(vdir, vroot)) //related to fsRoot
}

func vdir(related string) string {
	if related == "" {
		return ""
	}
	return testContext.joinPath(vroot, `/`, related)
}
