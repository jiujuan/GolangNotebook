package main

import (
    "fmt"
    "time"
)

func trackExecutionTime(name string) func() {
    start := time.Now()
    fmt.Printf("开始执行: %s\n", name)
    
    // 返回一个闭包作为defer
    return func() {
        duration := time.Since(start)
        fmt.Printf("函数 %s 执行完成，耗时: %v\n", name, duration)
    }
}

func importantFunction() {
    defer trackExecutionTime("importantFunction")()
    
    // 模拟耗时操作
    time.Sleep(100 * time.Millisecond)
    fmt.Println("重要函数执行中...")
    time.Sleep(100 * time.Millisecond)
}

func logFunctionCall() {
    defer func() {
        fmt.Println("函数执行完毕，清理工作完成")
    }()
    
    fmt.Println("函数正在执行...")
    // 模拟可能发生的错误
    panic("模拟错误")
}

func main() {
    importantFunction()
    fmt.Println("---")
    logFunctionCall()
}