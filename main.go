package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/saskcan/hoagie/yahooFinance"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

// maximum frequency for requests to avoid overusage errors
const maxRequestFrequency time.Duration = time.Duration(1) * time.Second

// CandlesSinceDateRequest represents a request to retrieve candles since the date provided
type CandlesSinceDateRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Frequency string    `json:"frequency"`
	Date      string    `json:"date"`
}

// Job represents a request to do work
type Job struct {
	Type string                  `json:"type"`
	Data CandlesSinceDateRequest `json:"data"`
}

func main() {
	conn, err := amqp.Dial("amqp://localhost:5672/")
	if err != nil {
		fmt.Printf("Could not connect to rabbitMQ\n")
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		fmt.Printf("Could not connect to channel\n")
		return
	}
	defer ch.Close()

	// jobs queue
	jobsQueue, err := ch.QueueDeclare(
		"jobs", // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		fmt.Printf("Could not declare queue\n")
		return
	}

	msgs, err := ch.Consume(
		jobsQueue.Name, // queue
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		fmt.Printf("Failed to register a consumer\n")
		return
	}

	// data queue
	dataQueue, err := ch.QueueDeclare(
		"data", // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		fmt.Printf("Failed to register a consumer")
		return
	}

	var lastJobTime = time.Now()
	//var queuedJobs []*common.Job

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var job Job
			//fmt.Printf("%v", string(d.Body))
			//fmt.Printf("%s\n", string(d.Body))
			err := json.Unmarshal(d.Body, &job)
			if err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				continue
			}

			if job.Type != "CandlesSinceDateRequest" {
				fmt.Printf("Was expecting CandlesSinceDateRequest but got %s\n", job.Type)
				continue
			}

			log.Printf("Received a job: %v", job.Data)

			// do not exceed maxRequestFrequency
			earliestStartTime := lastJobTime.Add(maxRequestFrequency)
			time.Sleep(time.Until(earliestStartTime))
			lastJobTime = time.Now()

			// stubbed
			s, err := time.Parse("2006-01-02T15:04:05Z", job.Data.Date)
			if err != nil {
				fmt.Printf("Could not parse date: %v", err)
			}

			candles, err := yahooFinance.RetrieveData(job.Data.Symbol, job.Data.Frequency, s, time.Now())
			if err != nil {
				fmt.Printf("Could not retrieve data from Yahoo Finance: %v", err)
				continue
			}

			// fmt.Print(candles)

			//fmt.Printf("Symbol: %s Frequency: %s Start: %v End: %v\n", job.Data.Symbol, job.Data.Frequency, job.Data.Date, time.Now())

			for _, candle := range candles {
				// set key for reference by consumer
				candle.ProductID = job.Data.ProductID
				// fmt.Printf("ID is %v\n", candle.ProductID)

				dataBytes, err := json.Marshal(&candle)
				if err != nil {
					fmt.Printf("Could not marshal json: %v", candle)
					break
				}

				err = ch.Publish(
					"",             // exchange
					dataQueue.Name, // routing key
					false,          // mandatory
					false,          // immediate
					amqp.Publishing{
						ContentType: "application/json",
						Body:        dataBytes,
					})
				if err != nil {
					fmt.Printf("Could not send to queue")
					break
				}
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	<-forever
}
