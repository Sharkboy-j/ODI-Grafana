package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

var (
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
)

func main() {

	influxDBURL := os.Getenv("INFLUX_DBURL")
	token := os.Getenv("INFLUX_TOKEN")
	org := os.Getenv("INFLUX_ORG")
	bucket := os.Getenv("INFLUX_BUCKET")

	client = influxdb2.NewClient(influxDBURL, token)
	writeAPI = client.WriteAPIBlocking(org, bucket)
	fmt.Println("Influx client started")

	done := make(chan bool, 1)

	go start()

	<-done
}

func start() {
	for {
		login()

		getStat()
	}
}

func login() {
	url := "http://192.168.1.1/boaform/admin/formLogin"
	method := "POST"

	payload := strings.NewReader("challenge=&username=admin&password=admin&save=Login&submit-url=%2Fadmin%2Flogin.asp")

	rClient := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := rClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
}

func getStat() {

	url := "http://192.168.1.1/status_pon.asp"
	method := "GET"

	rsClient := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:136.0) Gecko/20100101 Firefox/136.0")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Language", "ru-RU,ru;q=0.8,en-US;q=0.5,en;q=0.3")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("DNT", "1")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Referer", "http://192.168.1.1/left.html")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("Priority", "u=4")
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Cache-Control", "no-cache")

	for {
		res, err := rsClient.Do(req)
		if err != nil {
			fmt.Println(err)

			return
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		go parseHTML(string(body))

		time.Sleep(time.Second * time.Duration(1))
	}
}

func parseHTML(htmlContent string) {
	tm := time.Now()
	toExport := make(map[string]any)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	extractValue := func(paramName string) string {
		value := ""
		doc.Find("td").Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), paramName) {
				value = s.Next().Text()
			}
		})

		return strings.TrimSpace(value)
	}

	temp := extractNumber(extractValue("Temperature"))
	if temp != nil {
		toExport["temp_sfp"] = *temp
	}

	v := extractNumber(extractValue("Voltage"))
	if v != nil {
		toExport["volt"] = *v
	}

	txp := extractNumber(extractValue("Tx Power"))
	if v != nil {
		toExport["txp"] = *txp
	}

	rxp := extractNumber(extractValue("Rx Power"))
	if v != nil {
		toExport["rxp"] = *rxp
	}

	biasCurrent := extractNumber(extractValue("Bias Current"))
	if v != nil {
		toExport["bias_current"] = *biasCurrent
	}

	PushData(toExport, tm)

	//fmt.Println("Temperature:", extractValue("Temperature"))
	//fmt.Println("Voltage:", extractValue("Voltage"))
	//fmt.Println("Tx Power:", extractValue("Tx Power"))
	//fmt.Println("Rx Power:", extractValue("Rx Power"))
	//fmt.Println("Bias Current:", extractValue("Bias Current"))
	//fmt.Println("ONU State:", extractValue("ONU State"))
	//fmt.Println("Registered Status:", extractValue("Registered Status"))
}

func extractNumber(input string) *float64 {
	re := regexp.MustCompile(`-?\d+(\.\d+)?`)
	match := re.FindString(input)

	number, err := strconv.ParseFloat(match, 64)
	if err != nil {
		fmt.Println("Error parsing number:", err)
		return nil
	}

	return &number
}

func PushData(data map[string]any, tm time.Time) {
	p := influxdb2.NewPointWithMeasurement("sfp_data")
	p.SetTime(tm)

	for key, value := range data {
		p.AddField(key, value)
	}

	if len(p.FieldList()) == 0 {
		return
	}

	if err := writeAPI.WritePoint(context.Background(), p); err != nil {
		log.Printf("Error writing point to InfluxDB: %v", err)
	}

}
