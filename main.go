package main

import (
	"flag"
	"fmt"
	"sync"
)

var mu sync.Mutex
var pkgToObjToType = make(map[string]map[string]objectType)
var pkgToTypeToMethods = make(map[string]map[string]map[string]bool)
var pkgToCtorToRetTypes = make(map[string]map[string][]string)

//---

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func parseArgs() ([]string, []string) {
	var input, externalCallers arrayFlags
	flag.Var(&input, "i", "packages where unused exported symbols will be looked for")
	flag.Var(&externalCallers, "e", "packages who calls input packages")
	flag.Parse()
	return input, externalCallers
}

//---

func printArr(in string, lines []string) {
	fmt.Printf("%s:\n", in)
	for _, line := range lines {
		fmt.Printf("    %s\n", line)
	}
}

func main() {
	input, externalCallers := parseArgs()

	printArr("i", input)
	printArr("e", externalCallers)
	fmt.Println("results:")

	runAnalyzer(
		input,
		HarvestPkgLvlEntities,
		false,
	)
	runAnalyzer(
		append(input, externalCallers...),
		DetectCrossPkgLinks,
		true,
	)

	for file, objs := range pkgToObjToType {
		if len(objs) == 0 {
			continue
		}
		fmt.Println(file)
		for k := range objs {
			fmt.Printf("    %s\n", k)
		}
	}
}
