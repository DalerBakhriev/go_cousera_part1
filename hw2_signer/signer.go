package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SingleHash ...
var SingleHash = func(inputChan, outputChan chan interface{}) {

	wg0 := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for input := range inputChan {
		wg1 := &sync.WaitGroup{}
		wg0.Add(1)

		go func(inputStr string) {
			defer wg0.Done()

			var crc32Res string
			var md5Res string
			wg1.Add(1)

			go func(inputStr1 string) {
				defer wg1.Done()
				crc32Res = DataSignerCrc32(inputStr1)
			}(inputStr)

			wg1.Add(1)

			go func(inputStr2 string) {
				defer wg1.Done()
				mu.Lock()
				md5Res = DataSignerMd5(inputStr2)
				mu.Unlock()
				md5Res = DataSignerCrc32(md5Res)
			}(inputStr)
			wg1.Wait()
			outputChan <- crc32Res + "~" + md5Res
		}(strconv.Itoa(input.(int)))
	}
	wg0.Wait()
}

// MultiHash ...
var MultiHash = func(inputChan, outputChan chan interface{}) {
	thArray := []int{0, 1, 2, 3, 4, 5}

	wg1 := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for input := range inputChan {
		wg1.Add(1)

		go func(inputStr string) {
			var multiHashRes string
			hashMap := make(map[int]string, len(thArray))
			wg := &sync.WaitGroup{}

			for _, th := range thArray {
				wg.Add(1)
				go func(number int) {
					datToInsert := DataSignerCrc32(strconv.Itoa(number) + inputStr)
					mu.Lock()
					hashMap[number] = datToInsert
					mu.Unlock()
					wg.Done()
				}(th)
			}
			wg.Wait()

			keysArr := make([]int, 0, len(thArray))
			for key := range hashMap {
				keysArr = append(keysArr, key)
			}
			sort.Ints(keysArr)

			for _, key := range keysArr {
				multiHashRes += hashMap[key]
			}
			outputChan <- multiHashRes
			wg1.Done()
		}(input.(string))
	}
	wg1.Wait()
}

// CombineResults ...
var CombineResults = func(inputChan, outputChan chan interface{}) {
	var resArr []string
	for result := range inputChan {
		res, ok := result.(string)
		if !ok {
			fmt.Errorf("input data has a wrong type")
			return
		}
		resArr = append(resArr, res)
	}

	sort.Strings(resArr)

	var resStr string = strings.Join(resArr, "_")

	outputChan <- resStr
}

// ExecutePipeline ...
func ExecutePipeline(jobs ...job) {
	inputCh := make(chan interface{}, 100)
	outputCh := make(chan interface{}, 100)
	wg := &sync.WaitGroup{}
	for _, job := range jobs {
		wg.Add(1)
		go worker(wg, job, inputCh, outputCh)
		inputCh = outputCh
		outputCh = make(chan interface{}, 100)
	}

	wg.Wait()

}

func worker(waitgroup *sync.WaitGroup, j job, inputCh, outputCh chan interface{}) {
	defer waitgroup.Done()
	defer close(outputCh)
	j(inputCh, outputCh)
}

func main() {
	var testResult string = "aa"
	inputData := []int{0, 1, 1, 2, 3, 5, 8}
	jobsArray := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Errorf("can't convert result data to string")
			}
			testResult = data
		}),
	}
	start := time.Now()
	ExecutePipeline(jobsArray...)
	end := time.Since(start)

	fmt.Println(testResult)
	fmt.Println("pipeline took", end)
}
