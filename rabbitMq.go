package main

import (
        "crypto/tls"
        "fmt"
        "time"

        "github.com/cloudfoundry-community/go-cfenv"
        "github.com/streadway/amqp"
)

const (
        rabbitMqKey  = "rabbitmq"
        rabbitMqName = "RabbitMQ"

        rabbitMqTestCreatePublishingChannel = "Create publishing channel"
        rabbitMqTestDeclareQueue            = "Declare queue"
        rabbitMqTestPublishMessage          = "Publish message"
        rabbitMqTestCreateListeningChannel  = "Create listening channel"
        rabbitMqTestConsumeMessage          = "Consume message"
        rabbitMqTestCheckMessage            = "Check message"
)

type rabbitMqTest struct {
        connection *amqp.Connection
        qname      string
}

func rabbitMqTestNew(env *cfenv.App) SmokeTest {
        // TODO: replace with searching on tag basis, possibly resulting in multiple returns in case of multiple matches.
        rabbitMqServices, err := env.Services.WithLabel("p-rabbitmq")
        if err != nil {
                fmt.Println("RabbitMQ service not bound to smoketest app.")
                return &rabbitMqTest{}
        }

        uri := rabbitMqServices[0].Credentials["uri"].(string)

        amqpConnection, err := amqp.DialTLS(uri, &tls.Config{InsecureSkipVerify: true})
        if err != nil {
                fmt.Println("Error connecting to rabbitMQ: " + err.Error())
                return nil
        }

        return &rabbitMqTest{
                connection: amqpConnection,
                qname:      "smoketestsQueue",
        }
}

func (r *rabbitMqTest) listen(received chan SmokeTestResult, message string) {
        ch, err := r.connection.Channel()
        if err != nil {
                fmt.Println("error opening listener channel: " + err.Error())
                received <- SmokeTestResult{Name: rabbitMqTestCreateListeningChannel, Result: false, Error: err.Error()}
                close(received)
                return
        }
        defer ch.Close()
        received <- SmokeTestResult{Name: rabbitMqTestCreateListeningChannel, Result: true}

        msgs, err := ch.Consume(r.qname, "", true, false, false, false, nil)
        if err != nil {
                fmt.Println("error consuming messages: " + err.Error())
                received <- SmokeTestResult{Name: rabbitMqTestConsumeMessage, Result: false, Error: err.Error()}
                close(received)
                return
        }
        received <- SmokeTestResult{Name: rabbitMqTestConsumeMessage, Result: true}

        fmt.Println("Listener started...")
        for msg := range msgs {
                fmt.Printf("message: %s\n", msg.Body)
                if fmt.Sprintf("%s", msg.Body) == message {
                        received <- SmokeTestResult{Name: rabbitMqTestCheckMessage, Result: true}
                } else {
                        received <- SmokeTestResult{Name: rabbitMqTestCheckMessage, Result: false, Error: "Received message was different from sent message"}
                }
                close(received)
                return
        }
}

func (r *rabbitMqTest) run() SmokeTestResult {
        fmt.Println("Running rabbitmq tests")

        defer func() {
                if r := recover(); r != nil {
                        fmt.Printf("RabbitMQ test failed. Recovered from error:\n %v\n", r)
                }
        }()

        results := make([]SmokeTestResult, 0)

        // Create publishing channel.
        createPublishingChannel := func() (interface{}, error) {
                return r.connection.Channel()
        }
        obj, success := RunTestPart(createPublishingChannel, rabbitMqTestCreatePublishingChannel, &results)
        if !success {
                return OverallResult(rabbitMqKey, rabbitMqName, results)
        }
        channel := obj.(*amqp.Channel)
        defer channel.Close()

        // Declare queue.
        declareQueue := func() (interface{}, error) {
                return channel.QueueDeclare(r.qname, false, true, true, false, nil)
        }
        obj, success = RunTestPart(declareQueue, rabbitMqTestDeclareQueue, &results)
        if !success {
                return OverallResult(rabbitMqKey, rabbitMqName, results)
        }
        queue := obj.(amqp.Queue)

        // Create message body to send and start listening.
        message := fmt.Sprintf("%v", time.Now().Unix())
        fmt.Println("starting listener")
        listeningResults := make(chan SmokeTestResult)
        go r.listen(listeningResults, message)

        // Publish message.
        msg := amqp.Publishing{ContentType: "text/plain", Body: []byte(message)}
        err := channel.Publish("", queue.Name, false, false, msg)
        if err != nil {
                results = append(results, SmokeTestResult{Name: rabbitMqTestPublishMessage, Result: false, Error: err.Error()})
                return OverallResult(rabbitMqKey, rabbitMqName, results)
        }
        results = append(results, SmokeTestResult{Name: rabbitMqTestPublishMessage, Result: true})

        for listeningResult := range listeningResults {
                results = append(results, listeningResult)
        }

        return OverallResult(rabbitMqKey, rabbitMqName, results)
}
