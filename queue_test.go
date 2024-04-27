package queue

import (
	"fmt"
	"testing"
	"time"
)

func TestWait(t *testing.T) {
	// 创建一个队列
	que := New()
	// 开启10个请求
	for i := 0; i < 10; i++ {
		que.Push(i)
	}
	go func() {
		for {
			fmt.Println(que.Pop().(int))
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			fmt.Println(que.Pop().(int))
			time.Sleep(1 * time.Second)
		}
	}()

	que.Wait()
	fmt.Println("down")
}

func TestCLose(t *testing.T) {
	que := New()

	for i := 0; i < 10; i++ {
		que.Push(i)
	}

	go func() {
		for {
			v := que.Pop()
			if v != nil {
				fmt.Println(v.(int))
				time.Sleep(1 * time.Second)
			}
		}
	}()

	go func() {
		for {
			v := que.Pop()
			if v != nil {
				fmt.Println(v.(int))
				time.Sleep(1 * time.Second)
			}
		}
	}()

	que.Close()
	que.Wait()
	fmt.Println("down")
}
