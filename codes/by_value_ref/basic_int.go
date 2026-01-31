package main

import "fmt"

// int 基本类型传值
func modifyInt(x int) {
    x = 100
    fmt.Printf("函数内部: x = %d, 地址 = %p\n", x, &x)
}

// 传指针示例（模拟传引用）
func modifyIntPointer(num *int) {
    *num = 100 // 修改原始值
    fmt.Println("函数内部修改后的值:", *num)
}

func main() {
    a := 10
    fmt.Printf("传值 调用前: a = %d, 地址 = %p\n", a, &a)

    modifyInt(a)
    fmt.Printf("传值 调用后: a = %d, 地址 = %p\n", a, &a)

    a2 := 12
    fmt.Printf("传指针 调用前: a2 = %d, 地址 = %p\n", a2, &a2)
    modifyIntPointer(&a)
    fmt.Printf("传指针 调用后: a2 = %d, 地址 = %p\n", a2, &a2)

}
