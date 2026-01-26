package main

import "fmt"

func basicExample() {
    fmt.Println("函数开始执行")
    
    // 声明第一个defer
    defer fmt.Println("第一个defer执行")
    
    fmt.Println("函数中间执行")
    
    // 声明第二个defer，最后声明的 defer 先执行
    defer fmt.Println("第二个defer执行")
    
    fmt.Println("函数即将结束")
}

func main() {
    basicExample()
    // 输出顺序：
    // 函数开始执行
    // 函数中间执行
    // 函数即将结束
    // 第二个defer执行
    // 第一个defer执行
}