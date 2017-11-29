// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"path/filepath"
	"regexp"
)

//Ally: import local package by "#/xxx" style

// match "<root>/src/..." case to find <root>
var srcRE = regexp.MustCompile(`(^.+)[\\|/]src(?:$|\\|/)`)

// SearchLocalRoot find the <root> path that contains such patten of sub-tree "<root>/src/vendor/" up from curPath,
// which is the root of local project.
// Actually, a LocalRoot is a private GoPath that is accessible to sub-packages only.
// It returns "" if not found.
// The expected working tree of LocalRoot is:
//	LocalRoot
//	│
//	├─bin
//	├─pkg
//	└─src
//	    │  ...
//	    │
//	    ├─vendor
//	    │      ...
//	    └─...
func (ctxt *Context) SearchLocalRoot(curPath string) string {
	dir := curPath
	withSrc := ""
	for {
		// if dir = `c:\root\src\prj\src\main`
		// match[0 ]= []string{"c:\\root\\src\\prj\\src\\", "c:\\root\\src\\prj"}
		if match := srcRE.FindAllStringSubmatch(dir, 1); match != nil {
			withSrc, dir = match[0][0], match[0][1]
			if vendor := ctxt.joinPath(withSrc, "vendor"); ctxt.isDir(vendor) {
				return filepath.Clean(dir)
			}
		} else {
			break
		}
	}
	return ""
}

// GetLocalRootRelPath joins localRootPath and rootBasedPath
// rootBasedPath must format as "#/foo"
func GetLocalRootRelatedPath(localRootPath, rootBasedPath string) string {
	if IsLocalRootBasedImport(rootBasedPath) {
		relPath := GetLocalRootRelatedImportPath(rootBasedPath)
		return filepath.ToSlash(filepath.Join(localRootPath, relPath))
	}
	return rootBasedPath
}

// GetLocalRootRelatedImportPath conver #/x/y/z to x/y/z
func GetLocalRootRelatedImportPath(imported string) string {
	//Ally:import "#/foo" is valid style
	if len(imported) > 0 && imported[0] == '#' {
		imported = imported[1:]
		if len(imported) > 0 && imported[0] == '/' {
			imported = imported[1:]
		}
	}
	if len(imported) == 0 {
		imported = "."
	}
	return imported
}

// IsValidImport verify if imported is a valid import string
// #/... style is valid.
//func IsValidImport(imported string) bool {
//	return parser.IsValidImport(imported)
//}

// ValidateImportPath returns Unquote of path if valid
//func ValidateImportPath(path string) (string, error) {
//	s, err := strconv.Unquote(path)
//	if err != nil {
//		return "", err
//	}
//	if s == "" {
//		return "", fmt.Errorf("empty string")
//	}
//	sCheck := s
//	if len(sCheck) > 2 && sCheck[:2] == "#/" { //Ally:import "#/foo" is valid style
//		sCheck = sCheck[2:]
//	}
//	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
//	for _, r := range sCheck {
//		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
//			return s, fmt.Errorf("invalid character %#U", r)
//		}
//	}
//	return s, nil
//}

// IsLocalRootBasedImport reports whether the import path is
// a local root related import path, like "#/foo"
// "#"will be replaced with which contains sub-directory "vendor" up from current package path.
func IsLocalRootBasedImport(path string) bool {
	localStyle := len(path) > 2 && path[:2] == "#/" || path == "#"
	return localStyle
}

//// dirToImportPath returns the pseudo-import path we use for a package
//// outside the Go path. It begins with _/ and then contains the full path
//// to the directory. If the package lives in c:\home\gopher\my\pkg then
//// the pseudo-import path is _/c_/home/gopher/my/pkg.
//// Using a pseudo-import path like this makes the ./ imports no longer
//// a special case, so that all the code to deal with ordinary imports works
//// automatically.
//func dirToImportPath(dir string) string {
//	return pathpkg.Join("_", strings.Map(makeImportValid, filepath.ToSlash(dir)))
//}

//func makeImportValid(r rune) rune {
//	// Should match Go spec, compilers, and ../../go/parser/parser.go:/isValidImport.
//	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
//	if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
//		return '_'
//	}
//	return r
//}

// VendoredImportPath returns the expansion of path when it appears in parent.
// If parent is x/y/z, then path might expand to x/y/z/vendor/path, x/y/vendor/path,
// x/vendor/path, vendor/path, or else stay path if none of those exist.
// VendoredImportPath returns the expanded path or, if no expansion is found, the original.
//func VendoredImportPath(parent *Package, path string) (found string) {
//	if parent == nil || parent.Root == "" {
//		return path
//	}

//	dir := filepath.Clean(parent.Dir)
//	root := filepath.Join(parent.Root, "src")
//	if !hasFilePathPrefix(dir, root) || parent.ImportPath != "command-line-arguments" && filepath.Join(root, parent.ImportPath) != dir {
//		// Look for symlinks before reporting error.
//		dir = expandPath(dir)
//		root = expandPath(root)
//	}

//	// Fix #22863: main package in GoPath/src/ runs "go install" fail.
//	// see: https://github.com/golang/go/issues/22863
//	// When path="GoPath/src", dir==root, it will always fail but not expected.
//	if dir != root && parent.Internal.LocalRoot != "" && !parent.Internal.Local {
//		if !hasFilePathPrefix(dir, root) || len(dir) <= len(root) || dir[len(root)] != filepath.Separator || parent.ImportPath != "command-line-arguments" && !parent.Internal.Local && filepath.Join(root, parent.ImportPath) != dir {
//			base.Fatalf("unexpected directory layout:\n"+
//				"	import path: %s\n"+
//				"	root: %s\n"+
//				"	dir: %s\n"+
//				"	expand root: %s\n"+
//				"	expand dir: %s\n"+
//				"	separator: %s",
//				parent.ImportPath,
//				filepath.Join(parent.Root, "src"),
//				filepath.Clean(parent.Dir),
//				root,
//				dir,
//				string(filepath.Separator))
//			//fmt.Printf("VendoredImportPath path=%s\nparent=%#v\n", path, parent)
//			//panic("check fail")
//		}
//	}

//	vpath := "vendor/" + path
//	for i := len(dir); i >= len(root); i-- {
//		if i < len(dir) && dir[i] != filepath.Separator {
//			continue
//		}
//		// Note: checking for the vendor directory before checking
//		// for the vendor/path directory helps us hit the
//		// isDir cache more often. It also helps us prepare a more useful
//		// list of places we looked, to report when an import is not found.
//		if !isDir(filepath.Join(dir[:i], "vendor")) {
//			continue
//		}
//		targ := filepath.Join(dir[:i], vpath)
//		if isDir(targ) && hasGoFiles(targ) {
//			importPath := parent.ImportPath
//			if importPath == "command-line-arguments" {
//				// If parent.ImportPath is 'command-line-arguments'.
//				// set to relative directory to root (also chopped root directory)
//				importPath = dir[len(root)+1:]
//			}
//			// We started with parent's dir c:\gopath\src\foo\bar\baz\quux\xyzzy.
//			// We know the import path for parent's dir.
//			// We chopped off some number of path elements and
//			// added vendor\path to produce c:\gopath\src\foo\bar\baz\vendor\path.
//			// Now we want to know the import path for that directory.
//			// Construct it by chopping the same number of path elements
//			// (actually the same number of bytes) from parent's import path
//			// and then append /vendor/path.
//			chopped := len(dir) - i
//			if chopped == len(importPath)+1 {
//				// We walked up from c:\gopath\src\foo\bar
//				// and found c:\gopath\src\vendor\path.
//				// We chopped \foo\bar (length 8) but the import path is "foo/bar" (length 7).
//				// Use "vendor/path" without any prefix.
//				return vpath
//			}
//			return importPath[:len(importPath)-chopped] + "/" + vpath
//		}
//	}
//	return path
//}

// PackageType represents type of a imported package
type PackageType uint

const (
	PackageStandAlone PackageType = iota //import "./../xx" style, invalid
	PackageLocalRoot                     //import "#/x/y/z" style
	PackageGlobal                        //import "x/y/z" style, not find yet
	PackageVendor                        //import "x/y/z" style, find from vendor tree
	PackageGoRoot                        //import "x/y/z" style, find from GoRoot
	PackageGoPath                        //import "x/y/z" style, find from GoPath
)

func (t PackageType) String() string {
	switch t {
	case PackageStandAlone:
		return "StandAlone"
	case PackageVendor:
		return "Vendor"
	case PackageLocalRoot:
		return "LocalRoot"
	case PackageGlobal:
		return "Global"
	case PackageGoRoot:
		return "GoRoot"
	case PackageGoPath:
		return "GoPath"
	}
	return "unknown"
}

type PackageInfo struct {
	ImportPath string //original import path: "x/y/z" "#/x/y/z"
	ImportDir  string //Dir that import this package
	Dir        string //Dir of imported package
	Root       string //Root of imported package
	Signature  string //Signature of imported package, to
	Type       PackageType
}

// Search package information by path and srcDir
// import "fmt"
func (info *PackageInfo) GetSignature(path, srcDir string) error {
	//	searchLocalRoot := func() string {
	//		localRoot := ""
	//		if parent != nil {
	//			localRoot = parent.Internal.LocalRoot
	//		} else {
	//			localRoot = cfg.BuildContext.SearchLocalRoot(srcDir)
	//		}
	//		return localRoot
	//	}
	//
	//	isLocal := build.IsLocalImport(path)
	//	localRoot := searchLocalRoot()
	//	isLocalRootRelated := localRoot != "" && localRoot != cfg.BuildContext.GOROOT
	//	var debugDeprecatedImportcfgDir string
	//	if isLocalRootRelated {
	//		importPath = build.GetLocalRootRelatedPath(localRoot, path)
	//		if importPath == "." {
	//			importPath = srcDir
	//		}
	//		if strings.HasPrefix(importPath, localRoot) {
	//			importPath = dirToImportPath(importPath)
	//		}
	//	} else if isLocal {
	//		importPath = dirToImportPath(filepath.Join(srcDir, path))
	//	} else if DebugDeprecatedImportcfg.enabled {
	//		if d, i := DebugDeprecatedImportcfg.lookup(parent, path); d != "" {
	//			debugDeprecatedImportcfgDir = d
	//			importPath = i
	//		}
	//	} else if mode&UseVendor != 0 {
	//		// We do our own vendor resolution, because we want to
	//		// find out the key to use in packageCache without the
	//		// overhead of repeated calls to buildContext.Import.
	//		// The code is also needed in a few other places anyway.
	//		path = VendoredImportPath(parent, path)
	//		importPath = path
	//	}
	return nil
}

func (info *PackageInfo) Find(path, srcDir string) error {
	return nil
}
