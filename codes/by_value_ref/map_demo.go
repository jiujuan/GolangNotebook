package main

import (
	"fmt"
)

func modifyMap(m map[string]int) {
	m["key1"] = 100 // 会修改原map
	m["new1"] = 200 // 会修改原map
	m["new2"] = 300
}

func main() {
	myMap := map[string]int{"key1": 1, "key2": 2}
	fmt.Printf("函数调用前: %v\n", myMap)

	modifyMap(myMap)
	fmt.Printf("函数调用后: %v\n", myMap)
}
