package main

import (
	"bufio"
	"container/ring"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// размер буфера
const sizeOfBuffer = 3

// интервал очистки буфера и передачи данных потребителю
const delayTime time.Duration = 5 * time.Second

type ringBuffer struct {
	start  *ring.Ring
	end    *ring.Ring
	isFull bool
	m      sync.RWMutex
}

func (r *ringBuffer) Read() (int, bool) {
	r.m.Lock()
	defer r.m.Unlock()
	// буфер пуст
	if r.end == r.start && !r.isFull {
		return 0, false
	}
	num := r.start.Value.(int)
	r.start = r.start.Next()
	// прочитали из полного буфера
	if r.isFull {
		r.isFull = false
	}
	return num, true
}

func (r *ringBuffer) Write(n int) {
	r.m.Lock()
	if r.isFull {
		// сдвигаем начало кольца
		r.start = r.start.Next()
	}
	r.end.Value = n
	r.end = r.end.Next()
	// буфер заполнен
	if r.end == r.start {
		r.isFull = true
	}
	r.m.Unlock()
}
func NewRingBuffer(size int) *ringBuffer {
	start := ring.New(size)
	end := start
	return &ringBuffer{start: start, end: end, isFull: false, m: sync.RWMutex{}}
}

// оставляем в пайплайне  положительные числа
func filter1(done <-chan struct{}, input <-chan int) <-chan int {
	dataStream := make(chan int)
	go func() {
		defer close(dataStream)
		// читаем канал пока он не будет закрыт
		for num := range input {
			if num < 0 {
				continue
			}
			select {
			case <-done:
				return
			case dataStream <- num:
			}
		}

	}()
	return dataStream
}

// оставляем в пайплайне числа кратные 3
func filter2(done <-chan struct{}, input <-chan int) <-chan int {
	dataStream := make(chan int)
	go func() {
		defer close(dataStream)
		// читаем канал пока он не будет закрыт
		for num := range input {
			if num == 0 || num%3 != 0 {
				continue
			}
			select {
			case <-done:
				return
			case dataStream <- num:
			}
		}

	}()
	return dataStream
}

func main() {
	// Создаем входной канал поставщика, загружаются данные от поставщика в канал
	// Данные получаем из консоли
	inputData := func() (<-chan int, <-chan struct{}) {
		var scanner = bufio.NewScanner(os.Stdin)
		done := make(chan struct{})
		input := make(chan int)
		go func() {
			for scanner.Scan() {
				text := scanner.Text()
				switch text {
				case "exit":
					close(done)
					close(input)
					return
				default:
					num, err := strconv.Atoi(text)
					// получили из консоли не число
					if err != nil {
						continue
					}
					// передаем число в канал
					input <- num
				}
			}
		}()
		return input, done
	}

	bufferStage := func(done <-chan struct{}, input <-chan int) <-chan int {
		dataStream := make(chan int)
		r := NewRingBuffer(sizeOfBuffer)
		go func() {
			for {
				select {
				case <-done:
					return
				// заполнять кольцевой буфер
				case num := <-input:
					r.Write(num)
				}
			}
		}()

		go func() {
			for {
				select {
				case <-done:
					return
				// опустошаем буфер
				// и передаем данные
				case <-time.Tick(delayTime):
					for n, ok := r.Read(); ok; n, ok = r.Read() {
						dataStream <- n
					}
				}
			}
		}()
		return dataStream
	}
	// потребитель. получает данные из буфера и выводит их в консоль
	consumer := func(done <-chan struct{}, input <-chan int) {
		// идем по потоку данных в цикле
		for {
			select {
			case <-done:
				return
			case num := <-input:
				fmt.Println("Получены данные: ", num)
			}
		}
	}
	//канал с данными из консоли и канал для выхода
	input, done := inputData()
	pipeline := bufferStage(done, filter2(done, filter1(done, input)))
	consumer(done, pipeline)
}
