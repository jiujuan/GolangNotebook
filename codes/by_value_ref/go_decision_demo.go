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