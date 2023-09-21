package main

import (
	"fmt"
	"strconv"
	//"time"
	"sort"
	"strings"
	"sync"
)

var md5Mutex *sync.Mutex

// сюда писать код

func main() {
	fmt.Printf("start")
	for i := 0; i < 7; i++ {
		SimpleTest(strconv.Itoa(i))
	}
}

//SimpleTest - sync pipeline
func SimpleTest(data string) {
	fmt.Println(data + " SingleHash data " + data)
	result1 := DataSignerMd5(data)
	fmt.Println(data + " SingleHash md5(data) " + result1)
	result2 := DataSignerCrc32(result1)
	fmt.Println(data + " SingleHash crc32(md5(data)) " + result2)
	result3 := DataSignerCrc32(data)
	fmt.Println(data + " SingleHash crc32(data) " + result3)
	result4 := result3 + "~" + result2
	fmt.Println(data + " SingleHash result " + result4)

	mResults := make([]string, 6)
	resultFull := ""

	for i := 0; i < 6; i++ {
		st := strconv.Itoa(i)
		mResults[i] = DataSignerCrc32(st + result4)
		fmt.Println(result4 + " MultiHash: crc32(th+step1)) " + st + " " + mResults[i])
		resultFull += mResults[i]
	}

	fmt.Println(result4 + " MultiHash result:\n" + resultFull)
}

//ExecutePipeline - async pipeline
func ExecutePipeline(jobs ...job) {

	wg := &sync.WaitGroup{}
	md5Mutex = &sync.Mutex{}
	in := make(chan interface{})

	for _, job := range jobs {
		//fmt.Println("job " + strconv.Itoa(idx))
		wg.Add(1)
		out := make(chan interface{})
		go pipWorker(job, in, out, wg)
		in = out
	}
	wg.Wait()
}

func pipWorker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	job(in, out)
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for val := range in {
		//fmt.Println("sh " + convertToStr(val))
		wg.Add(1)
		go singleWorker(val, out, wg)
	}
	wg.Wait()
}

func singleWorker(val interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	data := convertToStr(val)

	crc32Chan := make(chan string)
	crc32Md5Chan := make(chan string)
	go crc32Worker(data, crc32Chan)

	md5Mutex.Lock()
	md5Result := DataSignerMd5(data)
	md5Mutex.Unlock()
	go crc32Worker(md5Result, crc32Md5Chan)

	crc32Result := <-crc32Chan
	close(crc32Chan)
	crc32Md5Result := <-crc32Md5Chan
	close(crc32Md5Chan)

	result := crc32Result + "~" + crc32Md5Result
	out <- result
}

func convertToStr(val interface{}) string {
	toStr, ok := val.(string)
	if ok {
		return toStr
	}
	toInt, ok := val.(int)
	if ok {
		return strconv.Itoa(toInt)
	}
	panic("Error convertToStr!")
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for val := range in {
		//fmt.Println("mh " + convertToStr(val))
		wg.Add(1)
		go multiWorker(val, out, wg)
	}
	wg.Wait()
}

func multiWorker(val interface{}, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	count := 6
	data := convertToStr(val)
	resultCh := make(chan MultiHashOneData, count)
	for i := 0; i < count; i++ {
		task := MultiHashOneData{
			Number: i,
			Data:   strconv.Itoa(i) + data,
		}
		go multiCrc32Worker(task, resultCh)
	}

	resultArr := make([]MultiHashOneData, count)
	recvCount := 0
	for res := range resultCh {
		resultArr[res.Number] = res
		recvCount++
		if recvCount == count {
			close(resultCh)
		}
	}
	result := ""
	for _, part := range resultArr {
		result += part.Data
	}
	//fmt.Println("multihash collected " + result)
	out <- result
}

func multiCrc32Worker(val MultiHashOneData, out chan MultiHashOneData) {
	val.Data = DataSignerCrc32(val.Data)
	out <- val
}

type MultiHashOneData struct {
	Number int
	Data   string
}

func crc32Worker(val string, out chan string) {
	out <- DataSignerCrc32(val)
}

func crc32WorkerWg(val string, out chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	out <- DataSignerCrc32(val)
}

func md5Worker(val string, out chan string, mu *sync.Mutex) {
	mu.Lock()
	result := DataSignerMd5(val)
	mu.Unlock()
	out <- result
}

func CombineResults(in, out chan interface{}) {
	resultArr := make([]string, 0)
	for val := range in {
		data := val.(string)
		//fmt.Println("CombineResults " + data)
		resultArr = append(resultArr, data)
	}
	sort.Strings(resultArr)
	result := strings.Join(resultArr, "_")
	//fmt.Println("CombineResults FULL " + result)
	out <- result
}
