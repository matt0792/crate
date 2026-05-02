package main

import (
	"fmt"

	"github.com/matt0792/crate/crate"
)

func main() {
	crate.Project("test2")

	type item struct {
		Foo string
	}

	fmt.Println(crate.Size())
	fmt.Println(crate.Namespaces())
	fmt.Println(crate.Count("test2"))

	// // for range 100000 {
	// crate.Store("foostore", item{
	// 	Foo: "foo",
	// })
	// // }

	// amt := crate.DeleteBy("foostore", func(item) bool {
	// 	return true
	// })

	// time.Sleep(5 * time.Second)
}
