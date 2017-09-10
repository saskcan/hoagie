package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/saskcan/finance/common"
	"github.com/saskcan/finance/hoagie/yahooFinance"
	"github.com/streadway/amqp"
)

// maximum frequency for requests to avoid overusage errors
const maxRequestFrequency time.Duration = time.Duration(1) * time.Second

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)

		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")

	failOnError(err, "Failed to connect to RabbitMQ")

	defer conn.Close()

	ch, err := conn.Channel()

	failOnError(err, "Failed to open a channel")

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
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		jobsQueue.Name, // queue
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	failOnError(err, "Failed to register a consumer")

	// data queue
	dataQueue, err := ch.QueueDeclare(
		"data", // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	failOnError(err, "Failed to declare a queue")

	var lastJobTime time.Time = time.Now()
	//var queuedJobs []*common.Job

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			job := new(common.Job)
			//fmt.Printf("%v", string(d.Body))
			err := json.Unmarshal(d.Body, job)
			failOnError(err, "Failed to unmarshal the message")
			log.Printf("Received a job: %v", job)

			// do not exceed maxRequestFrequency
			earliestStartTime := lastJobTime.Add(maxRequestFrequency)
			time.Sleep(time.Until(earliestStartTime))
			lastJobTime = time.Now()

			candles, err := yahooFinance.RetrieveData(job.Symbol, job.Frequency, job.Range)
			failOnError(err, "Failed to retrieve data from Yahoo Finance")

			for _, candle := range candles {

				dataBytes, err := json.Marshal(&candle)
				failOnError(err, "Could not marshal candle to json")

				err = ch.Publish(
					"",             // exchange
					dataQueue.Name, // routing key
					false,          // mandatory
					false,          // immediate
					amqp.Publishing{
						ContentType: "application/json",
						Body:        dataBytes,
					})
				failOnError(err, "Could not publish to channel")
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	<-forever
}
