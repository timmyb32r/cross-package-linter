# cross-package-linter

cross-package-linter - analysis your golang packages and indicates symbols, who defined as exported, but nowhere imported.


# example

```shell
> ./cross-package-linter -i ./pkg/...
i:
    ./pkg/...
e:
results:
/Users/timmyb32r/go/src/cross-package-linter/pkg/packageA
    StructAUnused
    ConstAUnused
    FuncAUnused
    VarAUnused
    NewQ
/Users/timmyb32r/go/src/cross-package-linter/pkg/packageB
    FuncB
```


# usage

```shell
export CGO_ENABLED=0 && ./cross-package-linter -i ~/my_project/internal/... -i ~/my_project/pkg/... -e ~/my_project/cmd/...
```

Allowed to point multiple -e & -i

`-i` - include, include to analysis - for example, project's `pkg` or `internal` directory

`-e` - external, for example, project's `cmd` directory. For cases, when some symbol defined into pkg, and uses only from cmd


# motivation

Why I think it's important - extra exported symbols reduces readability of code.

Golang forces you to expose private symbols, when they are used outside the package (bcs otherwise it won't compile). But golang don't force you to hide public symbols, whenever they are public by accident. This asymmetry lead to reducing code readability, and this linter aimed to overcome this problem.

