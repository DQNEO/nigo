package main

import (
	"os"

	"github.com/DQNEO/babygo/internal/codegen"
	"github.com/DQNEO/babygo/internal/ir"
	"github.com/DQNEO/babygo/internal/sema"
	"github.com/DQNEO/babygo/internal/universe"
	"github.com/DQNEO/babygo/lib/ast"
	"github.com/DQNEO/babygo/lib/fmt"
	"github.com/DQNEO/babygo/lib/mylib"
	"github.com/DQNEO/babygo/lib/parser"
	"github.com/DQNEO/babygo/lib/path"
	"github.com/DQNEO/babygo/lib/strings"
	"github.com/DQNEO/babygo/lib/token"
)

var fset *token.FileSet

// --- builder ---
//var CurrentPkg *ir.PkgContainer

type PackageToBuild struct {
	path  string
	name  string
	files []string
}

func resolveImports(file *ast.File) {
	mapImports := make(map[string]bool)
	for _, imprt := range file.Imports {
		// unwrap double quote "..."
		rawValue := imprt.Path.Value
		pth := rawValue[1 : len(rawValue)-1]
		base := path.Base(pth)
		mapImports[base] = true
	}
	for _, ident := range file.Unresolved {
		// lookup imported package name
		_, ok := mapImports[ident.Name]
		if ok {
			ident.Obj = &ast.Object{
				Kind: ast.Pkg,
				Name: ident.Name,
			}
		}
	}
}

// "some/dir" => []string{"a.go", "b.go"}
func findFilesInDir(dir string) []string {
	dirents, _ := mylib.Readdirnames(dir)
	var r []string
	for _, dirent := range dirents {
		if dirent == "_.s" {
			continue
		}
		if strings.HasSuffix(dirent, ".go") || strings.HasSuffix(dirent, ".s") {
			r = append(r, dirent)
		}
	}
	return r
}

func isStdLib(pth string) bool {
	return !strings.Contains(pth, ".")
}

func getImportPathsFromFile(file string) []string {
	fset := &token.FileSet{}
	astFile0 := parseImports(fset, file)
	var paths []string
	for _, importSpec := range astFile0.Imports {
		rawValue := importSpec.Path.Value
		pth := rawValue[1 : len(rawValue)-1]
		paths = append(paths, pth)
	}
	return paths
}

func removeNode(tree DependencyTree, node string) {
	for _, paths := range tree {
		delete(paths, node)
	}

	delete(tree, node)
}

func getKeys(tree DependencyTree) []string {
	var keys []string
	for k, _ := range tree {
		keys = append(keys, k)
	}
	return keys
}

type DependencyTree map[string]map[string]bool

// Do topological sort
// In the result list, the independent (lowest level) packages come first.
func sortTopologically(tree DependencyTree) []string {
	var sorted []string
	for len(tree) > 0 {
		keys := getKeys(tree)
		mylib.SortStrings(keys)
		for _, _path := range keys {
			children, ok := tree[_path]
			if !ok {
				panic("not found in tree")
			}
			if len(children) == 0 {
				// collect leaf node
				sorted = append(sorted, _path)
				removeNode(tree, _path)
			}
		}
	}
	return sorted
}

func getPackageDir(importPath string) string {
	if isStdLib(importPath) {
		return prjSrcPath + "/" + importPath
	} else {
		return srcPath + "/" + importPath
	}
}

func collectDependency(tree DependencyTree, paths map[string]bool) {
	for pkgPath, _ := range paths {
		if pkgPath == "unsafe" || pkgPath == "runtime" {
			continue
		}
		packageDir := getPackageDir(pkgPath)
		fnames := findFilesInDir(packageDir)
		children := make(map[string]bool)
		for _, fname := range fnames {
			if !strings.HasSuffix(fname, ".go") {
				// skip ".s"
				continue
			}
			_paths := getImportPathsFromFile(packageDir + "/" + fname)
			for _, pth := range _paths {
				if pth == "unsafe" || pth == "runtime" {
					continue
				}
				children[pth] = true
			}
		}
		tree[pkgPath] = children
		collectDependency(tree, children)
	}
}

var srcPath string
var prjSrcPath string

func collectAllPackages(inputFiles []string) []string {
	directChildren := collectDirectDependents(inputFiles)
	tree := make(DependencyTree)
	collectDependency(tree, directChildren)
	sortedPaths := sortTopologically(tree)

	// sort packages by this order
	// 1: pseudo
	// 2: stdlib
	// 3: external
	paths := []string{"unsafe", "runtime"}
	for _, pth := range sortedPaths {
		if isStdLib(pth) {
			paths = append(paths, pth)
		}
	}
	for _, pth := range sortedPaths {
		if !isStdLib(pth) {
			paths = append(paths, pth)
		}
	}
	return paths
}

func collectDirectDependents(inputFiles []string) map[string]bool {
	importPaths := make(map[string]bool)
	for _, inputFile := range inputFiles {
		paths := getImportPathsFromFile(inputFile)
		for _, pth := range paths {
			importPaths[pth] = true
		}
	}
	return importPaths
}

func collectSourceFiles(pkgDir string) []string {
	fnames := findFilesInDir(pkgDir)
	var files []string
	for _, fname := range fnames {
		srcFile := pkgDir + "/" + fname
		files = append(files, srcFile)
	}
	return files
}

func parseImports(fset *token.FileSet, filename string) *ast.File {
	f, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		panic(filename + ":" + err.Error())
	}
	return f
}

func parseFile(fset *token.FileSet, filename string) *ast.File {
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		panic(err.Error())
	}
	return f
}

// compile compiles go files of a package into an assembly file, and copy input assembly files into it.
func compile(universe *ast.Scope, fset *token.FileSet, pkgPath string, name string, gofiles []string, asmfiles []string, outFilePath string) *ir.PkgContainer {
	_pkg := &ir.PkgContainer{Name: name, Path: pkgPath, Fset: fset}
	_pkg.FileNoMap = make(map[string]int)
	outAsmFile, err := os.Create(outFilePath)
	if err != nil {
		panic(err)
	}
	fout = outAsmFile
	codegen.Fout = fout
	printf("#=== Package %s\n", _pkg.Path)

	codegen.TypesMap = make(map[string]*codegen.DtypeEntry)
	codegen.TypeId = 1

	pkgScope := ast.NewScope(universe)
	for i, file := range gofiles {
		fileno := i + 1
		_pkg.FileNoMap[file] = fileno
		printf("  .file %d \"%s\"\n", fileno, file)

		astFile := parseFile(fset, file)
		//		logf("[main]package decl lineno = %s\n", fset.Position(astFile.Package))
		_pkg.Name = astFile.Name.Name
		_pkg.AstFiles = append(_pkg.AstFiles, astFile)
		for name, obj := range astFile.Scope.Objects {
			pkgScope.Objects[name] = obj
		}
	}
	for _, astFile := range _pkg.AstFiles {
		resolveImports(astFile)
		var unresolved []*ast.Ident
		for _, ident := range astFile.Unresolved {
			obj := pkgScope.Lookup(ident.Name)
			if obj != nil {
				ident.Obj = obj
			} else {
				obj := universe.Lookup(ident.Name)
				if obj != nil {

					ident.Obj = obj
				} else {

					// we should allow unresolved for now.
					// e.g foo in X{foo:bar,}
					unresolved = append(unresolved, ident)
				}
			}
		}
		for _, dcl := range astFile.Decls {
			_pkg.Decls = append(_pkg.Decls, dcl)
		}
	}

	printf("#--- walk \n")
	sema.Walk(_pkg)
	codegen.GenerateCode(_pkg)

	// append static asm files
	for _, file := range asmfiles {
		fmt.Fprintf(outAsmFile, "# === static assembly %s ====\n", file)
		asmContents, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		outAsmFile.Write(asmContents)
	}

	outAsmFile.Close()
	fout = nil
	codegen.Fout = nil
	sema.CurrentPkg = nil
	return _pkg
}

func buildAll(args []string) {
	workdir := os.Getenv("WORKDIR")
	if workdir == "" {
		workdir = "/tmp"
	}

	var inputFiles []string
	for _, arg := range args {
		switch arg {
		case "-DG":
			codegen.DebugCodeGen = true
		default:
			inputFiles = append(inputFiles, arg)
		}
	}

	paths := collectAllPackages(inputFiles)
	var packagesToBuild []*PackageToBuild
	for _, _path := range paths {
		files := collectSourceFiles(getPackageDir(_path))
		packagesToBuild = append(packagesToBuild, &PackageToBuild{
			name:  path.Base(_path),
			path:  _path,
			files: files,
		})
	}

	packagesToBuild = append(packagesToBuild, &PackageToBuild{
		name:  "main",
		path:  "main",
		files: inputFiles,
	})

	var universe *ast.Scope = universe.CreateUniverse()
	fset = token.NewFileSet()
	sema.Fset = fset
	var builtPackages []*ir.PkgContainer
	for _, _pkg := range packagesToBuild {
		if _pkg.name == "" {
			panic("empty pkg name")
		}
		var asmBasename []byte
		for _, ch := range []byte(_pkg.path) {
			if ch == '/' {
				ch = '.'
			}
			asmBasename = append(asmBasename, ch)
		}
		outFilePath := fmt.Sprintf("%s/%s", workdir, string(asmBasename)+".s")
		var gofiles []string
		var asmfiles []string
		for _, f := range _pkg.files {
			if strings.HasSuffix(f, ".go") {
				gofiles = append(gofiles, f)
			} else if strings.HasSuffix(f, ".s") {
				asmfiles = append(asmfiles, f)
			}

		}
		pkgC := compile(universe, fset, _pkg.path, _pkg.name, gofiles, asmfiles, outFilePath)
		builtPackages = append(builtPackages, pkgC)
	}

	outFilePath := fmt.Sprintf("%s/%s", workdir, "__INIT__.s")
	outAsmFile, err := os.Create(outFilePath)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(outAsmFile, ".text\n")
	fmt.Fprintf(outAsmFile, "# Initializes all packages except for runtime\n")
	fmt.Fprintf(outAsmFile, ".global __INIT__.init\n")
	fmt.Fprintf(outAsmFile, "__INIT__.init:\n")
	for _, _pkg := range builtPackages {
		// A package with no imports is initialized by assigning initial values to all its package-level variables
		//  followed by calling all init functions in the order they appear in the source
		if _pkg.Name != "runtime" {
			fmt.Fprintf(outAsmFile, "  callq %s.__initVars \n", _pkg.Name)
		}
		if _pkg.HasInitFunc {
			fmt.Fprintf(outAsmFile, "  callq %s.init \n", _pkg.Name)
		}
	}
	fmt.Fprintf(outAsmFile, "  ret\n")
	outAsmFile.Close()
}
