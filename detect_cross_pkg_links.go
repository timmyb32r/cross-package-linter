package main

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var DetectCrossPkgLinks = &analysis.Analyzer{
	Name:     "cross_package_linter",
	Doc:      `-`,
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func getPkgName(in ast.Expr) string {
	switch ww := in.(type) {
	case *ast.Ident: // aws_credentials.Credentials
		return ww.Name
	case *ast.SelectorExpr: // buildinfo.Info.ArcadiaSourceRevision
		return getPkgName(ww.X)
	case *ast.CallExpr:
		return getPkgName(ww.Fun)
	case *ast.IndexExpr: // s.stats[key].Inc() -> s
		return getPkgName(ww.X)
	case *ast.TypeAssertExpr: // req.(*console.ExecuteOperationRequest).Operation -> req
		return getPkgName(ww.X)
	case *ast.ParenExpr: // (Kafka)
		return getPkgName(ww.X)
	case *ast.StarExpr: // unsafe.Pointer(&pattern)).Data
		return ""
	case *ast.CompositeLit: // protojson.MarshalOptions{UseProtoNames: true}.Marshal(logSafeMessage) - speaking about 'UseProtoNames'
		return ""
	case *ast.UnaryExpr: // &net.Dialer{
		return getPkgName(ww.X)
	default:
		panic("!")
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	mu.Lock()
	defer mu.Unlock()

	cacheObj := newObject()
	for k := range pkgToObjToType {
		cacheObj.add(k)
	}
	for k := range pkgToTypeToMethods {
		cacheObj.add(k)
	}

	for _, file := range pass.Files {
		imports := extractImports(file) // shortName->importPath
		ast.Inspect(file, func(node ast.Node) bool {
			if selectorExpr, ok := node.(*ast.SelectorExpr); ok {
				pkgName := getPkgName(selectorExpr)
				if pkgName == "" {
					return true
				}
				objName := selectorExpr.Sel.Name // objName
				if importFullPath, ok := imports[pkgName]; ok {
					fullPath := cacheObj.fullPathByShortPath(importFullPath)
					if fullPath == "" {
						return true
					}

					//------------------------------------------------------
					// remove obj

					if _, ok := pkgToObjToType[fullPath]; ok {
						if _, ok := pkgToObjToType[fullPath][objName]; ok {
							delete(pkgToObjToType[fullPath], objName)
						}
					}

					//------------------------------------------------------
					// if object is constructor - then remove all return types

					if _, ok := pkgToCtorToRetTypes[fullPath]; ok {
						if retTypes, ok := pkgToCtorToRetTypes[fullPath][objName]; ok {
							for _, retType := range retTypes {
								delete(pkgToObjToType[fullPath], retType)
							}
						}
					}
				}
			}
			return true
		})
	}

	return nil, nil
}
