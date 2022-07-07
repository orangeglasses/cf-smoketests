package main

import (
	"fmt"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/go-redis/redis"
)

const (
)

type redisTest struct {
	client *redis.Client
	redisKey string
	redisName string

}

func redisTestNew(env *cfenv.App, serviceName, friendlyName string) SmokeTest {
	// TODO: replace with searching on tag basis, possibly resulting in multiple returns in case of multiple matches.
	redisServices, err := env.Services.WithLabel(serviceName)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	creds := redisServices[0].Credentials

	return &redisTest{
		client: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%v:%v", creds["host"].(string), creds["port"].(float64)),
			Password: creds["password"].(string),
			DB:       0,
		}),
		redisKey: serviceName,
		redisName: friendlyName,
	}
}

func (r *redisTest) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)

	ping := func() (interface{}, error) {
		return r.client.Ping().Result()
	}
	obj, success := RunTestPart(ping, "Ping", &results)
	if !success {
		return OverallResult(r.redisKey, r.redisName, results)
	}
	pong := obj.(string)

	if pong != "PONG" {
		results = append(results, SmokeTestResult{Name: "Pong", Result: false, Error: "No PONG reply from Redis"})
	} else {
		results = append(results, SmokeTestResult{Name: "Pong", Result: true})
	}

	return OverallResult(r.redisKey, r.redisName, results)
}
