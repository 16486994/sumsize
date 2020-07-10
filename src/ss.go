package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ( //定义变量组
	dir          *string = flag.String("p", ".", "input directory")    //dir是指针变量：用于指定输入目录，用flag的好处是配置参数方便，第一个参数是配置名称，可以在运行时显示的写出来。运行时可以指定路径go run ss.go -p=c:/windows
	routineCount *int    = flag.Int("c", 10, "concurrency of program") //开启线程数
	verbose      *bool   = flag.Bool("v", false, "print verbose progress")
)

func walkDir(dir string, fileSizes chan int64, concurrency chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, entry := range dirEntries(dir, concurrency) {
		if entry.IsDir() {
			wg.Add(1)
			subdir := filepath.Join(dir, entry.Name())
			go walkDir(subdir, fileSizes, concurrency, wg)
		} else {
			fileSizes <- entry.Size()
		}
	}
}
func dirEntries(dir string, concurrency chan struct{}) []os.FileInfo {
	concurrency <- struct{}{}

	defer func() {
		<-concurrency
	}()

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du: %s\n", err.Error())
		return nil
	}
	return entries
}

func main() {
	flag.Parse()
	tick := make(<-chan time.Time) // 定时channel, 定期打印当前计算的文件总数和文件总大小
	if *verbose {
		tick = time.Tick(500 * time.Millisecond) // 每隔500毫秒tick会收到来自time.Tick的消息
	}

	var totalFiles, totalBytes int64
	var wg sync.WaitGroup

	startTime := time.Now()                          // 记录开始遍历的时间，以计算总的遍历时间
	fileSizesChan := make(chan int64, *routineCount) // 存储每个文件大小的channel

	var concurrencyChan = make(chan struct{}, *routineCount) //控制并发度
	wg.Add(1)
	go walkDir(*dir, fileSizesChan, concurrencyChan, &wg)

	go func() { // 必须起一个goroutine, 以便一边遍历目录一边计算文件大小
		wg.Wait()
		close(fileSizesChan)
	}()

loop:
	for {
		select { // select会在都准备好的情况下随机挑选一个case执行
		case size, ok := <-fileSizesChan: // 只有当close(fileSizes)这句执行到，显示关闭掉channel之后，才会跳出range循环并且这时已经读取完了所有的数据。
			if !ok {
				break loop
			}
			totalFiles++
			totalBytes += size
		case <-tick:
			fmt.Printf("%d files  %f \n", totalFiles, float64(totalBytes))
		}
	}
	fmt.Printf("%d files  %f \n", totalFiles, float64(totalBytes))

	fmt.Println("总的计算时间: ", time.Since(startTime))
}
