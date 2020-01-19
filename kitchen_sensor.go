package main

import (
	"fmt"
	//	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	client "github.com/influxdata/influxdb1-client"
	"log"
	"math"
	"net/url"
	"strconv"
	"time"
)

const (
	database = "kitchen"
)

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}
func New(text string) error {
	return &errorString{text}
}

type KitchenSensor struct {
	Temperature    float64
	CarbonMonoxide float64
	GasLevel       float64
	Humidity       float64
}

func getRaundedFloat(floatString string) float64 {
	// returns float rounded to 2 decimal points
	if tempInt, err := strconv.ParseFloat(floatString, 32); err == nil {

		return math.Floor(tempInt*100) / 100
	} else {
		return -99
	}

}

func scrape(KitchenSensor KitchenSensor) KitchenSensor {

	c := colly.NewCollector()

	// On every a element which has href attribute call callback
	c.OnHTML(".temperature", func(body *colly.HTMLElement) {
		KitchenSensor.Temperature = getRaundedFloat(body.Text)
	})
	c.OnHTML(".co_level", func(body *colly.HTMLElement) {
		KitchenSensor.CarbonMonoxide = getRaundedFloat(body.Text)
	})
	c.OnHTML(".gas_level", func(body *colly.HTMLElement) {
		KitchenSensor.GasLevel = getRaundedFloat(body.Text)
	})
	c.OnHTML(".humidity", func(body *colly.HTMLElement) {
		KitchenSensor.Humidity = getRaundedFloat(body.Text)
	})
	// Before making a Requestest print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping on https://hackerspaces.org
	c.Visit("http://192.168.64.110/")
	return KitchenSensor

}

func updateDB(c *client.Client, KitchenSensor KitchenSensor) {
	var tags = map[string]string{
		"sensor":   "DHT22",
		"platform": "esp32",
	}

	eventTime := time.Now().Add(time.Second * -20)

	fields := map[string]interface{}{
		"Temperature":    KitchenSensor.Temperature,
		"CarbonMonoxide": KitchenSensor.CarbonMonoxide,
		"GasLevel":       KitchenSensor.GasLevel,
		"Humidity":       KitchenSensor.Humidity,
	}

	bp := client.BatchPoints{
		Points: []client.Point{
			{
				Measurement: "mesurements",
				Tags:        tags,
				Time:        eventTime.Add(time.Second * 10),
				Fields:      fields,
			},
		},
		Database:        database,
		RetentionPolicy: "one_week",
	}

	r, err := c.Write(bp)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	if r != nil {
		log.Fatalf("unexpected response. expected %v, actual %v", nil, r)
	}
}

func influxDBClient() *client.Client {
	host, err := url.Parse("http://localhost:8086")
	if err != nil {
		log.Fatal(err)
	}

	conf := client.Config{
		URL: *host,
	}
	c, err := client.NewClient(conf)
	if err != nil {
		log.Fatalf("unexpected error.  expected %v, actual %v", nil, err)
	}
	return c

}

func main() {
	c := influxDBClient()
	// Instantiate default collector
	var readings = KitchenSensor{0, 0, 0, 0}
	readings = scrape(readings)
	fmt.Println("The values :", readings)

	updateDB(c, readings)

}
