## 基本概念

Go 语言中**只有传值**，没有传引用。这是一个重要的基础认知。但 Go 里面的一些类型（如切片Slice、map、channel）虽然是传值，但其内部包含指向底层数据的指针，因此表现得像"传引用"。

**传值 (Pass by Value)**

函数参数传值，将参数的值复制一份传递给函数，函数内部对参数的修改不会影响原始数据。

Go 语言中所有参数默认都是传值。

**传引用 (Pass by Reference) **

Go 语言中实际上没有真正的传引用，通过传递指针(Pointer)来实现类似引用的效果。

函数可以通过指针修改原始数据。

## 基本类型和struct传值机制例子

### 基本类型的传值

基础数据类型 int, float, bool, string 等。

下面看看 int 类型的传值，下面代码例子，

by_value_ref/basic_int.go

```go
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
```

go run 运行代码：

```shell
go run ./basic_int.go

传值 调用前: a = 10, 地址 = 0xc00000a0a8
函数内部: x = 100, 地址 = 0xc00000a0d0
传值 调用后: a = 10, 地址 = 0xc00000a0a8
传指针 调用前: a2 = 12, 地址 = 0xc00000a0d8
函数内部修改后的值: 100
传指针 调用后: a2 = 12, 地址 = 0xc00000a0d8
```

上面例子运行的结果，可以看到，函数内部的 x 值改变了，但是函数 `modifyInt` 调用前后 x 的值没有改变。

这里例子说明函数参数传值，将参数的值复制一份传递给函数，函数内部对参数的修改不会影响原始数据值。

基本类型传指针也是同样。

### 结构体struct传值

struct 结构体的传值，看下面代码例子，

by_value_ref/basic_struct.go，

struct 结构体传值和传指针（**传指针的值**）的情况

```go
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
```

go run 运行后的结果

```shell
go run ./basic_struct.go

Person 赋值 原始: {Name:原始 Age:20}, 地址 = 0xc000094030
ByValue 函数内部：{Name:王麻子 Age:34}, 地址 = 0xc000094060
传值后: {Name:原始 Age:20}
ByPointer 函数内部：&{Name:李四 Age:40}, 指针地址 = 0xc0000a8020， 指向的地址 = 0xc000094030
传指针后: {Name:李四 Age:40}
```

从上面代码可以看到，struct 传值给函数后，函数内部值改变了；函数运行完，原始传入的值没有改变。

struct 传指针给函数后，函数运行完，原始传入的值已经改变了。

## 引用语义的传值

### 切片slice

by_value_ref/slice_demo.go，看代码例子

```go
package main

import (
	"fmt"
)

// 传值
func modifySlice(s []int) {
	s[0] = 999         // 会修改原切片值
	s = append(s, 100) // 可能不会影响原切片（取决于容量）
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
```

go run 运行后

```shell
go run ./slice_demo.go

main start
调用前 原始值: [1 2 3 4 5], len = 5, cap = 5
函数内 传值: [999 2 3 4 5 100], len = 6, cap = 10
传值 函数调用后: [999 2 3 4 5]
函数内 传指针后 append增加: [999 2 3 4 5 200]
传指针 函数调用后: [999 2 3 4 5 200]
```

从上面嗲嘛例子可以看出，slice传值后，slice值是否改变，要看slice的容量，如果容量够，那么传值后值都改变了。如果容量不够，append增加值，函数内部slice值改变，但是函数调用完后外部slice值没改变，没增加 100 。

slice 传指针，函数内部和调用后，slice 值都改变了。

### map

by_value_ref/map_demo.go，看代码例子

```go
package main

import (
	"fmt"
)

func modifyMap(m map[string]int) {
	m["key1"] = 100 // 会修改原map
	m["new1"] = 200 // 会修改原map
	m["new2"] = 300
}

func main() {
	myMap := map[string]int{"key1": 1, "key2": 2}
	fmt.Printf("函数调用前: %v\n", myMap)

	modifyMap(myMap)
	fmt.Printf("函数调用后: %v\n", myMap)
}
```

go run运行后

```shell
go run ./map_demo.go

函数调用前: map[key1:1 key2:2]
函数调用后: map[key1:100 key2:2 new1:200 new2:300]
```

传 map 函数调用后，直接修改了 map 的值。

### channel

channel_demo.go

```go
package main

import (
    "fmt"
)

func sendToChannel(ch chan int) {
    ch <- 42 // 会修改原channel
}

func main() {
    ch := make(chan int, 1)
    sendToChannel(ch)
    fmt.Println(<-ch) // 输出: 42
}
```

go run 运行输出 42，修改了值。也就是说传 channel 就会修改值



还有个综合例子 ：by_value_and_ref_all_demo.go

## 什么时候传值 什么时候传指针

### 判断条件

**判断优先传值的一些条件**

- 数据是基础数据类型（int, string, bool等）
- 函数只读取数据，不修改值
- 小型结构体（几个字段）
- 在并发环境中，需要注意数据安全

**判断优先传指针的一些条件**

- 数据是大结构体或包含大量数据
- 函数需要修改原始数据
- 性能是关键考虑因素
- 需要避免内存浪费

更清晰的判断标准

| 标准         | 传值           | 传指针                       |
| ------------ | -------------- | ---------------------------- |
| **大小**     | < 64字节       | >= 64字节                    |
| **是否修改** | 不需要修改原值 | 需要修改原值                 |
| **性能**     | 小对象复制快   | 大对象避免复制               |
| **并发**     | 天然并发安全   | 需要额外同步                 |
| **nil处理**  | 不适用         | 可以表示"无值"               |
| **接口实现** | 值类型实现接口 | 指针类型实现接口（保持一致） |

代码例子：go_decision_demo.go

```go
package main

import (
	"fmt"
	"math"
	"unsafe"
)

// 决策辅助工具
type DecisionHelper struct{}

func (d DecisionHelper) ShouldUsePointer(data interface{}, willModify bool, isLarge bool) bool {
	// 简化判断逻辑
	return willModify || isLarge
}

func (d DecisionHelper) EstimateSize(data interface{}) uintptr {
	switch v := data.(type) {
	case int:
		return unsafe.Sizeof(v)
	case string:
		return unsafe.Sizeof(v) + uintptr(len(v))
	case []int:
		return unsafe.Sizeof(v) + uintptr(len(v))*unsafe.Sizeof(int(0))
	case map[string]int:
		return unsafe.Sizeof(v) + uintptr(len(v))*64 // 估算map大小
	default:
		return unsafe.Sizeof(v)
	}
}

// 实际开发中的决策示例

// 场景1：配置结构 - 通常很大，使用指针
type Config struct {
	ServerName    string            `json:"server_name"`
	Port          int               `json:"port"`
	DatabaseURL   string            `json:"database_url"`
	Timeout       int               `json:"timeout"`
	Features      map[string]bool   `json:"features"`
	RetrySettings struct {
		MaxAttempts int `json:"max_attempts"`
		Backoff      int `json:"backoff"`
	} `json:"retry_settings"`
}

func LoadConfig(cfg *Config) error {
	// 大结构体，使用指针
	*cfg = Config{
		ServerName:  "localhost",
		Port:        8080,
		DatabaseURL: "postgresql://localhost/db",
		Features:    map[string]bool{"cache": true, "ssl": false},
	}
	return nil
}

// 场景2：用户信息 - 中等大小，可能修改，使用指针
type User struct {
	ID       int
	Name     string
	Email    string
	Age      int
	Active   bool
	Metadata map[string]interface{}
}

func UpdateUser(user *User, field string, value interface{}) error {
	// 需要修改原始数据，使用指针
	switch field {
	case "Name":
		user.Name = value.(string)
	case "Age":
		user.Age = value.(int)
	case "Active":
		user.Active = value.(bool)
	default:
		user.Metadata[field] = value
	}
	return nil
}

// 场景3：计算函数 - 只读，使用传值
func CalculateStats(numbers []int) (sum, avg, max, min int) {
	if len(numbers) == 0 {
		return 0, 0, 0, 0
	}
	
	sum = 0
	max = numbers[0]
	min = numbers[0]
	
	for _, num := range numbers {
		sum += num
		if num > max {
			max = num
		}
		if num < min {
			min = num
		}
	}
	
	avg = sum / len(numbers)
	return
}

// 场景4：响应结构 - 传值更清晰
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data"`
}

func CreateSuccessResponse(data interface{}) APIResponse {
	// 返回新值，比返回指针更清晰
	return APIResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

// 场景5：工具函数 - 根据参数大小决定
type Point struct {
	X, Y float64
}

func Distance(p1, p2 Point) float64 {
	// 小结构体，传值
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// 性能监控示例
func PerformanceComparison() {
	decisionHelper := DecisionHelper{}
	
	// 测试不同大小的数据
	smallData := 42
	largeData := make([]int, 10000)
	
	fmt.Printf("小数据大小: %d 字节\n", decisionHelper.EstimateSize(smallData))
	fmt.Printf("大数据大小: %d 字节\n", decisionHelper.EstimateSize(largeData))
	
	// 判断是否使用指针
	shouldUsePointer1 := decisionHelper.ShouldUsePointer(smallData, false, false)
	shouldUsePointer2 := decisionHelper.ShouldUsePointer(largeData, false, true)
	
	fmt.Printf("小数据应该使用指针: %v\n", shouldUsePointer1)
	fmt.Printf("大数据应该使用指针: %v\n", shouldUsePointer2)
}

// 最佳实践示例
type BestPractices struct{}

func (bp BestPractices) HandleUserOperation() {
	// 1. 创建用户 - 返回值
	user := bp.createUser()
	
	// 2. 更新用户 - 指针
	bp.updateUser(&user, "name", "NewName")
	
	// 3. 处理用户列表 - 切片传值（切片本身很小）
	users := []User{{ID: 1}, {ID: 2}, {ID: 3}}
	count, avgAge := bp.calculateUserStats(users)
	
	fmt.Printf("用户统计: count=%d, avgAge=%d\n", count, avgAge)
}

func (bp BestPractices) createUser() User {
	return User{ID: 1, Name: "Default", Age: 25}
}

func (bp BestPractices) updateUser(user *User, field, value string) {
	// 修改原始用户信息
	user.Name = value
}

func (bp BestPractices) calculateUserStats(users []User) (count, avgAge int) {
	if len(users) == 0 {
		return 0, 0
	}
	
	totalAge := 0
	for _, user := range users {
		totalAge += user.Age
		count++
	}
	
	avgAge = totalAge / count
	return
}

func main() {
	fmt.Println("=== Go传值vs传指针决策指南 ===\n")
	
	PerformanceComparison()
	
	fmt.Println("\n=== 实际场景示例 ===")
	
	// 场景1：配置加载
	var config Config
	LoadConfig(&config)
	fmt.Printf("配置加载成功: %+v\n", config)
	
	// 场景2：用户更新
	user := User{ID: 1, Name: "Alice", Age: 25}
	UpdateUser(&user, "Age", 26)
	fmt.Printf("用户更新后: %+v\n", user)
	
	// 场景3：统计计算
	numbers := []int{1, 2, 3, 4, 5}
	sum, avg, max, min := CalculateStats(numbers)
	fmt.Printf("统计结果: sum=%d, avg=%d, max=%d, min=%d\n", sum, avg, max, min)
	
	// 场景4：API响应
	response := CreateSuccessResponse(map[string]interface{}{
		"message": "Hello World",
	})
	fmt.Printf("API响应: %+v\n", response)
	
	// 场景5：点距离计算
	p1 := Point{X: 0, Y: 0}
	p2 := Point{X: 3, Y: 4}
	distance := Distance(p1, p2)
	fmt.Printf("点距离: %.2f\n", distance)
	
	fmt.Println("\n=== 最佳实践演示 ===")
	bestPractices := BestPractices{}
	bestPractices.HandleUserOperation()
}
```

### 最后总结

- **Go只有传值**，但某些类型内部含指针，表现像引用

- **小对象（<64字节）传值**，大对象传指针

- **需要修改原值必须传指针**

- **切片、map、channel虽是传值，但能修改底层数据**

- **方法接收者保持一致性**：要么都用值，要么都用指针

- **并发场景考虑线程安全**：传值天然安全，传指针需加锁