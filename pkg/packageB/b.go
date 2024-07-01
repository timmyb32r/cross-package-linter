package packageB

import (
	packageAA "cross-package-linter/pkg/packageA"
	"fmt"
)

func init() {
	fmt.Println(packageAA.VarA)
	fmt.Println(packageAA.ConstA)
}

type structB struct {
	a packageAA.StructA
}

func FuncB() { packageAA.FuncA() }

func foo() {
	obj := &packageAA.Obj{}
	obj.MethodA()
}
