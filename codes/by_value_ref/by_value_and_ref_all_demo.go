package main

import (
	"fmt"
)

// 综合例子

// 传值示例
func passByValue(num int) {
	num = 100 // 修改副本
	fmt.Println("函数内部修改后的值:", num)
}

// 传指针示例（模拟传引用）
func passByReference(num *int) {
	*num = 100 // 修改原始值
	fmt.Println("函数内部修改后的值:", *num)
}

// 切片传值示例（特殊情况）
func modifySlice(slice []int) {
	slice[0] = 100           // 会修改原始切片
	slice = append(slice, 4) // 不会影响原始切片
}

// 切片传指针示例
func modifySlicePtr(slice *[]int) {
	(*slice)[0] = 100
	*slice = append(*slice, 4)
}

// map传值示例（特殊情况）
func modifyMap(m map[string]int) {
	m["key"] = 100 // 会修改原始map
}

// 结构体传值
type Person struct {
	Name string
	Age  int
}

func modifyPersonValue(p Person) {
	p.Age = 30
	fmt.Println("函数内部修改后:", p)
}

// 结构体传指针
func modifyPersonPointer(p *Person) {
	p.Age = 30
	fmt.Println("函数内部修改后:", *p)
}

func main() {
	fmt.Println("=== 基础类型传值 vs 传引用 ===")

	// 基础类型传值
	num := 42
	fmt.Println("原始值:", num)
	passByValue(num)
	fmt.Println("调用函数后:", num)

	fmt.Println("\n--- 分隔线 ---")

	// 基础类型传指针
	num2 := 42
	fmt.Println("原始值:", num2)
	passByReference(&num2)
	fmt.Println("调用函数后:", num2)

	fmt.Println("\n=== 切片传值 ===")

	// 切片传值
	slice := []int{1, 2, 3}
	fmt.Println("原始切片:", slice)
	modifySlice(slice)
	fmt.Println("调用函数后:", slice)

	fmt.Println("\n=== 切片传指针 ===")

	// 切片传指针
	slice2 := []int{1, 2, 3}
	fmt.Println("原始切片:", slice2)
	modifySlicePtr(&slice2)
	fmt.Println("调用函数后:", slice2)

	fmt.Println("\n=== Map传值 ===")

	// Map传值
	m := map[string]int{"key": 1}
	fmt.Println("原始Map:", m)
	modifyMap(m)
	fmt.Println("调用函数后:", m)

	fmt.Println("\n=== 结构体传值 vs 传指针 ===")

	// 结构体传值
	person1 := Person{Name: "Alice", Age: 25}
	fmt.Println("原始结构体:", person1)
	modifyPersonValue(person1)
	fmt.Println("传值调用后:", person1)

	fmt.Println("\n--- 分隔线 ---")

	// 结构体传指针
	person2 := Person{Name: "Bob", Age: 25}
	fmt.Println("原始结构体:", person2)
	modifyPersonPointer(&person2)
	fmt.Println("传指针调用后:", person2)
}
