package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/go-cfenv"
)

type SmokeTestProgram interface {
	init(*cfenv.App)
	run() []SmokeTestResult
	publish([]SmokeTestResult) error
}

type smokeTestProgram struct {
	tests []SmokeTest
}

type SmokeTest interface {
	run() SmokeTestResult
}

type SmokeTestResult struct {
	Key              string            `json:"key,omitempty"`
	Result           bool              `json:"result"`
	Name             string            `json:"name"`
	Error            string            `json:"error,omitempty"`
	ErrorDescription string            `json:"errorDescription,omitempty"`
	StatusCode       *int              `json:"statusCode,omitempty"`
	Results          []SmokeTestResult `json:"results,omitempty"`
}

func (s *smokeTestProgram) init(env *cfenv.App) {
	s.tests = append(s.tests,
		meTestNew(),
		mySQLTestNew(env),
		rabbitMqTestNew(env, "p-rabbitmq", "RabbitMQ Shared Cluster"),
		rabbitMqTestNew(env, "p.rabbitmq", "RabbitMQ On-Demand"),
		redisTestNew(env, "p-redis", "Redis Shared Cluster"),
		redisTestNew(env, "p.redis", "Redis On-Demand"),
		postgresTestNew(env, "postgres-db", "Postgres"),
	        smbTestNew(env, "shared-volume", "shared SMB Volume (netApp)"))
}

func (s *smokeTestProgram) run() []SmokeTestResult {

//	results := make([]SmokeTestResult, len(s.tests), len(s.tests))
	var results []SmokeTestResult

	for _, test := range s.tests {
		if test != nil {
			results = append(results, test.run())
		}
	}
	return results
}

func (s *smokeTestProgram) publish(results []SmokeTestResult) error {
	// Read dashboard data endpoint from environment.
	dashboardDataEndpoint := os.Getenv("DASHBOARD_DATA_ENDPOINT")
	if dashboardDataEndpoint == "" {
		return fmt.Errorf("DASHBOARD_DATA_ENDPOINT env variable not set. Cannot post data to dashboard.")
	}

	// Post data to dashboard.
	resultBytes, err := json.Marshal(results)
	if err != nil {
		return err
	}
	postResponse, err := http.Post(dashboardDataEndpoint, "application/json", bytes.NewReader(resultBytes))
	if err != nil {
		return err
	}
	defer postResponse.Body.Close()

	// Check response (we expect a 204).
	if statusCode := postResponse.StatusCode; statusCode != http.StatusNoContent {
		return errors.New(fmt.Sprintf("Received unexpected status code %d from dashboard", statusCode))
	}
	return nil
}

type TestPart func() (interface{}, error)

func RunTestPart(testPart TestPart, testName string, results *[]SmokeTestResult) (interface{}, bool) {
	obj, err := testPart()
	if err != nil {
		fmt.Println(err.Error())
		*results = append(*results, SmokeTestResult{Name: testName, Result: false, Error: err.Error()})
		return nil, false
	}
	*results = append(*results, SmokeTestResult{Name: testName, Result: true})
	return obj, true
}

func OverallResult(key, name string, results []SmokeTestResult) SmokeTestResult {
	overallResult := true
	for _, res := range results {
		overallResult = overallResult && res.Result
	}
	return SmokeTestResult{Key: key, Name: name, Result: overallResult, Results: results}
}
