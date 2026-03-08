package main

import (
	"fmt"
	"time"
)

// 函数装饰器实现简单示例

// 1. 定义函数类型
type HandlerFunc func(string)

// 2. 定义装饰器：记录执行时间
func WithTiming(h HandlerFunc) HandlerFunc {
	return func(name string) {
		start := time.Now()
		h(name) // 执行原函数
		fmt.Printf("函数 %s 执行耗时: %v\n", name, time.Since(start))
	}
}

// 3. 目标函数
func SayHello(name string) {
	fmt.Println("Hello,", name)
}

func main() {
	// 使用装饰器
	timedHello := WithTiming(SayHello)
	timedHello("World")
}
