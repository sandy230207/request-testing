package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v4"
	"github.com/prometheus/common/log"
)

var (
	count        = 0
	countErr     = 0
	countSuccess = 0
)

func sendRequest(datas [][]byte, dataCount int, url string) error {
	rand.Seed(time.Now().UnixNano())
	count++
	data := datas[rand.Intn(dataCount)]
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		countErr++
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "postman")
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}
	client := &http.Client{Transport: transCfg}
	res, err := client.Do(req)
	if err != nil {
		countErr++
		return err
	}
	// fmt.Println("response Status:", res.Status)
	// fmt.Println("response Headers:", res.Header)
	// body, _ := ioutil.ReadAll(res.Body)
	// fmt.Println("response Body:", string(body))
	countSuccess++
	defer res.Body.Close()
	return nil
}

func job(datas *jobLoop) string {
	loopCount := datas.loopCount
	for i := 0; i < loopCount; i++ {
		err := sendRequest(datas.datas, datas.dataCount, datas.url)
		if err != nil {
			log.Errorln(err)
		}
	}
	return "End"
}

type request struct {
	Subject string   `json:"subject"`
	Prs     []string `json:"prs"`
	Urls    []string `json:"urls"`
	Froms   []string `json:"froms"`
	Sdrip   string   `json:"sdrip"`
}

func dataGenerator(dataCount int) [][]byte {
	time := time.Now().UnixNano()
	rand.Seed(time)
	gofakeit.Seed(time)
	var data [][]byte
	for i := 0; i < dataCount; i++ {
		froms := strings.Join(
			[]string{
				gofakeit.Name(),
				" <",
				gofakeit.Email(),
				">",
			}, "")
		req := &request{
			Subject: "WW91ciBhY2NvdW50IGhhcyBiZWVuIGJsb2NrZWQ=",
			Prs:     []string{"8869" + strconv.FormatInt(1000000000000000+rand.Int63n(99999999999999), 10)},
			Urls:    []string{base64.StdEncoding.EncodeToString([]byte(gofakeit.URL()))},
			Froms:   []string{base64.StdEncoding.EncodeToString([]byte(froms))},
			Sdrip:   gofakeit.IPv4Address(),
		}
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return nil
		}
		data = append(data, reqJSON)
	}
	return data
}

func getConfig() (int, int, int, string) {
	var dataCount, threadCount, loopCount int
	data, err := strconv.ParseInt(os.Getenv("DATA_COUNT"), 10, 32)
	if err != nil {
		data = 1000000
		log.Errorln("Invalid input: ", err)
	}
	dataCount = int(data)
	if dataCount < 1 {
		dataCount = 1
		log.Errorf("DATA_COUNT cannot less than 1. It would be set to %v.", dataCount)
	}
	thread, err := strconv.ParseInt(os.Getenv("THREAD_COUNT"), 10, 32)
	if err != nil {
		thread = 1000
		log.Errorln("Invalid input: ", err)
	}
	threadCount = int(thread)
	if threadCount < 1 {
		threadCount = 1
		log.Errorf("THREAD_COUNT cannot less than 1. It would be set to %v.", threadCount)
	}
	loop, err := strconv.ParseInt(os.Getenv("LOOP_COUNT"), 10, 32)
	if err != nil {
		loop = 1000
		log.Errorln("Invalid input: ", err)
	}
	loopCount = int(loop)
	if loopCount < 1 {
		loopCount = 1
		log.Errorf("LOOP_COUNT cannot less than 1. It would be set to %v.", loopCount)
	}
	url := os.Getenv("REQUEST_URL")
	if url == "" {
		log.Errorf("REQUEST_URL cannt be empty.")
		return
	}
	return dataCount, threadCount, loopCount, url
}

type jobLoop struct {
	datas     [][]byte
	loopCount int
	dataCount int
	url       string
}

func main() {
	dataCount, threadCount, loopCount, url := getConfig()
	fmt.Printf("dataCount: %v  threadCount: %v  loopCount: %v\n", dataCount, threadCount, loopCount)
	results := make(chan string)
	datas := &jobLoop{
		datas:     dataGenerator(dataCount),
		loopCount: loopCount,
		dataCount: dataCount,
		url:       url,
	}
	for i := 0; i < threadCount; i++ {
		go func(num int) {
			results <- job(datas)
		}(i)
	}
	for i := 0; i < threadCount; i++ {
		log.Infoln(<-results)
	}
	log.Infof("Total Requests: %v\nError Requests Count: %v\nSuccessful Requests Count:%v\n", count, countErr, countSuccess)
}
