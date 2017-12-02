// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

//Ally:  refer local package by [import "#/x/y/z"] style

import (
	"fmt"
	"go/parser"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	wd        = getwd()
	goRootSrc = filepath.Join(Default.GOROOT, "src")
	gblSrcs   = Default.SrcDirs()
)

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

type FormatImport struct {
	ImportPath string
	Dir        string
	Root       string
	Type       PackageType
	Style      ImportStyle
	Formated   bool
}

// FormatImportPath convert "." "./x/y/z" "../x/y/z" style import path to "#/x/y/z" "x/y/z" style if possible.
func (ctxt *Context) FormatImportPath(imported, importerDir string) (formated FormatImport, err error) {
	formated.ImportPath = imported
	formated.Type = PackageUnknown

	if formated.Style, err = GetImportStyle(imported); err != nil {
		return
	}

	if style := formated.Style; style.IsRelated() { //import "./xxx"
		if importerDir == "" {
			err = fmt.Errorf("import %q: import relative to unknown directory", imported)
			return
		}
		if dir := ctxt.joinPath(importerDir, imported); ctxt.isDir(dir) {
			formated.Dir = dir
			formated.Formated = true

			if localRoot := ctxt.SearchLocalRoot(dir); localRoot != "" { //from local root
				localRootSrc := ctxt.joinPath(localRoot, "src")
				if sub, ok_ := ctxt.hasSubdir(localRootSrc, dir); ok_ {
					formated.ImportPath = "#/" + sub
					formated.Root = localRoot
					formated.Type = PackageLocalRoot
					formated.Style = ImportStyleLocalRoot
					return
				}
			}

			if sub, ok_ := ctxt.hasSubdir(goRootSrc, dir); ok_ { //from GoRoot
				formated.ImportPath = sub
				formated.Root = ctxt.GOROOT
				formated.Type = PackageGoRoot
				formated.Style = ImportStyleGlobal
				return
			}

			gopaths := ctxt.gopath()
			for _, gopath := range gopaths { //from GoPath
				gopathsrc := ctxt.joinPath(gopath, "src")
				if sub, ok_ := ctxt.hasSubdir(gopathsrc, dir); ok_ {
					formated.ImportPath = sub
					formated.Root = gopath
					formated.Type = PackageGoPath
					formated.Style = ImportStyleGlobal
					return
				}
			}

			//StandAlone package out of LocalPath/GoRoot/GoPath
			formated.ImportPath = imported
			formated.Type = PackageStandAlone
		} else {
			err = fmt.Errorf("import %q: cannot find package at %s", imported, dir)
			return
		}
	}
	return
}

func (ctxt *Context) searchFromLocalRoot(imported, curPath string) (fullPath string) {
	return ""
}

func (ctxt *Context) searchFromVendorPath(imported, curPath string) (fullPath string) {
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

func (ctxt *Context) searchFromGoRoot(imported string) string {
	if dir := filepath.Join(goRootSrc, imported); ctxt.isDir(dir) /*&& hasGoFiles(ctxt, dir)*/ {
		return dir
	}
	return ""
}

func (ctxt *Context) searchFromGoPath(imported string) string {
	gopaths := ctxt.gopath()
	for _, gopath := range gopaths {
		if dir := ctxt.joinPath(gopath, "src", imported); ctxt.isDir(dir) && hasGoFiles(ctxt, dir) {
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

// IsLocalRootBasedImport reports whether the import path is
// a local root related import path, like "#/foo"
// "#"will be replaced with which contains sub-directory "vendor" up from current package path.
func IsLocalRootBasedImport(path string) bool {
	localStyle := len(path) > 2 && path[:2] == "#/" || path == "#"
	return localStyle
}

// ImportStyle represents style of a package import statement
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
func (st ImportStyle) IsRelated() bool   { return st.IsSelf() || st == ImportStyleRelated }
func (st ImportStyle) IsLocalRoot() bool { return st == ImportStyleLocalRoot }
func (st ImportStyle) IsGlobal() bool    { return st == ImportStyleGlobal }

// RealImportPath returns real related path to Root
// Especially convert "#/x/y/z" to "x/y/z"
func (st ImportStyle) RealImportPath(imported string) string {
	formated := imported
	switch st {
	case ImportStyleLocalRoot: //conver #/x/y/z to x/y/z
		if len(formated) > 0 && formated[0] == '#' {
			formated = formated[1:]
			if len(formated) > 0 && formated[0] == '/' {
				formated = formated[1:]
			}
		} else {
			panic(imported)
		}
		if len(formated) == 0 {
			formated = "."
		}
	}
	return formated
}

func (st ImportStyle) FullImportPath(imported, root string) string {
	realImportPath := st.RealImportPath(imported)
	return filepath.Join(root, realImportPath)
}

func GetImportStyle(imported string) (ImportStyle, error) {
	if imported == "" || !parser.IsValidImport(imported) {
		return ImportStyleUnknown, fmt.Errorf("import %q: invalid import path", imported)
	}
	if imported[0] == '/' {
		return ImportStyleUnknown, fmt.Errorf("import %q: cannot import absolute path", imported)
	}

	switch lead := imported[0]; {
	case lead == '.':
		if len(imported) == 1 {
			return ImportStyleSelf, nil
		} else {
			if imported == ".." || strings.HasPrefix(imported, "./") || strings.HasPrefix(imported, "../") {
				return ImportStyleRelated, nil
			}
		}
	case lead == '#':
		if imported == "#" || strings.HasPrefix(imported, "#/") {
			return ImportStyleLocalRoot, nil
		}
	default:
		return ImportStyleGlobal, nil
	}

	return ImportStyleUnknown, fmt.Errorf("import %q: invalid import path", imported)
}

// IsValidImportStatement return if a import path in statement is valid
// import "./xxx" "../xxx" is not allowed in statement
func CheckImportStatement(imported string) error {
	if style, err := GetImportStyle(imported); err == nil {
		if style.IsRelated() || style.IsSelf() {
			return fmt.Errorf("import %q: related import not allowed in statement", imported)
		}
	} else {
		return err
	}
	return nil
}

// PackageType represents type of a imported package
type PackageType uint8

const (
	PackageUnknown    PackageType = iota //unknown, invalid
	PackageStandAlone                    //import "./../xx" style, which is out of LocalPath/GoRoot/GoPath
	PackageLocalRoot                     //import "#/x/y/z" style
	PackageGlobal                        //import "x/y/z" style, not find yet
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

// PackagePath represent path information of a package
type PackagePath struct {
	ImportPath  string      // Regular original import path like: "x/y/z" "#/x/y/z" "./foo" "../foo" "#" "."
	Dir         string      // Dir of imported package
	Signature   string      // Signature of imported package, which is unique for every package Dir
	LocalRoot   string      // LocalRoot of imported package
	Root        string      // Root of imported package
	SrcRoot     string      // package source root directory ("" if unknown)
	PkgRoot     string      // package install root directory ("" if unknown)
	BinDir      string      // command install directory ("" if unknown)
	ConflictDir string      // this directory shadows Dir in $GOPATH
	Type        PackageType // PackageType of this package
}

func (p *PackagePath) Init() {
	p.ImportPath = ""
	p.Dir = ""
	p.Root = ""
	p.Signature = ""
	p.Type = 0
}

func (p *PackagePath) FindImportFromWd(ctxt *Context, imported string) error {
	return p.FindImport(ctxt, imported, wd)
}

func (p *PackagePath) FindImport(ctxt *Context, imported, srcDir string) error {
	formated, err := ctxt.FormatImportPath(imported, srcDir)
	if err != nil {
		return err
	}

	p.Type = formated.Type
	p.Root = formated.Root
	p.ImportPath = formated.ImportPath
	p.Dir = formated.Dir

	if !formated.Formated { //not import "./../foo" style
		switch style := formated.Style; {
		case style.IsLocalRoot(): //import "#/x/y/z" style
			localRoot := ctxt.SearchLocalRoot(srcDir)
			if localRoot == "" {
				return fmt.Errorf(`import %q: cannot find local-root(with sub-tree "<root>/src/vendor") up from %s`, imported, srcDir)
			}
			p.Type = PackageLocalRoot
			p.LocalRoot = localRoot
			p.Root = localRoot
			p.ImportPath = imported
			relPath := style.RealImportPath(imported)

			p.Dir = ctxt.joinPath(localRoot, "src", "vendor", relPath)
			if !ctxt.isDir(p.Dir) {
				p.Dir = ctxt.joinPath(localRoot, "src", relPath)
				if !ctxt.isDir(p.Dir) {
					return fmt.Errorf("import %q: cannot find package from local-root %s", imported, p.LocalRoot)
				}
			}

		case style.IsGlobal(): //import "x/y/z" style
			if err := p.searchGlobalPackage(p.ImportPath); err != nil {
				return err
			}
		}
	}

	if p.LocalRoot == "" {
		p.LocalRoot = ctxt.SearchLocalRoot(p.Dir)
	}

	return nil
}

// searchGlobalPackage find a global style package "x/y/z" form GoRoot/GoPath
// p.ImportPath must have been setted
func (p *PackagePath) searchGlobalPackage(imported string) error {
	return nil
}

// genSignature returns signature of a package
// which is unique for every dir
func (p *PackagePath) genSignature() {
}

func (info *PackagePath) Find(path, srcDir string) error {
	return nil
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot determine current directory: " + err.Error())
	}
	return wd
}

// dirToImportPath returns the pseudo-import path we use for a package
// outside the Go path. It begins with _/ and then contains the full path
// to the directory. If the package lives in c:\home\gopher\my\pkg then
// the pseudo-import path is _/c_/home/gopher/my/pkg.
// Using a pseudo-import path like this makes the ./ imports no longer
// a special case, so that all the code to deal with ordinary imports works
// automatically.
func dirToImportPath(dir string) string {
	return filepath.Join("_", strings.Map(makeImportValid, filepath.ToSlash(dir)))
}

func makeImportValid(r rune) rune {
	// Should match Go spec, compilers, and ../../go/parser/parser.go:/isValidImport.
	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
	if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
		return '_'
	}
	return r
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
