package queue

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/eapache/queue.v1"
)

// 自定义 队列
type Queue struct {
	sync.Mutex
	popable *sync.Cond
	buffer  *queue.Queue
	closed  bool
	count   int32
	cc      chan interface{}
	once    sync.Once
}

// 创建 自定义队列
func New() *Queue {
	ch := &Queue{
		buffer: queue.New(),
	}
	ch.popable = sync.NewCond(&ch.Mutex)
	return ch
}

// Len 返回队列长度
func (q *Queue) Len() int {
	return (int)(atomic.LoadInt32(&q.count))
}

// Close 关闭队列
func (q *Queue) Close() {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	if !q.closed {
		q.closed = true
		atomic.StoreInt32(&q.count, 0)
		q.popable.Broadcast()
	}
}

// IsClose 验证队列是否关闭 (暴露 closed 字段)
func (q *Queue) IsClose() bool {
	return q.closed
}

// Wait 等待队列消费完成
func (q *Queue) Wait() {
	for {
		if q.closed || q.Len() == 0 {
			break
		}
		runtime.Gosched() // 用于让出CPU时间片
	}
}

// Pop 从队列中取出（阻塞模式）
func (q *Queue) Pop() (v interface{}) {

	c := q.popable

	q.Mutex.Lock()
	defer q.Mutex.Unlock()

	for q.Len() == 0 && !q.closed {
		c.Wait()
	}

	// 如果队列已关闭 立即返回
	if q.closed {
		return
	}

	if q.Len() > 0 {
		buffer := q.buffer
		v = buffer.Peek()
		buffer.Remove()
		atomic.AddInt32(&q.count, -1)
	}

	return
}

// TryPop 试着从队列取出（非阻塞模式）返回 ok == false 表示空
func (q *Queue) TryPop() (v interface{}, ok bool) {

	q.Mutex.Lock()
	defer q.Mutex.Unlock()

	if q.Len() > 0 {
		buffer := q.buffer
		v = buffer.Peek()
		buffer.Remove()
		atomic.AddInt32(&q.count, -1)
		ok = true
	} else if q.closed {
		ok = true
	}

	return
}

// TryPopTimeout 试着从队列取出（塞模式+timeout）返回 ok == false 表示超时
func (q *Queue) TryPopTimeout(tm time.Duration) (v interface{}, ok bool) {
	q.once.Do(func() {
		q.cc = make(chan interface{}, 1)
	})

	// 从队列中取出，通过通道返回
	go func(v *chan interface{}) {
		c := q.popable

		q.Mutex.Lock()
		defer q.Mutex.Unlock()

		for q.Len() == 0 && !q.closed {
			c.Wait()
		}

		if q.closed {
			*v <- nil
			return
		}

		if q.Len() > 0 {
			buffer := q.buffer
			tmp := buffer.Peek()
			buffer.Remove()
			atomic.AddInt32(&q.count, -1)
			*v <- tmp
		} else {
			*v <- nil
		}
	}(&q.cc)

	ok = true
	timeout := time.After(tm)
	select {
	case v = <-q.cc:
	case <-timeout:
		if !q.closed {
			q.popable.Signal()
		}
		ok = false
	}
	return
}

// Push 插入到队列中 （非阻塞模式）
func (q *Queue) Push(v interface{}) {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	if !q.closed {
		q.buffer.Add(v)
		atomic.AddInt32(&q.count, 1)
		q.popable.Signal()
	}
}
