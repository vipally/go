// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

//Ally:  refer local package by [import "#/x/y/z"] style

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	wd        = getwd()
	goRootSrc = filepath.Join(Default.GOROOT, "src")
)

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot determine current directory: " + err.Error())
	}
	return wd
}

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

// FormatImportPath convert "." "./x/y/z" "../x/y/z" style import path to "#/x/y/z" "x/y/z" style if possible.
func (ctxt *Context) FormatImportPath(path, srcDir string) (formated, root string, ok bool) {
	if path == "" || path[0] == '.' {
		if dir := ctxt.joinPath(srcDir, path); ctxt.isDir(dir) {
			if localRoot := ctxt.SearchLocalRoot(dir); localRoot != "" { //from local root
				localRootSrc := ctxt.joinPath(localRoot, "src")
				if sub, ok := ctxt.hasSubdir(localRootSrc, dir); ok {
					return "#/" + sub, localRoot, true
				}
			}

			if sub, ok := ctxt.hasSubdir(goRootSrc, dir); ok { //from GoRoot
				return sub, ctxt.GOROOT, true
			}

			gopaths := ctxt.gopath()
			for _, gopath := range gopaths { //from GoPath
				gopathsrc := ctxt.joinPath(gopath, "src")
				if sub, ok := ctxt.hasSubdir(gopathsrc, dir); ok {
					return sub, gopath, true
				}
			}
		}
	}
	return path, srcDir, false
}

func (ctxt *Context) SearchFromLocalRoot(imported, curPath string) (fullPath string) {
	return ""
}

func (ctxt *Context) SearchFromVendorPath(imported, curPath string) (fullPath string) {
	//find any parent dir that contains "vendor" dir
	for dir, lastDir := filepath.Clean(curPath), ""; dir != lastDir; dir, lastDir = filepath.Dir(dir), dir {
		if vendor := ctxt.joinPath(dir, "vendor"); ctxt.isDir(vendor) {
			if vendored := ctxt.joinPath(vendor, imported); ctxt.isDir(vendored) && hasGoFiles(ctxt, vendored) {
				return vendored
			}
		}

	}
	return ""
}

func (ctxt *Context) SearchFromGoRoot(imported string) string {
	if dir := filepath.Join(goRootSrc, imported); ctxt.isDir(dir) /*&& hasGoFiles(ctxt, dir)*/ {
		return dir
	}
	return ""
}

func (ctxt *Context) SearchFromGoPath(imported string) string {
	gopath := ctxt.gopath()
	for _, root := range gopath {
		if dir := ctxt.joinPath(root, "src", imported); ctxt.isDir(dir) && hasGoFiles(ctxt, dir) {
			return dir
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

// ImportStyle represents style of a package imort
type ImportStyle uint8

const (
	ImportStyleUnknown   ImportStyle = iota
	ImportStyleSelf                  //import "."
	ImportStyleRelated               //import "./x/y/z" "../x/y/z"
	ImportStyleLocalRoot             //import "#/x/y/z" "#"
	ImportStyleGlobal                //import "x/y/z"
)

func (st ImportStyle) String() string {
	switch st {
	case ImportStyleSelf:
		return "ImportStyleSelf"
	case ImportStyleRelated:
		return "ImportStyleRelated"
	case ImportStyleLocalRoot:
		return "ImportStyleLocalRoot"
	case ImportStyleGlobal:
		return "ImportStyleGlobal"
	}
	return "ImportStyleUnknown"
}

func (st ImportStyle) IsValid() bool     { return st >= ImportStyleSelf && st <= ImportStyleGlobal }
func (st ImportStyle) IsSelf() bool      { return st == ImportStyleSelf }
func (st ImportStyle) IsRelated() bool   { return st == ImportStyleRelated }
func (st ImportStyle) IsLocalRoot() bool { return st == ImportStyleLocalRoot }
func (st ImportStyle) IsGlobal() bool    { return st == ImportStyleGlobal }

func GetImportStyle(imported string) ImportStyle {
	if imported != "" {
		switch lead := imported[0]; {
		case lead == '.':
			if len(imported) == 1 {
				return ImportStyleSelf
			} else {
				return ImportStyleRelated
			}
		case lead == '#':
			return ImportStyleLocalRoot
		case lead == '/' || lead == '\\':
			return ImportStyleUnknown
		default:
			return ImportStyleGlobal
		}
	}
	return ImportStyleUnknown
}

// PackageType represents type of a imported package
type PackageType uint8

const (
	PackageUnknown    PackageType = iota //unknown, invalid
	PackageStandAlone                    //import "./../xx" style, out of GoPath/GoRoot/LocalRoot
	PackageLocalRoot                     //import "#/x/y/z" style
	PackageVendor                        //import "x/y/z" style, find from vendor tree
	PackageGoRoot                        //import "x/y/z" style, find from GoRoot
	PackageGoPath                        //import "x/y/z" style, find from GoPath
)

func (t PackageType) IsValid() bool             { return t >= PackageStandAlone && t <= PackageGoPath }
func (t PackageType) IsStandAlonePackage() bool { return t == PackageStandAlone }
func (t PackageType) IsLocalPackage() bool      { return t == PackageLocalRoot }
func (t PackageType) IsStdPackage() bool        { return t == PackageGoRoot }
func (t PackageType) IsGlobalPackage() bool     { return t == PackageGoPath }
func (t PackageType) IsVendoredPackage() bool   { return t == PackageVendor }

func (t PackageType) String() string {
	switch t {
	case PackageStandAlone:
		return "StandAlone"
	case PackageVendor:
		return "Vendor"
	case PackageLocalRoot:
		return "LocalRoot"
	case PackageGoRoot:
		return "GoRoot"
	case PackageGoPath:
		return "GoPath"
	}
	return "Unknown"
}

func (t PackageType) Signature() string {
	return ""
}

// PackagePath represent path information of a package
type PackagePath struct {
	ImportPath  string      // Regular original import path like: "x/y/z" "#/x/y/z" "." "../foo" "#"
	Dir         string      // Dir of imported package
	Root        string      // Root of imported package
	LocalRoot   string      // LocalRoot of imported package
	Signature   string      // Signature of imported package, which is unique for every package Dir
	ConflictDir string      // this directory shadows Dir in $GOPATH
	Type        PackageType // Type of this package
}

func (p *PackagePath) Init() {
	p.ImportPath = ""
	p.Dir = ""
	p.Root = ""
	p.Signature = ""
	p.Type = 0
}

func (p *PackagePath) ParseImportFromWd(ctxt *Context, imported string) error {
	return p.ParseImport(ctxt, imported, wd)
}

func (p *PackagePath) ParseImport(ctxt *Context, imported, srcDir string) error {
	p.ImportPath = imported
	if imported == "" {
		return fmt.Errorf("import %q: invalid import path", imported)
	}
	if imported[0] == '/' {
		return fmt.Errorf("import %q: cannot import absolute path", imported)
	}

	switch {
	case IsLocalRootBasedImport(imported): //import "#/x/y/z" style
		localRoot := ctxt.SearchLocalRoot(srcDir)
		if localRoot == "" {
			return fmt.Errorf(`import %q: cannot find local-root(with sub-tree "<root>/src/vendor") up from %s`, imported, srcDir)
		}
		p.Type = PackageLocalRoot
		p.LocalRoot = localRoot
		p.Root = localRoot
		p.ImportPath = imported
		p.Dir = ctxt.joinPath(localRoot, GetLocalRootRelatedImportPath(imported))
	case IsLocalImport(imported): //import "./../foo" style
		full := filepath.Join(srcDir, imported)
		p.Dir = full
		if sub, ok := ctxt.hasSubdir(goRootSrc, full); ok {
			p.Type = PackageGoRoot
			p.ImportPath = sub
			p.Root = ctxt.GOROOT
		}
	default: //import "x/y/z" style
	}
	if p.LocalRoot == "" {
		p.LocalRoot = ctxt.SearchLocalRoot(p.Dir)
	}

	return nil
}

// GetSignature returns signature of a package
// which is unique for every dir
func (p *PackagePath) GetSignature(path, srcDir string) (string, error) {
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
	return "", nil
}

func (info *PackagePath) Find(path, srcDir string) error {
	return nil
}
