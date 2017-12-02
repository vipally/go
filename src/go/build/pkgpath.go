// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

//Ally:  refer local package by [import "#/x/y/z"] style

import (
	"fmt"
	"go/parser"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	wd        = getwd()                              // current working dir
	goRootSrc = filepath.Join(Default.GOROOT, "src") // GoRoot/src
	gblSrcs   = Default.SrcDirs()                    // GoRoot/src & GoPaths/src
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

// FormatImportPath convert "." "./x/y/z" "../x/y/z" style import path to "#/x/y/z" "x/y/z" style if possible.
func (ctxt *Context) FormatImportPath(imported, importerDir string) (formated FormatImport, err error) {
	err = formated.FormatImportPath(ctxt, imported, importerDir)
	return
}

// FormatImport is formated import infomation, which prefers "#/foo" "x/y/z" to "./x/y/z" if possible.
type FormatImport struct {
	ImportPath  string      // formated import path,prefer "#/foo" "x/y/z" to "./x/y/z" if possible
	Dir         string      // full directory of imported package
	Root        string      // Root of imported package
	Type        PackageType // Type of imported package
	Style       ImportStyle // Style of ImportPath
	ConflictDir string      // this directory shadows Dir in $GOPATH
	Formated    bool        // ImportPath has changed from origin, maybe from "./../foo" to "#/foo" "x/y/z/foo".
}

// FormatImportPath convert "." "./x/y/z" "../x/y/z" style import path to "#/x/y/z" "x/y/z" style if possible.
func (fi *FormatImport) FormatImportPath(ctxt *Context, imported, importerDir string) (err error) {
	fi.ImportPath = imported
	fi.Type = PackageUnknown

	if fi.Style, err = GetImportStyle(imported); err != nil {
		return
	}

	if style := fi.Style; style.IsRelated() { //import "./../xxx"
		if importerDir == "" {
			err = fmt.Errorf("import %q: import relative to unknown directory", imported)
			return
		}
		if dir := ctxt.joinPath(importerDir, imported); ctxt.isDir(dir) {
			fi.Dir = dir
			fi.Formated = true

			if localRoot := ctxt.SearchLocalRoot(dir); localRoot != "" { //from local root
				localRootSrc := ctxt.joinPath(localRoot, "src")
				if sub, ok_ := ctxt.hasSubdir(localRootSrc, dir); ok_ {
					if sub != "" && sub != "." {
						fi.ImportPath = "#/" + sub
					} else {
						fi.ImportPath = "#"
					}
					fi.Root = localRoot
					fi.Type = PackageLocalRoot
					fi.Style = ImportStyleLocalRoot
					return
				}
			}

			if ok := fi.findGlobalRoot(ctxt, fi.Dir); ok {
				return
			}

			//StandAlone package out of LocalPath/GoRoot/GoPath
			fi.ImportPath = imported
			fi.Type = PackageStandAlone
		} else {
			err = fmt.Errorf("import %q: cannot find package at %s", imported, dir)
			return
		}
	}
	return
}

// findGlobalRoot root form GoRoot/GoPath for fullDir
func (fi *FormatImport) findGlobalRoot(ctxt *Context, fullDir string) bool {
	findRootSrc := ""
	for _, rootsrc := range gblSrcs {
		if sub, ok := ctxt.hasSubdir(rootsrc, fullDir); ok {
			fi.ImportPath = sub
			fi.Root = rootsrc[:len(rootsrc)-4] //remove suffix "/src"
			fi.Style = ImportStyleGlobal
			if rootsrc == goRootSrc {
				fi.Type = PackageGoRoot
			} else {
				fi.Type = PackageGoPath
			}
			findRootSrc = rootsrc
			break
		}
	}

	found := findRootSrc != ""
	if found { //check if conflict
		for _, rootsrc := range gblSrcs {
			if rootsrc != findRootSrc {
				if dir := ctxt.joinPath(rootsrc, fi.ImportPath); ctxt.isDir(dir) {
					fi.ConflictDir = dir
					break
				}
			}
		}
	}
	return found
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
	PackageLocalRoot                     //import "#/x/y/z" style
	PackageGlobal                        //import "x/y/z" style, not find yet
	PackageGoRoot                        //import "x/y/z" style, find from GoRoot
	PackageGoPath                        //import "x/y/z" style, find from GoPath
	PackageStandAlone                    //import "./../xx" style, which is out of LocalRoot/GoRoot/GoPath
)

func (t PackageType) IsValid() bool             { return t >= PackageStandAlone && t <= PackageGoPath }
func (t PackageType) IsStandAlonePackage() bool { return t == PackageStandAlone }
func (t PackageType) IsLocalPackage() bool      { return t == PackageLocalRoot }
func (t PackageType) IsStdPackage() bool        { return t == PackageGoRoot }
func (t PackageType) IsGlobalPackage() bool     { return t == PackageGoPath }

func (t PackageType) String() string {
	switch t {
	case PackageStandAlone:
		return "StandAlone"
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
	IsVendor    bool        // From vendor path
	Type        PackageType // PackageType of this package
	Style       ImportStyle // Style of ImportPath
}

func (p *PackagePath) Init() {
	p.ImportPath = ""
	p.Dir = ""
	p.Root = ""
	p.Signature = ""
	p.Type = 0
}

func (p *PackagePath) FindImportFromWd(ctxt *Context, imported string, mode ImportMode) error {
	return p.FindImport(ctxt, imported, wd, mode)
}

func (p *PackagePath) FindImport(ctxt *Context, imported, srcDir string, mode ImportMode) error {
	var fmted FormatImport
	if err := fmted.FormatImportPath(ctxt, imported, srcDir); err != nil {
		return err
	}

	p.Type = fmted.Type
	p.Root = fmted.Root
	p.ImportPath = fmted.ImportPath
	p.Dir = fmted.Dir
	p.Style = fmted.Style

	if !fmted.Formated { //not import "./../foo" style
		switch style := fmted.Style; {
		case style.IsLocalRoot(): //import "#/x/y/z" style
			localRoot := ctxt.SearchLocalRoot(srcDir)
			if localRoot == "" {
				return fmt.Errorf(`import %q: cannot find local-root(with sub-tree "<root>/src/vendor/") up from %s`, imported, srcDir)
			}
			p.Type = PackageLocalRoot
			p.LocalRoot = localRoot
			p.ImportPath = imported

			relPath := style.RealImportPath(imported)
			dir := ""
			if dir = ctxt.joinPath(localRoot, "src", "vendor", relPath); !ctxt.isDir(p.Dir) {
				if dir = ctxt.joinPath(localRoot, "src", relPath); !ctxt.isDir(p.Dir) {
					return fmt.Errorf("import %q: cannot find sub-package under local-root %s", imported, p.LocalRoot)
				}
			}
			p.Dir = dir
			p.Root = localRoot

		case style.IsGlobal(): //import "x/y/z" style
			if err := p.searchGlobalPackage(ctxt, p.ImportPath, srcDir, mode); err != nil {
				return err
			}
		}
	}

	return nil
}

// searchGlobalPackage find a global style package "x/y/z" form GoRoot/GoPath
// p.ImportPath must have been setted
func (p *PackagePath) searchGlobalPackage(ctxt *Context, imported, srcDir string, mode ImportMode) error {

	// tried records the location of unsuccessful package lookups
	var tried struct {
		vendor    []string
		goroot    string
		gopath    []string
		localroot string
	}
	gopath := ctxt.gopath()
	binaryOnly := false
	pkga := ""

	// Vendor directories get first chance to satisfy import.
	if mode&IgnoreVendor == 0 && srcDir != "" {
		searchVendor := func(root string, ptype PackageType) bool {
			sub, ok := ctxt.hasSubdir(root, srcDir)
			if !ok || !strings.HasPrefix(sub, "src/") || strings.Contains(sub, "/testdata/") {
				return false
			}
			for sub != "" {
				vendor := ctxt.joinPath(root, sub, "vendor")

				//ignore local vendor if not search for local vendor
				if !ptype.IsLocalPackage() && p.LocalRoot != "" {
					if _, ok := ctxt.hasSubdir(p.LocalRoot, vendor); ok {
						sub = parentPath(p.LocalRoot)
						continue
					}
				}

				if ctxt.isDir(vendor) {
					dir := ctxt.joinPath(vendor, imported)
					if ctxt.isDir(dir) && hasGoFiles(ctxt, dir) {
						p.Dir = dir
						p.ImportPath = pathpkg.Join(sub[:4], "vendor", imported) //remove prefix "src/"
						p.Type = ptype
						p.Root = root
						p.IsVendor = true
						return true
					}
					tried.vendor = append(tried.vendor, dir)
				}
				sub = parentPath(sub)
			}
			return false
		}

		//search local vendor first
		if localRoot := p.searchLocalRoot(ctxt, srcDir); localRoot != "" {
			if searchVendor(localRoot, PackageLocalRoot) {
				p.Root = localRoot
				goto Found
			}
		}
		if searchVendor(ctxt.GOROOT, PackageGoRoot) {
			goto Found
		}
		for _, root := range gopath {
			if searchVendor(root, PackageGoPath) {
				goto Found
			}
		}
	}

	// Determine directory from import path.

	//search goroot
	if ctxt.GOROOT != "" {
		dir := ctxt.joinPath(ctxt.GOROOT, "src", imported)
		isDir := ctxt.isDir(dir)
		binaryOnly = !isDir && mode&AllowBinary != 0 && pkga != "" && ctxt.isFile(ctxt.joinPath(ctxt.GOROOT, pkga))
		if isDir || binaryOnly {
			p.Dir = dir
			p.Type = PackageGoRoot
			p.Root = ctxt.GOROOT
			goto Found
		}
		tried.goroot = dir
	}
	//search gopath
	for _, root := range gopath {
		dir := ctxt.joinPath(root, "src", imported)
		isDir := ctxt.isDir(dir)
		binaryOnly = !isDir && mode&AllowBinary != 0 && pkga != "" && ctxt.isFile(ctxt.joinPath(root, pkga))
		if isDir || binaryOnly {
			p.Dir = dir
			p.Root = root
			p.Type = PackageGoPath
			goto Found
		}
		tried.gopath = append(tried.gopath, dir)
	}
	//search local root
	if localRoot := p.searchLocalRoot(ctxt, srcDir); localRoot != "" {
		dir := ctxt.joinPath(localRoot, "src", imported)
		isDir := ctxt.isDir(dir)
		binaryOnly = !isDir && mode&AllowBinary != 0 && pkga != "" && ctxt.isFile(ctxt.joinPath(localRoot, pkga))
		if isDir || binaryOnly {
			p.Dir = dir
			p.Root = localRoot
			p.Type = PackageLocalRoot
			p.ImportPath = "#/" + p.ImportPath
			goto Found
		}
		tried.localroot = dir
	}

	if true {
		// package was not found
		var paths []string
		format := "\t%s (vendor tree)"
		for _, dir := range tried.vendor {
			paths = append(paths, fmt.Sprintf(format, dir))
			format = "\t%s"
		}
		if tried.goroot != "" {
			paths = append(paths, fmt.Sprintf("\t%s (from $GOROOT)", tried.goroot))
		} else {
			paths = append(paths, "\t($GOROOT not set)")
		}
		format = "\t%s (from $GOPATH)"
		for _, dir := range tried.gopath {
			paths = append(paths, fmt.Sprintf(format, dir))
			format = "\t%s"
		}
		if len(tried.gopath) == 0 {
			paths = append(paths, "\t($GOPATH not set. For more details see: 'go help gopath')")
		}
		if tried.localroot != "" {
			paths = append(paths, fmt.Sprintf("\t%s (from $LocalRoot)", tried.localroot))
		}

		return fmt.Errorf("cannot find package %q in any of:\n%s", imported, strings.Join(paths, "\n"))
	}

Found:
	if p.Root != "" {
		p.SrcRoot = ctxt.joinPath(p.Root, "src")
		p.PkgRoot = ctxt.joinPath(p.Root, "pkg")
		p.BinDir = ctxt.joinPath(p.Root, "bin")
		if pkga != "" {
			//p.PkgTargetRoot = ctxt.joinPath(p.Root, pkgtargetroot)
			//p.PkgObj = ctxt.joinPath(p.Root, pkga)
		}
	}
	p.searchLocalRoot(ctxt, srcDir)
	p.genSignature()
	return nil
}

func (p *PackagePath) searchLocalRoot(ctxt *Context, srcDir string) string {
	if p.LocalRoot == "" {
		p.LocalRoot = ctxt.SearchLocalRoot(srcDir)
	}
	return p.LocalRoot
}

// genSignature returns signature of a package
// which is unique for every dir
func (p *PackagePath) genSignature() {
	switch {
	case p.Type.IsStandAlonePackage() || p.Type.IsLocalPackage():
		p.Signature = dirToImportPath(p.Dir)
	default:
		p.Signature = p.Style.RealImportPath(p.ImportPath)
	}
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot determine current directory: " + err.Error())
	}
	return wd
}

func parentPath(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
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
