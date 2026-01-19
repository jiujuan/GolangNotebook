package info

import "fmt"

type Student struct {
	Name  string
	Age   int
	Grade string
}

func GetInfo(stu *Student) {
	fmt.Printf("student info: name=%s, age=%d, grade=%s \n", stu.Name, stu.Age, stu.Grade)
}

func SetInfo(name string, age int, grade string) *Student {
	var stu Student
	stu.Name = name
	stu.Age = age
	stu.Grade = grade
	return &stu
}
