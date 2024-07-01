package main

import (
	"strings"
)

type object struct {
	paths map[string]bool
	cache map[string]string
}

func (o *object) add(in string) {
	o.paths[in] = true
}

func (o *object) fullPathByShortPath(in string) string {
	if result, ok := o.cache[in]; ok {
		return result
	}
	if !strings.Contains(in, "/") {
		return ""
	}
	if strings.HasPrefix(in, "k8s") ||
		strings.HasPrefix(in, "a.yandex-team.ru/library") ||
		strings.HasPrefix(in, "google.golang.org") ||
		strings.HasPrefix(in, "golang.org") ||
		in == "encoding/gob" ||
		in == "crypto/tls" ||
		in == "hash/fnv" ||
		in == "database/sql" ||
		strings.HasPrefix(in, "github.com") ||
		in == "encoding/json" ||
		in == "transfer_manager/go/proto/api" ||
		in == "net/http" {
		return ""
	}
	if strings.HasPrefix(in, "a.yandex-team.ru/") {
		in = strings.TrimPrefix(in, "a.yandex-team.ru/")
	}
	count := 0
	result := ""
	for currPath := range o.paths {
		if strings.HasSuffix(currPath, "/"+in) {
			count++
			result = currPath
		}
	}
	if count != 1 {
		//panic("!")
		//fmt.Println(in)
		return ""
	}
	o.cache[in] = result
	return result
}

func newObject() *object {
	return &object{
		paths: make(map[string]bool),
		cache: make(map[string]string),
	}
}
