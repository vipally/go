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
	showResult  = true
)

func init() {
	//testContext.GOOS = "linux"
	if testContext.GOOS == "windows" {
		vroot = `v:`
	}
	testContext.IsDir = func(vdir string) bool {
		dir := full(vdir)
		fi, err := os.Stat(dir)
		return err == nil && fi.IsDir()
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
		[]string{"__goroot__/src", "__goroot__"},
		[]string{"__goroot__/src/fmt", "__goroot__"},
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
