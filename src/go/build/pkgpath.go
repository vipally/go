// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

//Ally:  refer local package by [import "#/x/y/z"] style

import (
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	// illegal chars in import path
	illegalImportChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
)

var (
	//match "." ".." "./xxx" "../xxx"
	//relatedRE = regexp.MustCompile(`^\.{1,2}(?:/.+)*?`)

	//match "/testdata/" or "\\testdata\\"
	testdataRE = regexp.MustCompile(`(?:^|\\|/)testdata(?:$|\\|/)`)

	// match "<root>/src/..." case to find <root>
	// it will match the longest path if more than 1 "/src/" found
	srcRE = regexp.MustCompile(`(^.+)[\\|/]src(?:$|\\|/)`)
)

// SetGoRoot change GOROOT and RefreshEnvCache
func (ctxt *Context) SetGoRoot(root string) {
	ctxt.GOROOT = root
	ctxt.RefreshEnvCache()
}

// SetGoRoot change GOPATH and RefreshEnvCache
func (ctxt *Context) SetGoPath(path string) {
	ctxt.GOPATH = path
	ctxt.RefreshEnvCache()
}

// RefreshEnvCache update cached environment valuse when GOROOT/GOPATH changes
func (ctxt *Context) RefreshEnvCache() {
	ctxt.goRootSrc = ctxt.joinPath(ctxt.GOROOT, "src")
	ctxt.gblSrcs = ctxt.SrcDirs()
	ctxt.gopaths = ctxt.gopath()
}

// SearchLocalRoot find the <root> path that contains such patten of sub-tree "<root>/src/vendor/" up from curPath,
// which is the root of local project.
// SearchLocalRoot will never match path under GoRoot.
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
	root, _ := ctxt.searchLocalRoot(curPath)
	return root
}

func (ctxt *Context) searchLocalRoot(curPath string) (root, src string) {
	//do not match LocalRoot under GoRoot
	if _, ok := ctxt.hasSubdir(ctxt.GOROOT, curPath); ok {
		return
	}

	for root, rootSrc := filepath.Clean(curPath), ""; ; {
		// if dir = `c:\root\src\prj\src\main`
		// match[0]= []string{`c:\root\src\prj\src\`, `c:\root\src\prj`}
		if match := srcRE.FindAllStringSubmatch(root, 1); match != nil {
			rootSrc, root = match[0][0], match[0][1]
			if vendor := ctxt.joinPath(rootSrc, "vendor"); ctxt.isDir(vendor) {
				return root, rootSrc
			}
		} else {
			break
		}
	}
	return
}

// GetPackageSignature return signature of an import package.
// For PackageStandAlone and PackageLocalRoot, return a pseudo-import path by dirToImportPath
// For PackageGoRoot and PackageGoPath, return path
// A fast way to find key of package chache
func (ctxt *Context) GetPkgSignature(imported, importerDir, parentRoot string, mode ImportMode) string {
	var packagePath PackagePath
	_ = packagePath.FindImport(ctxt, imported, importerDir, mode) //ignore error
	return packagePath.Signature
}

// VenderedLocalRootPath find if vendored from localRoot
// imported must "x/y/z" style
func (ctxt *Context) VendoredLocalRootPath(imported, srcDir, localRoot string) string {
	if localRoot != "" {
		vendored := ctxt.matchVendorFromRoot(imported, srcDir, localRoot, localRoot, PackageLocalRoot, nil)
		if vendored != "" {
			return vendored
		}
	}
	return imported
}

func getOrDef(s, def string) string {
	if s != "" {
		return s
	}
	return def
}

// FormatImport is formated import infomation, which prefers "#/foo" "x/y/z" to "./x/y/z" if possible.
type FormatImport struct {
	OriginImportPath string      // original import path. like: "." "./../xx" "#/xx" "xx"
	ImporterDir      string      // dir of importer
	FmtImportPath    string      // formated import path. like: "#/x/y/z" "x/y/z", full path like "c:\x\y\z" for standalone packages
	Dir              string      // full directory of imported package
	Root             string      // Root of imported package
	ConflictDir      string      // this directory shadows Dir in $GOPATH/$GoPath
	Type             PackageType // Type of formated ImportPath
	Style            ImportStyle // Style of formated ImportPath
	Formated         bool        // FmtImportPath has changed from OriginImportPath, maybe from "./../foo" to "#/foo" "x/y/z/foo".
}

// PackagePath represent path information of a package
type PackagePath struct {
	ImporterDir string // dir of importer

	ImportPath  string // Regular import path related to Root, full path like "c:\x\y\z" for standalone packages
	Dir         string // Dir of imported package
	Root        string // Root of imported package
	ConflictDir string // this directory shadows Dir in $GOPATH/$GoPath

	OriginImportPath string      // original import path. like: "." "./../xx" "#/xx" "xx"
	FmtImportPath    string      // formated import path. like: "#/x/y/z" "x/y/z", full path like "c:\x\y\z" for standalone packages
	LocalRoot        string      // LocalRoot of imported package
	Signature        string      // Signature of imported package, which is unique for every package Dir
	Type             PackageType // Type of formated ImportPath
	Style            ImportStyle // Style of formated ImportPath
	IsVendor         bool        // From vendor path
}

func (fi *FormatImport) Init() {
	fi.OriginImportPath = ""
	fi.FmtImportPath = ""
	fi.ImporterDir = ""
	fi.Dir = ""
	fi.Root = ""
	fi.ConflictDir = ""
	fi.Style = ImportStyleUnknown
	fi.Type = PackageUnknown
	fi.Formated = false
}

// FormatImportPath convert "." "./x/y/z" "../x/y/z" style import path to "#/x/y/z" "x/y/z" style if possible.
// It checks the literal of import style only, but doesn't check whether the target package path exists.
func (fi *FormatImport) FormatImportPath(ctxt *Context, imported, importerDir string) (err error) {
	fi.OriginImportPath = imported
	fi.ImporterDir = importerDir
	fi.FmtImportPath = imported
	fi.Type = PackageUnknown

	if fi.Style, err = GetImportStyle(imported); err != nil {
		return
	}

	if fi.Style.IsRelated() { //import "./../xxx"
		if importerDir == "" {
			return fmt.Errorf("import %q: import relative to unknown directory", imported)
		}

		dir := ctxt.joinPath(importerDir, imported)
		fi.Dir = dir
		fi.Formated = true

		if localRoot, localRootSrc := ctxt.searchLocalRoot(dir); localRoot != "" { //from local root
			if sub, ok_ := ctxt.hasSubdir(localRootSrc, dir); ok_ && !inTestdata(sub) {
				importPath := "#"
				if sub != "" && sub != "." {
					importPath = "#/" + sub
				}

				fi.FmtImportPath = importPath
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
		fi.FmtImportPath = "."
		fi.Type = PackageStandAlone
		fi.Style = ImportStyleSelf
	} else {
		if inTestdata(fi.FmtImportPath) {
			return fmt.Errorf("import %q: cannot refer package under testdata", fi.FmtImportPath)
		}
	}
	return
}

// findGlobalRoot find root form GoRoot/GoPath for fullDir
func (fi *FormatImport) findGlobalRoot(ctxt *Context, fullDir string) bool {
	foundRootSrc := ""
	for _, rootsrc := range ctxt.gblSrcs {
		if sub, ok := ctxt.hasSubdir(rootsrc, fullDir); ok && !inTestdata(sub) {
			fi.FmtImportPath = sub
			fi.Root = rootsrc[:len(rootsrc)-4] //remove suffix "/src"
			fi.Style = ImportStyleGlobal
			if ctxt.goRootSrc == rootsrc {
				fi.Type = PackageGoRoot
			} else {
				fi.Type = PackageGoPath
			}
			foundRootSrc = rootsrc
			break
		}
	}

	found := foundRootSrc != ""
	if found { //check if conflict
		for _, rootsrc := range ctxt.gblSrcs {
			if rootsrc != foundRootSrc {
				if dir := ctxt.joinPath(rootsrc, fi.FmtImportPath); ctxt.isDir(dir) {
					fi.ConflictDir = dir
					break
				}
			}
		}
	}
	return found
}

// ImportStyle represents style of a package import statement
type ImportStyle uint8

const (
	ImportStyleUnknown   ImportStyle = iota //unknown, invalid
	ImportStyleSelf                         //import "."
	ImportStyleRelated                      //import "./x/y/z" "../x/y/z"
	ImportStyleLocalRoot                    //import "#/x/y/z" "#"
	ImportStyleGlobal                       //import "x/y/z"

	importStyleEnd // end of ImportStyle, invalid
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
func (st ImportStyle) GoString() string {
	return st.String()
}

func (st ImportStyle) IsValid() bool     { return st > 0 && st < importStyleEnd }
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
	if err := ValidateImportPath(imported); err != nil {
		return ImportStyleUnknown, err
	}

	switch lead := imported[0]; {
	case lead == '.':
		if len(imported) == 1 {
			return ImportStyleSelf, nil //"."
		} else {
			second := imported[1]
			switch second {
			case '/':
				return ImportStyleRelated, nil //"./xxx"
			case '.':
				if len(imported) == 2 || imported[2] == '/' {
					return ImportStyleRelated, nil //".." "../xxx"
				}
			}
		}
	case lead == '#':
		if len(imported) == 1 || imported[1] == '/' {
			return ImportStyleLocalRoot, nil //"#" "#/xxx"
		}
	default:
		return ImportStyleGlobal, nil //"x/y/z"
	}

	return ImportStyleUnknown, fmt.Errorf("import %q: invalid import path", imported)
}

// IsValidImportStatement return if a import path in statement is valid
// import "./xxx" "../xxx" is not allowed in statement
func CheckImportStatement(imported string) error {
	style, err := GetImportStyle(imported)
	if err != nil {
		return err
	}
	if style.IsRelated() || style.IsSelf() {
		return fmt.Errorf("import %q: related import not allowed in statement", imported)
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

	packageTypeEnd //end of PackageType, invalid
)

func (t PackageType) IsValid() bool             { return t > 0 && t < packageTypeEnd }
func (t PackageType) IsStandAlonePackage() bool { return t == PackageStandAlone }
func (t PackageType) IsLocalPackage() bool      { return t == PackageLocalRoot }
func (t PackageType) IsStdPackage() bool        { return t == PackageGoRoot }
func (t PackageType) IsGlobalPackage() bool     { return t == PackageGoPath }

func (t PackageType) String() string {
	switch t {
	case PackageStandAlone:
		return "PackageStandAlone"
	case PackageLocalRoot:
		return "PackageLocalRoot"
	case PackageGoRoot:
		return "PackageGoRoot"
	case PackageGoPath:
		return "PackageGoPath"
	}
	return "PackageUnknown"
}

func (t PackageType) GoString() string {
	return t.String()
}

// copy PackagePath info to Package object
func (p *Package) copyFromPackagePath(ctxt *Context, pp *PackagePath) error {
	p.ImportPath = pp.ImportPath
	p.Dir = pp.Dir
	p.LocalRoot = pp.LocalRoot
	p.Root = pp.Root
	p.ConflictDir = pp.ConflictDir

	p.FmtImportPath = pp.FmtImportPath
	p.Signature = pp.Signature
	p.IsVendor = pp.IsVendor
	p.Type = pp.Type
	p.Style = pp.Style

	p.Goroot = p.Type.IsStdPackage()

	var pkgtargetroot string
	var pkga string
	var pkgerr error
	suffix := ""
	if ctxt.InstallSuffix != "" {
		suffix = "_" + ctxt.InstallSuffix
	}
	switch ctxt.Compiler {
	case "gccgo":
		pkgtargetroot = "pkg/gccgo_" + ctxt.GOOS + "_" + ctxt.GOARCH + suffix
	case "gc":
		pkgtargetroot = "pkg/" + ctxt.GOOS + "_" + ctxt.GOARCH + suffix
	default:
		// Save error for end of function.
		pkgerr = fmt.Errorf("import %q: unknown compiler %q", p.ImportPath, ctxt.Compiler)
	}

	// standalone imports have no installed path
	if !pp.Type.IsStandAlonePackage() {
		switch ctxt.Compiler {
		case "gccgo":
			dir, elem := pathpkg.Split(p.ImportPath)
			pkga = pkgtargetroot + "/" + dir + "lib" + elem + ".a"
		case "gc":
			pkga = pkgtargetroot + "/" + p.ImportPath + ".a"
		}
	}

	if p.Root != "" {
		p.SrcRoot = ctxt.joinPath(p.Root, "src")
		p.PkgRoot = ctxt.joinPath(p.Root, "pkg")
		p.BinDir = ctxt.joinPath(p.Root, "bin")
		if pkga != "" {
			p.PkgTargetRoot = ctxt.joinPath(p.Root, pkgtargetroot)
			p.PkgObj = ctxt.joinPath(p.Root, pkga)
		}
	}

	return pkgerr
}

func (p *PackagePath) Init() {
	p.ImportPath = ""
	p.Dir = ""
	p.Signature = ""
	p.LocalRoot = ""
	p.Root = ""
	p.ConflictDir = ""
	p.IsVendor = false
	p.Type = PackageUnknown
	p.Style = ImportStyleUnknown
}

func (p *PackagePath) FindImportFromWd(ctxt *Context, imported string, mode ImportMode) error {
	return p.FindImport(ctxt, imported, getwd(), mode)
}

func (p *PackagePath) copyFromFormatImport(fmted *FormatImport) {
	p.Style = fmted.Style
	p.Type = fmted.Type
	p.Root = fmted.Root
	p.OriginImportPath = fmted.OriginImportPath
	p.ImporterDir = fmted.ImporterDir
	p.FmtImportPath = fmted.FmtImportPath
	p.Dir = fmted.Dir
	p.ImportPath = p.Style.RealImportPath(p.FmtImportPath)
}

//format import path, and copy the info
func (p *PackagePath) formatImportAndCopy(ctxt *Context, imported, srcDir string) (bool, error) {
	var fmted FormatImport
	err := fmted.FormatImportPath(ctxt, imported, srcDir)
	p.copyFromFormatImport(&fmted)
	return fmted.Formated, err
}

func (p *PackagePath) FindImport(ctxt *Context, imported, srcDir string, mode ImportMode) (err error) {
	defer func() {
		p.searchLocalRoot(ctxt, srcDir)
		p.genSignature()
	}()

	formated := false
	if formated, err = p.formatImportAndCopy(ctxt, imported, srcDir); err != nil {
		return err
	}

	if !formated { //not import "./../foo" style
		switch style := p.Style; {
		case style.IsLocalRoot(): //import "#/x/y/z" style
			localRoot := ctxt.SearchLocalRoot(srcDir)
			if localRoot == "" {
				return fmt.Errorf(`import %q: cannot find local-root(with sub-tree "<root>/src/vendor/") up from %s`, imported, srcDir)
			}
			if inTestdata(imported) {
				return fmt.Errorf("import %q: cannot refer package under testdata", imported)
			}
			p.Type = PackageLocalRoot
			p.LocalRoot = localRoot
			p.ImportPath = style.RealImportPath(imported)

			relPath := p.ImportPath
			dir := ""
			if dir = ctxt.joinPath(localRoot, "src", "vendor", relPath); mode&IgnoreVendor != 0 || !ctxt.isDir(dir) {
				if dir = ctxt.joinPath(localRoot, "src", relPath); !ctxt.isDir(dir) {
					return fmt.Errorf("import %q: cannot find sub-package under local-root %s", imported, p.LocalRoot)
				}
			}
			p.Dir = dir
			p.Root = localRoot

		case style.IsGlobal(): //import "x/y/z" style
			if err := p.findGlobalPackage(ctxt, p.ImportPath, srcDir, mode); err != nil {
				return err
			}
		}
	} else {
		if !ctxt.isDir(p.Dir) { //check if a valid dir
			return fmt.Errorf("import %q: cannot find package at %s", imported, p.Dir)
		}
	}

	return nil
}

// searchGlobalPackage find a global style package "x/y/z" form GoRoot/GoPath
// p.ImportPath must have been setted
func (p *PackagePath) findGlobalPackage(ctxt *Context, imported, srcDir string, mode ImportMode) error {
	if inTestdata(imported) {
		return fmt.Errorf("import %q: cannot refer package under testdata", imported)
	}

	// tried records the location of unsuccessful package lookups
	var tried struct {
		vendor    []string
		goroot    string
		gopath    []string
		localroot string
	}
	binaryOnly := false
	pkga := ""

	// Vendor directories get first chance to satisfy import.
	if mode&IgnoreVendor == 0 && srcDir != "" {
		localRoot := p.searchLocalRoot(ctxt, srcDir)
		vendored, root, ptype := ctxt.findFromVendor(imported, srcDir, localRoot, &tried.vendor)
		if vendored != "" {
			p.ImportPath = vendored
			p.Root = root
			p.Type = ptype
			p.IsVendor = true
			p.Dir = ctxt.joinPath(root, "src", vendored)
			return nil
		}
	}

	// Determine directory from import path.

	//search goroot
	if ctxt.GOROOT != "" {
		dir := ctxt.joinPath(ctxt.goRootSrc, imported)
		isDir := ctxt.isDir(dir)
		binaryOnly = !isDir && mode&AllowBinary != 0 && pkga != "" && ctxt.isFile(ctxt.joinPath(ctxt.GOROOT, pkga))
		if isDir || binaryOnly {
			p.Dir = dir
			p.Type = PackageGoRoot
			p.Root = ctxt.GOROOT
			p.IsVendor = false
			return nil
		}
		tried.goroot = dir
	}

	//search gopath
	for _, root := range ctxt.gopaths {
		dir := ctxt.joinPath(root, "src", imported)
		isDir := ctxt.isDir(dir)
		binaryOnly = !isDir && mode&AllowBinary != 0 && pkga != "" && ctxt.isFile(ctxt.joinPath(root, pkga))
		if isDir || binaryOnly {
			p.Dir = dir
			p.Root = root
			p.Type = PackageGoPath
			p.IsVendor = false
			return nil
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
			p.IsVendor = false
			return nil
		}
		tried.localroot = dir
	}

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
		paths = append(paths, fmt.Sprintf("\t%s (from #LocalRoot)", tried.localroot))
	}
	return fmt.Errorf("cannot find package %q in any of:\n%s", imported, strings.Join(paths, "\n"))
}

func (p *PackagePath) searchLocalRoot(ctxt *Context, srcDir string) string {
	if p.LocalRoot == "" {
		p.LocalRoot = ctxt.SearchLocalRoot(srcDir)
	}
	return p.LocalRoot
}

func (ctxt *Context) findFromVendor(imported, srcDir, localRoot string,
	triedvendor *[]string) (vendored, root string, ptype PackageType) {

	//search local vendor first
	if localRoot != "" {
		vendored = ctxt.matchVendorFromRoot(imported, srcDir, localRoot, localRoot, PackageLocalRoot, triedvendor)
		if vendored != "" {
			root, ptype = localRoot, PackageLocalRoot
			return
		}
	}

	vendored = ctxt.matchVendorFromRoot(imported, srcDir, ctxt.GOROOT, localRoot, PackageGoRoot, triedvendor)
	if vendored != "" {
		root, ptype = ctxt.GOROOT, PackageGoRoot
		return
	}

	for _, root_ := range ctxt.gopaths {
		vendored = ctxt.matchVendorFromRoot(imported, srcDir, root_, localRoot, PackageGoPath, triedvendor)
		if vendored != "" {
			root, ptype = root_, PackageGoPath
			return
		}
	}
	return
}

//try to find matched vendor under root
func (ctxt *Context) matchVendorFromRoot(imported, srcDir, root, localRoot string,
	ptype PackageType, triedvendor *[]string) (vendored string) {
	sub, ok := ctxt.hasSubdir(root, srcDir)
	if !ok || !strings.HasPrefix(sub, "src/") || inTestdata(sub) {
		return
	}

	//ignore local vendor if search for global vendor
	if !ptype.IsLocalPackage() && localRoot != "" {
		if _, ok := ctxt.hasSubdir(localRoot, srcDir); ok {
			parent := parentPath(localRoot)
			sub, _ = ctxt.hasSubdir(root, parent)
		}
	}

	for sub != "" {
		vendor := ctxt.joinPath(root, sub, "vendor")

		//fmt.Printf("search vendor: \n\troot[%s] \n\ttype[%v] \n\tsub[%s] \n\tvendor[%s]\n\tLocalRoot[%s]\n", root, ptype, sub, vendor, localRoot)

		if ctxt.isDir(vendor) {
			dir := ctxt.joinPath(vendor, imported)
			if ctxt.isDir(dir) && hasGoFiles(ctxt, dir) {
				//remove prefix "src/" from sub
				if sub = sub[3:]; len(sub) > 0 {
					sub = sub[1:]
				}
				vendored = pathpkg.Join(sub, "vendor", imported)
				return
			}
			if triedvendor != nil {
				*triedvendor = append(*triedvendor, dir)
			}
		}
		sub = parentPath(sub)
	}
	return
}

// genSignature returns signature of a package
// which is unique for every dir
func (p *PackagePath) genSignature() {
	switch {
	case !p.Type.IsValid() || p.Type.IsStandAlonePackage() || p.Type.IsLocalPackage():
		p.Signature = DirToImportPath(p.Dir)
	default:
		p.Signature = p.Style.RealImportPath(p.ImportPath)
	}
}

// IsValidImportPath return if a import path is valid, which returns bool
func IsValidImportPath(imported string) bool {
	if imported == "" || imported[0] == '/' {
		return false
	}

	sCheck := imported
	for len(sCheck) > 0 && sCheck[0] == '.' { //remove prefix "."
		sCheck = sCheck[1:]
	}

	// ".../xxx" ".//xxx" is invalid
	// "." ".." is valid
	if len(imported) > len(sCheck)+2 || len(sCheck) > 0 && pathpkg.Clean(sCheck) != sCheck {
		return false
	}

	//import "#" "#/foo" is valid style
	if len(imported) == len(sCheck) && len(sCheck) > 0 && sCheck[0] == '#' {
		sCheck = sCheck[1:]
	}

	for _, r := range sCheck {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalImportChars, r) {
			return false
		}
	}
	return true
}

// ValidateImportPath return if a import path is valid, which returns error
func ValidateImportPath(imported string) error {
	if imported == "" {
		return fmt.Errorf("import %q: invalid import path", imported)
	}
	if imported[0] == '/' {
		return fmt.Errorf("import %q: cannot import absolute path", imported)
	}

	sCheck := imported
	for len(sCheck) > 0 && sCheck[0] == '.' { //remove prefix "."
		sCheck = sCheck[1:]
	}

	// ".../xxx" ".//xxx" is invalid
	// "." ".." is valid
	if len(imported) > len(sCheck)+2 || len(sCheck) > 0 && pathpkg.Clean(sCheck) != sCheck {
		return fmt.Errorf("import %q: invalid import path", imported)
	}

	//import "#" "#/foo" is valid style
	if len(imported) == len(sCheck) && len(sCheck) > 0 && sCheck[0] == '#' {
		sCheck = sCheck[1:]
	}

	for _, r := range sCheck {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalImportChars, r) {
			return fmt.Errorf("import %q: invalid character %#U", imported, r)
		}
	}

	return nil
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot determine current directory: " + err.Error())
	}
	return wd
}

func parentPath(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		c := path[i]
		if c == '\\' || c == '/' {
			return path[:i]
		}
	}
	return ""
}

// DirToImportPath returns the pseudo-import path we use for a package
// outside the Go path. It begins with _/ and then contains the full path
// to the directory. If the package lives in c:\home\gopher\my\pkg then
// the pseudo-import path is _/c_/home/gopher/my/pkg.
// Using a pseudo-import path like this makes the ./ imports no longer
// a special case, so that all the code to deal with ordinary imports works
// automatically.
func DirToImportPath(dir string) string {
	return filepath.Join("_", strings.Map(makeImportValid, filepath.ToSlash(dir)))
}

func makeImportValid(r rune) rune {
	// Should match Go spec, compilers, and ../../go/parser/parser.go:/isValidImport.
	if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalImportChars, r) {
		return '_'
	}
	return r
}

// p.Dir directory may or may not exist. Gather partial information first, check if it exists later.
// Determine canonical import path, if any.
// Exclude results where the import path would include /testdata/.
func inTestdata(sub string) bool {
	return testdataRE.MatchString(sub)
}
