package main

import (
	"fmt"
	"sync"
)

type SafeMap struct {
	mu sync.Mutex
	m map[string]int
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		m: make(map[string]int),
	}
}

// Set 用于设置键值对
func (s *SafeMap) Set(key string, value int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

// Get 用于获取健的值
func (s *SafeMap) Get(key string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok :=s.m[key]
	return val, ok
}

// Delete 方法用于删除指定键
func (s *SafeMap) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

// Keys 方法返回所有键的切片
func (s *SafeMap) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys =append(keys, k)
	}
	return keys
}

func main() {
	s :=NewSafeMap()

	// 并发写数据，启动 30 个goroutine写
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			s.Set(key, id*10)
		}(i)
	}

	wg.Wait()

	for _, k:= range s.Keys() {
		if v, ok := s.Get(k); ok {
			fmt.Printf("%s = %d \n", k, v) // 打印所有数据
		}
	}
}