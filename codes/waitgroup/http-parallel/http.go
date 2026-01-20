package main

import (
	"fmt"
	"net/http"
	"sync"
	"io/ioutil"
)

func main() {
	urls := []string{"https://www.so.com", "https://www.baidu.com"}

	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			resp, err := http.Get(url)
			if err != nil {
				fmt.Println("Error 1:", err)
				return
			}

			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error 2:", err)
			}
			fmt.Println(url, "StatusCode: ", resp.StatusCode)
			fmt.Println("len(body): ", len(body))

		}(url)
	}

	wg.Wait() // 等待所有请求完成
	fmt.Printf("====end %s \n", "END！===")
}
