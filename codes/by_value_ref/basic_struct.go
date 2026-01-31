package main

import (
	"fmt"
)

type Person struct {
	Name string
	Age  int
}

// 传值：会复制整个结构体
func modifyPersonByValue(p Person) {
	p.Name = "王麻子"
	p.Age = 34
	fmt.Printf("ByValue 函数内部：%+v, 地址 = %p\n", p, &p)
}

// 传指针：只复制指针（8字节）
func modifyPersonByPointer(p *Person) {
	p.Name = "李四"
	p.Age = 40
	fmt.Printf("ByPointer 函数内部：%+v, 指针地址 = %p， 指向的地址 = %p\n", p, &p, p)
}

func main() {
	person := Person{Name: "原始", Age: 20}

	fmt.Printf("Person 赋值 原始: %+v, 地址 = %p\n", person, &person)

	modifyPersonByValue(person)
	fmt.Printf("传值后: %+v\n", person) // 值未改变

	modifyPersonByPointer(&person)
	fmt.Printf("传指针后: %+v\n", person) // 值已改变
}
