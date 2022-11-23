package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
)

var program SmokeTestProgram

func handlerStatus(w http.ResponseWriter, r *http.Request) {

	// Run all tests.
	results := program.run()

	// Write output to response.
	body, err := json.Marshal(results)
	if err != nil {
		body = []byte("{ \"me\": \"false\" }")
	}
	fmt.Fprintf(w, string(body))

	// Attempt to write output to queue for dashboard.
	err = program.publish(results)
	if err != nil {
		log.Printf("Unable to publish results to dashboard. Error: %v", err)
	}
}

func main() {
	appEnv, err := cfenv.Current()
	if err != nil {
		panic(err)
	}

	config, err := smokeTestsConfigLoad()
	if err != nil {
		panic(err)
	}

	program = &smokeTestProgram{}
	program.init(appEnv, config)

	http.HandleFunc("/v1/status", handlerStatus)
	http.ListenAndServe(fmt.Sprintf(":%v", appEnv.Port), nil)
}
