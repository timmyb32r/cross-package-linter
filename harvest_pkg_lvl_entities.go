package main

import (
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"path"
	"strings"
	"unicode"
)

var HarvestPkgLvlEntities = &analysis.Analyzer{
	Name:     "cross_package_linter",
	Doc:      `-`,
	Run:      runHarvestPkgLvlEntities,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

type objectType int

const (
	objFunc objectType = iota + 1
	objConst
	objVar
	objType
)

func extractPkgLevelObjects(pass *analysis.Pass) map[string]objectType {
	objToType := make(map[string]objectType)

	names := pass.Pkg.Scope().Names()
	for _, name := range names {
		currNameObj := pass.Pkg.Scope().Lookup(name)
		if !currNameObj.Exported() {
			continue
		}
		switch t := currNameObj.(type) {
		case *types.Var:
			objToType[t.Name()] = objVar
		case *types.Const:
			objToType[t.Name()] = objConst
		case *types.TypeName:
			//q := t.Type().Underlying()
			//switch q.(type) {
			//case *types.Interface:
			//	fmt.Println("!")
			//case *types.Struct:
			//	fmt.Println("!")
			//case *types.Basic: // it's when alias to basic type
			//	fmt.Println("!")
			//case *types.Slice: // it's when alias to slice of basic types
			//	fmt.Println("!")
			//case *types.Signature: // it's when alias to func
			//	fmt.Println("!")
			//case *types.Map: // it's when alias to map
			//	fmt.Println("!")
			//case *types.Pointer: // it's when alias to pointer (on another typename, for example)
			//	fmt.Println("!")
			//default:
			//	panic("!")
			//}
			//fmt.Println(q)
			objToType[t.Name()] = objType
		case *types.Func:
			objToType[t.Name()] = objFunc
		default:
			panic("!")
		}
	}

	return objToType
}

func isExportedName(objName string) bool {
	return unicode.IsUpper(rune(objName[0]))
}

func runHarvestPkgLvlEntities(pass *analysis.Pass) (interface{}, error) {
	mu.Lock()
	defer mu.Unlock()

	var currPackageFullPath string
	for _, file := range pass.Files {
		pathToFile := getPathToFile(pass, file)
		currPackageFullPath = path.Dir(pathToFile)
		break
	}

	objToType := extractPkgLevelObjects(pass)
	typeToMethods := make(map[string]map[string]bool)
	ctorToOutTypes := make(map[string][]string)

	for _, file := range pass.Files {
		pathToFile := getPathToFile(pass, file)
		if strings.HasSuffix(pathToFile, "_test.go") || strings.HasSuffix(pathToFile, "_mock.go") {
			continue
		}
		if strings.Contains(pathToFile, "/go/tests/") {
			continue
		}
		if isProtoCodeGeneratedFile(pathToFile) {
			continue
		}

		//------------------------------------------------------------------------
		// handle methods

		ast.Inspect(file, func(node ast.Node) bool {
			if funcDecl, ok := node.(*ast.FuncDecl); ok {
				if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 1 {
					panic("!")
				}
				if funcDecl.Recv != nil && len(funcDecl.Recv.List) != 0 {
					var objName string
					switch e := funcDecl.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						switch ee := e.X.(type) {
						case *ast.Ident: // func (r *ChannelReader) Close()
							objName = ee.Name
						case *ast.IndexExpr: // func (s *Set[T]) Add(value T)
							switch eee := ee.X.(type) {
							case *ast.Ident:
								objName = eee.Name
							default:
								panic("!")
							}
						case *ast.IndexListExpr: // func (d *MapIter[T, R]) Value() (R, error)
							switch eee := ee.X.(type) {
							case *ast.Ident:
								objName = eee.Name
							default:
								panic("!")
							}
						default:
							panic("!")
						}
					case *ast.IndexExpr: // func (c Comparator[T]) Compare() {...}
						switch ee := e.X.(type) {
						case *ast.Ident:
							objName = ee.Name
						default:
							panic("!")
						}
					case *ast.Ident: // func (v Values) GetTS() time.Time {...}
						objName = e.Name
					default:
						panic("!")
					}
					if objName == "" {
						panic("!")
					}
					methodName := funcDecl.Name.Name

					//-------------------------------------------------------------

					if isExportedName(methodName) {
						if _, ok := typeToMethods[currPackageFullPath]; !ok {
							typeToMethods = make(map[string]map[string]bool)
						}
						if _, ok := typeToMethods[objName]; !ok {
							typeToMethods[objName] = make(map[string]bool)
						}
						typeToMethods[objName][methodName] = true
					}
				}
			}
			return true
		})

		//------------------------------------------------------------------------
		// handle constructors

		ast.Inspect(file, func(node ast.Node) bool {
			if funcDecl, ok := node.(*ast.FuncDecl); ok {
				currCtorName := funcDecl.Name.String()
				if funcDecl.Type.Results == nil {
					return true
				}
				if isExportedName(currCtorName) {
					for _, el := range funcDecl.Type.Results.List {
						if ident, ok := el.Type.(*ast.StarExpr); ok {
							retTypeName := "_"
							switch retType := ident.X.(type) {
							case *ast.Ident:
								retTypeName = retType.Name
							case *ast.IndexExpr:
								if retTypeNameQ, ok := retType.X.(*ast.Ident); ok {
									retTypeName = retTypeNameQ.Name
								}
							}
							if _, ok := ctorToOutTypes[currCtorName]; !ok {
								ctorToOutTypes[currCtorName] = make([]string, 0)
							}
							if isExportedName(retTypeName) {
								ctorToOutTypes[currCtorName] = append(ctorToOutTypes[currCtorName], retTypeName)
							}
						}
					}
				}
			}
			return true
		})
	}

	typesAlreadyExported := make(map[string]bool)
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if funcDecl, ok := node.(*ast.FuncDecl); ok {
				funcName := funcDecl.Name.Name
				isExportedFunc := unicode.IsUpper(rune(funcName[0]))
				if !isExportedFunc {
					return true
				}
				if funcDecl == nil || funcDecl.Type == nil || funcDecl.Type.Results == nil || funcDecl.Type.Results.List == nil {
					return true
				}
				results := funcDecl.Type.Results.List
				for _, outputEl := range results {
					if ident, ok := outputEl.Type.(*ast.Ident); ok {
						if ident != nil {
							methodName := ident.Name
							if unicode.IsUpper(rune(methodName[0])) {
								typesAlreadyExported[ident.Name] = true
							}
						}
					}
				}
			}
			return true
		})
	}

	// fill results

	for k, v := range objToType {
		if _, ok := pkgToObjToType[currPackageFullPath]; !ok {
			pkgToObjToType[currPackageFullPath] = make(map[string]objectType)
		}
		if typesAlreadyExported[k] {
			continue
		}
		pkgToObjToType[currPackageFullPath][k] = v
	}
	for k, v := range typeToMethods {
		if _, ok := pkgToTypeToMethods[currPackageFullPath]; !ok {
			pkgToTypeToMethods[currPackageFullPath] = make(map[string]map[string]bool)
		}
		if typesAlreadyExported[k] {
			continue
		}
		pkgToTypeToMethods[currPackageFullPath][k] = v
	}
	for k, v := range ctorToOutTypes {
		if _, ok := pkgToCtorToRetTypes[currPackageFullPath]; !ok {
			pkgToCtorToRetTypes[currPackageFullPath] = make(map[string][]string)
		}
		if typesAlreadyExported[k] {
			continue
		}
		pkgToCtorToRetTypes[currPackageFullPath][k] = v
	}

	return nil, nil
}
