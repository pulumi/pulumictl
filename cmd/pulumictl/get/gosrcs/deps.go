package gosrcs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Package struct {
	Dir           string   // directory containing package sources
	ImportPath    string   // import path of package in dir
	ImportComment string   // path in import comment on package statement
	Name          string   // package name
	Doc           string   // package documentation string
	Target        string   // install path
	Shlib         string   // the shared library that contains this package (only set when -linkshared)
	Goroot        bool     // is this package in the Go root?
	Standard      bool     // is this package part of the standard Go library?
	Stale         bool     // would 'go install' do anything for this package?
	StaleReason   string   // explanation for Stale==true
	Root          string   // Go root or Go path dir containing this package
	ConflictDir   string   // this directory shadows Dir in $GOPATH
	BinaryOnly    bool     // binary-only package (no longer supported)
	ForTest       string   // package is only for use in named test
	Export        string   // file containing export data (when using -export)
	BuildID       string   // build ID of the compiled package (when using -export)
	Module        *Module  // info about package's containing module, if any (can be nil)
	Match         []string // command-line patterns matching this package
	DepOnly       bool     // package is only a dependency, not explicitly listed

	// Source files
	GoFiles           []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles          []string // .go source files that import "C"
	CompiledGoFiles   []string // .go files presented to compiler (when using -compiled)
	IgnoredGoFiles    []string // .go source files ignored due to build constraints
	IgnoredOtherFiles []string // non-.go source files ignored due to build constraints
	CFiles            []string // .c source files
	CXXFiles          []string // .cc, .cxx and .cpp source files
	MFiles            []string // .m source files
	HFiles            []string // .h, .hh, .hpp and .hxx source files
	FFiles            []string // .f, .F, .for and .f90 Fortran source files
	SFiles            []string // .s source files
	SwigFiles         []string // .swig files
	SwigCXXFiles      []string // .swigcxx files
	SysoFiles         []string // .syso object files to add to archive
	TestGoFiles       []string // _test.go files in package
	XTestGoFiles      []string // _test.go files outside package

	// Embedded files
	EmbedPatterns      []string // //go:embed patterns
	EmbedFiles         []string // files matched by EmbedPatterns
	TestEmbedPatterns  []string // //go:embed patterns in TestGoFiles
	TestEmbedFiles     []string // files matched by TestEmbedPatterns
	XTestEmbedPatterns []string // //go:embed patterns in XTestGoFiles
	XTestEmbedFiles    []string // files matched by XTestEmbedPatterns

	// Cgo directives
	CgoCFLAGS    []string // cgo: flags for C compiler
	CgoCPPFLAGS  []string // cgo: flags for C preprocessor
	CgoCXXFLAGS  []string // cgo: flags for C++ compiler
	CgoFFLAGS    []string // cgo: flags for Fortran compiler
	CgoLDFLAGS   []string // cgo: flags for linker
	CgoPkgConfig []string // cgo: pkg-config names

	// Dependency information
	Imports      []string          // import paths used by this package
	ImportMap    map[string]string // map from source import to ImportPath (identity entries omitted)
	Deps         []string          // all (recursively) imported dependencies
	TestImports  []string          // imports from TestGoFiles
	XTestImports []string          // imports from XTestGoFiles

	// Error information
	Incomplete bool            // this package or a dependency has an error
	Error      *PackageError   // error loading package
	DepsErrors []*PackageError // errors loading dependencies
}

type PackageError struct {
	ImportStack []string // shortest path from package named on command line to this one
	Pos         string   // position of error (if present, file:line:col)
	Err         string   // the error itself
}

type Module struct {
	Path      string       // module path
	Version   string       // module version
	Versions  []string     // available module versions (with -versions)
	Replace   *Module      // replaced by this module
	Time      *time.Time   // time version was created
	Update    *Module      // available update, if any (with -u)
	Main      bool         // is this the main module?
	Indirect  bool         // is this module only an indirect dependency of main module?
	Dir       string       // directory holding files for this module, if any
	GoMod     string       // path to go.mod file used when loading this module, if any
	GoVersion string       // go version used in module
	Retracted string       // retraction information, if any (with -retracted or -u)
	Error     *ModuleError // error loading module
}

type ModuleError struct {
	Err string // the error itself
}

var parentDir = string([]rune{'.', '.', os.PathSeparator})

type builder struct {
	repoRoot         string
	workingDirectory string

	packages map[string]*Package
	visited  map[string]bool

	sources []string
}

func (b *builder) findSources(packages []Package) (err error) {
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				err = e
			}
			panic(x)
		}
	}()

	b.packages = map[string]*Package{}

	for i := range packages {
		p := &packages[i]
		b.packages[p.ImportPath] = p
	}

	for importPath := range b.packages {
		b.visitPackage(importPath)
	}

	return nil
}

func (b *builder) visitPackage(importPath string) {
	if b.visited[importPath] {
		// We've already visited this package. Ignore it.
		return
	}
	b.visited[importPath] = true

	p := b.packages[importPath]
	if p == nil {
		// This can happen for certain Go-internal packages.
		return
	}

	// Visit this package's dependencies.
	for _, importPath := range p.Imports {
		b.visitPackage(importPath)
	}
	for _, importPath := range p.TestImports {
		b.visitPackage(importPath)
	}
	for _, importPath := range p.XTestImports {
		b.visitPackage(importPath)
	}

	// If this package's sources lie outside the repository, ignore them.
	packageDir, err := filepath.Rel(b.repoRoot, p.Dir)
	if err != nil {
		panic(err)
	}
	if strings.HasPrefix(packageDir, parentDir) {
		return
	}
	packageDir = filepath.Join(b.repoRoot, packageDir)

	addFiles := func(files []string) {
		for _, f := range files {
			repoAbsPath := filepath.Join(packageDir, f)
			dirRelPath, err := filepath.Rel(b.workingDirectory, repoAbsPath)
			if err != nil {
				panic(err)
			}

			b.sources = append(b.sources, dirRelPath)
		}
	}

	addFiles(p.GoFiles)
	addFiles(p.CgoFiles)
	addFiles(p.CompiledGoFiles)
	addFiles(p.CFiles)
	addFiles(p.CXXFiles)
	addFiles(p.MFiles)
	addFiles(p.HFiles)
	addFiles(p.FFiles)
	addFiles(p.SFiles)
	addFiles(p.SwigFiles)
	addFiles(p.SwigCXXFiles)
	addFiles(p.SysoFiles)
	addFiles(p.TestGoFiles)
	addFiles(p.XTestGoFiles)
	addFiles(p.EmbedFiles)
	addFiles(p.TestEmbedFiles)
	addFiles(p.XTestEmbedFiles)
}

func listPackages(dir string) ([]Package, error) {
	cmd := exec.Command("go", "list", "-json", "-test", "-deps", "./...")
	cmd.Dir = dir

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(pipe)
	var packages []Package
	for {
		var package_ Package
		if err = decoder.Decode(&package_); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		packages = append(packages, package_)
	}

	if err = cmd.Wait(); err != nil {
		return nil, err
	}

	return packages, nil
}

func findSources(repoRoot, workingDirectory string, packageDirs ...string) ([]string, error) {
	b := builder{
		repoRoot:         repoRoot,
		workingDirectory: workingDirectory,
		packages:         map[string]*Package{},
		visited:          map[string]bool{},
	}

	for _, dir := range packageDirs {
		dir, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}

		packages, err := listPackages(dir)
		if err != nil {
			return nil, fmt.Errorf("listing packages: %w", err)
		}
		b.findSources(packages)
	}

	return b.sources, nil
}
