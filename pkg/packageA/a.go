package packageA

var VarA = 1
var VarAUnused = 2

const ConstA = 1
const ConstAUnused = 2

type StructA struct{}
type StructAUnused struct{}

func FuncA()       {}
func FuncAUnused() {}

//---

type Obj struct{}

func (o *Obj) MethodA()       {}
func (o *Obj) MethodAUnused() {}

func NewQ() *Obj {
	return nil
}
