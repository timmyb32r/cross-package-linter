package main

import (
	"errors"
	"flag"
	"fmt"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type typeParseError struct {
	error
}

func loadingError(initial []*packages.Package) error {
	var err error
	if n := packages.PrintErrors(initial); n > 1 {
		err = fmt.Errorf("%d errors during loading", n)
	} else if n == 1 {
		err = errors.New("error during loading")
	} else {
		// no errors
		return nil
	}
	all := true
	packages.Visit(initial, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			typeOrParse := err.Kind == packages.TypeError || err.Kind == packages.ParseError
			all = all && typeOrParse
		}
	})
	if all {
		return typeParseError{err}
	}
	return err
}

func filterOutGodror(in []*packages.Package) []*packages.Package {
	newInitial := make([]*packages.Package, 0, len(in))
	for _, el := range in {
		if strings.Contains(el.PkgPath, "godror") {
			continue
		}
		newInitial = append(newInitial, el)
	}
	return newInitial
}

func load(patterns []string, loadTests bool) ([]*packages.Package, error) {
	mode := packages.LoadSyntax | packages.LoadAllSyntax | packages.NeedModule
	conf := packages.Config{
		Mode:  mode,
		Tests: loadTests,
	}
	initial, err := packages.Load(&conf, patterns...)
	initial = filterOutGodror(initial)
	if err == nil {
		if len(initial) == 0 {
			err = fmt.Errorf("%s matched no packages", strings.Join(patterns, " "))
		} else {
			err = loadingError(initial)
		}
	}
	return initial, err
}

func runAnalyzer(args []string, analyzer *analysis.Analyzer, loadTests bool) {
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	pkgs, err := load(args, loadTests)
	if err != nil {
		fmt.Printf("load returned an error: %s", err.Error())
		os.Exit(1)
	}
	type key struct {
		*analysis.Analyzer
		*packages.Package
	}

	type objectFactKey struct {
		obj types.Object
		typ reflect.Type
	}

	type packageFactKey struct {
		pkg *types.Package
		typ reflect.Type
	}

	type action struct {
		once         sync.Once
		a            *analysis.Analyzer
		pkg          *packages.Package
		pass         *analysis.Pass
		isroot       bool
		deps         []*action
		objectFacts  map[objectFactKey]analysis.Fact
		packageFacts map[packageFactKey]analysis.Fact
		result       interface{}
		diagnostics  []analysis.Diagnostic
		err          error
		duration     time.Duration
	}
	actions := make(map[key]*action)

	var mkAction func(a *analysis.Analyzer, pkg *packages.Package) *action
	mkAction = func(a *analysis.Analyzer, pkg *packages.Package) *action {
		k := key{a, pkg}
		act, ok := actions[k]
		if !ok {
			act = &action{a: a, pkg: pkg}

			// Add a dependency on each required analyzers.
			for _, req := range a.Requires {
				act.deps = append(act.deps, mkAction(req, pkg))
			}

			// An analysis that consumes/produces facts
			// must run on the package's dependencies too.
			if len(a.FactTypes) > 0 {
				paths := make([]string, 0, len(pkg.Imports))
				for path := range pkg.Imports {
					paths = append(paths, path)
				}
				sort.Strings(paths) // for determinism
				for _, path := range paths {
					dep := mkAction(a, pkg.Imports[path])
					act.deps = append(act.deps, dep)
				}
			}

			actions[k] = act
		}
		return act
	}
	for _, pkg := range pkgs {
		act := mkAction(analyzer, pkg)
		pass := &analysis.Pass{
			Analyzer:     act.a,
			Fset:         act.pkg.Fset,
			Files:        act.pkg.Syntax,
			OtherFiles:   act.pkg.OtherFiles,
			IgnoredFiles: act.pkg.IgnoredFiles,
			Pkg:          act.pkg.Types,
			TypesInfo:    act.pkg.TypesInfo,
			TypesSizes:   act.pkg.TypesSizes,
			TypeErrors:   act.pkg.TypeErrors,

			ResultOf:          nil,
			Report:            func(d analysis.Diagnostic) { act.diagnostics = append(act.diagnostics, d) },
			ImportObjectFact:  nil,
			ExportObjectFact:  nil,
			ImportPackageFact: nil,
			ExportPackageFact: nil,
			AllObjectFacts:    nil,
			AllPackageFacts:   nil,
		}
		act.pass = pass
		_, _ = analyzer.Run(act.pass)
	}
}
