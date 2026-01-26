package main

import (
    "fmt"
    "log"
)

func safeFunc() {
    // 必须在 panic 之前 defer
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("捕获到异常:", r) // 异常被处理，程序不崩溃
            log.Println("发生panic，已恢复")
        }
    }()

    fmt.Println("执行正常逻辑...")
    panic("发生严重错误") // 抛出异常
    fmt.Println("这行代码不会被执行")
}

func main() {
    safeFunc()
    fmt.Println("程序正常结束")
}
/**
 运行程序： go run .\recover.go

 输出：

执行正常逻辑...
捕获到异常: 发生严重错误
2026/01/26 23:44:08 发生panic，已恢复
程序正常结束 
 **/