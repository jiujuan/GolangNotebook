package main

import (
	"fmt"
	"time"
)

func main() {
	// 调用被测试函数
	doSomething()
}

func doSomething() {
	// 1. 在函数开头记录开始时间
	start := time.Now()

	// 2. 使用 defer 注册一个匿名函数，在函数退出时计算耗时
	defer func() {
		// 3. 计算耗时
		elapsed := time.Since(start)
		fmt.Printf("函数执行耗时: %s\n", elapsed)
	}()

	// 4. 模拟函数实际逻辑（例如：休眠 100 毫秒）
	time.Sleep(100 * time.Millisecond)
	fmt.Println("任务完成")
}