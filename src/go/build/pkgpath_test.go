package build

import (
	"fmt"
	"path/filepath"
	"testing"
)

var (
	fsRoot      = testContext.joinPath(getwd(), "testdata/fsroot")
	testContext = defaultContext()
)

func init() {
	testContext.GOROOT = testContext.joinPath(getwd(), "../../..")
	goRootSrc = testContext.joinPath(testContext.GOROOT, "src")
	testContext.GOPATH = fmt.Sprintf("%s%c%s%c%s", full("gopath1"), filepath.ListSeparator, full("gopath2"), filepath.ListSeparator, full("gopath3"))
	gblSrcs = testContext.SrcDirs()
}

func setWd(dir string) {
	wd = dir
}
func full(related string) string {
	return testContext.joinPath(fsRoot, related)
}

func TestA(t *testing.T) {}
