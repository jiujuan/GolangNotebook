package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	// 错误：没有先调用 Add，直接调用 Done
	// wg.Done()
	// fmt.Println("这里不会执行，因为上一行会panic")

	// 错误：调用 Done 的次数多于 Add
	wg.Add(1)
	wg.Done()
	wg.Done() // 计数器变为 -1，触发 panic
	fmt.Println("这里不会执行2，因为上一行会panic")
}