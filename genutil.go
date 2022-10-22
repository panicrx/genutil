package genutil

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"math"
	"os"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

const (
	envGOFILE    = "GOFILE"
	envGOPACKAGE = "GOPACKAGE"
	envGOLINE    = "GOLINE"
)

type Opts struct {
	file string
	pkg  string
	line int
}

type OptsFunc func(opts *Opts)

func FileName(name string) OptsFunc {
	return func(opts *Opts) {
		opts.file = name
	}
}

func PackageName(name string) OptsFunc {
	return func(opts *Opts) {
		opts.pkg = name
	}
}

func Line(line int) OptsFunc {
	return func(opts *Opts) {
		opts.line = line
	}
}

func applyOpts(fs []OptsFunc) (*Opts, error) {
	var ret Opts
	for _, f := range fs {
		f(&ret)
	}

	var ok bool
	if ret.file == "" {
		ret.file, ok = resolveEnvValue(envGOFILE)
		if !ok {
			return nil, errors.New("failed to determine input file")
		}
	}

	if ret.pkg == "" {
		ret.pkg, ok = resolveEnvValue(envGOPACKAGE)
		if !ok {
			return nil, errors.New("failed to determine package name")
		}
	}

	if ret.line == 0 {
		if lineStr, _ := resolveEnvValue(envGOLINE); lineStr != "" {
			_, err := fmt.Sscan(lineStr, &ret.line)
			if err != nil {
				return nil, fmt.Errorf("failed to determine source line: %w", err)
			}
		}
	}

	return &ret, nil
}

func LoadPackageAndFindClosestType(opts ...OptsFunc) (*packages.Package, *ast.File, *types.TypeName, error) {
	pkg, err := LoadPackage(opts...)
	if err != nil {
		return nil, nil, nil, err
	}

	f, tn, err := FindClosestType(pkg, opts...)
	if err != nil {
		return nil, nil, nil, err
	}

	return pkg, f, tn, nil
}

func LoadPackage(opts ...OptsFunc) (*packages.Package, error) {
	op, err := applyOpts(opts)
	if err != nil {
		return nil, err
	}

	pkg, err := loadPackage(op.pkg, op.file)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

func FindClosestType(pkg *packages.Package, opts ...OptsFunc) (*ast.File, *types.TypeName, error) {
	op, err := applyOpts(opts)
	if err != nil {
		return nil, nil, err
	}

	return findTypeDecl(pkg.Fset, pkg.Syntax, pkg.TypesInfo, op.file, op.line)
}

func resolveEnvValue(env string) (string, bool) {
	if env != "" {
		return os.LookupEnv(env)
	}

	return "", false
}

// loadPackage loads the package of file inputFileName.
func loadPackage(pkgName, inputFileName string) (*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax}, fmt.Sprintf("file=%s", inputFileName))
	if err != nil {
		return nil, err
	}

	var ret *packages.Package
	for _, pkg := range pkgs {
		if pkg.Name != pkgName {
			continue
		}

		if ret != nil {
			return nil, fmt.Errorf("multiple packages found with name %s", pkgName)
		}

		ret = pkg
	}

	if ret == nil {
		return nil, fmt.Errorf("no packages found with name %s", pkgName)
	}

	return ret, nil
}

// findTypeDecl find the relevant *types.TypeName from fset & info.
// If name is passed, a type with that name is searched for.
// Otherwise, the first type after line in inputFileName is returned.
// If the next declaration after line in inputFileName is not a *types.TypeName,
// an error is returned.
func findTypeDecl(fset *token.FileSet, syntax []*ast.File, info *types.Info, inputFileName string, line int) (*ast.File, *types.TypeName, error) {
	return findTypeDeclByPosition(fset, syntax, info, inputFileName, line)
}

// findTypeDeclByPosition finds the next *type.TypeName in inputFileName after line
func findTypeDeclByPosition(fset *token.FileSet, syntax []*ast.File, info *types.Info, inputFileName string, line int) (*ast.File, *types.TypeName, error) {
	var ret *types.TypeName
	var closestObject types.Object
	var retFile *ast.File
	closest := math.MaxInt32
	for _, object := range info.Defs {
		if object == nil {
			continue
		}

		p := fset.Position(object.Pos())
		if !sameFile(p.Filename, inputFileName) {
			continue
		}

		if p.Line < line || closest < p.Line {
			continue
		}

		ret = nil // we found something closer than our current closest thing
		closestObject = object

		c, ok := object.(*types.TypeName)
		if !ok {
			continue
		}

		f, err := resolveFile(syntax, c)
		if err != nil {
			return nil, nil, fmt.Errorf("genutil: failed to determine *ast.File: %w", err)
		}

		ret = c
		closest = p.Line
		retFile = f
	}

	if ret == nil {
		if closestObject != nil {
			return nil, nil, fmt.Errorf("failed to determine type: closest declaration is not a named type: %v", closestObject)
		}
		return nil, nil, fmt.Errorf("failed to determine type")
	}

	return retFile, ret, nil
}

func resolveFile(syntax []*ast.File, obj types.Object) (ret *ast.File, err error) {
	for _, file := range syntax {
		p, _ := astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
		switch l := len(p); l {
		case 0, 1:
			continue
		default:
			node := p[l-1]
			f, ok := node.(*ast.File)
			if !ok {
				return nil, fmt.Errorf("genutil: last node is not file: %T", node)
			}

			if ret != nil {
				return nil, errors.New("genutil: multiple files found for position")
			}

			ret = f
		}
	}

	return ret, nil
}

// sameFile determines if a and b point to the same file
func sameFile(a, b string) bool {
	as, err := os.Stat(a)
	if err != nil {
		panic(err)
	}

	bs, err := os.Stat(b)
	if err != nil {
		panic(err)
	}

	return os.SameFile(as, bs)
}
