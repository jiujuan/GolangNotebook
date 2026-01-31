package main

import (
	"fmt"
)

// 传值
func modifySlice(s []int) {
	s[0] = 999         // 会修改原切片值
	s = append(s, 100) // 可能不会修改原切片值（取决于容量）
	fmt.Printf("函数内 传值: %v, len = %d, cap = %d\n", s, len(s), cap(s))
}

// 传指针
func modifySlicePointer(s *[]int) {
	*s = append(*s, 200)
	fmt.Printf("函数内 传指针后 append增加: %v\n", *s)
}

func main() {
	fmt.Println("main start")

	slice := []int{1, 2, 3, 4, 5}
	fmt.Printf("调用前 原始值: %v, len = %d, cap = %d\n", slice, len(slice), cap(slice))

	// 传值
	modifySlice(slice)
	fmt.Printf("传值 函数调用后: %v\n", slice)

	// 如果要在函数内修改切片本身，需要传指针
	modifySlicePointer(&slice)
	fmt.Printf("传指针 函数调用后: %v\n", slice)
}
