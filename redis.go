package main

import (
	"fmt"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/go-redis/redis"
)

const (
	redisKey  = "redis"
	redisName = "Redis"
)

type redisTest struct {
	client *redis.Client
}

func redisTestNew(env *cfenv.App) SmokeTest {
	redisServices, err := env.Services.WithLabel("p-redis")
	if err != nil {
		fmt.Println(err.Error())
		return &redisTest{}
	}

	creds := redisServices[0].Credentials

	return &redisTest{
		client: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%v:%v", creds["host"].(string), creds["port"].(float64)),
			Password: creds["password"].(string),
			DB:       0,
		}),
	}
}

func (r *redisTest) run() SmokeTestResult {

	results := make([]SmokeTestResult, 0)

	ping := func() (interface{}, error) {
		return r.client.Ping().Result()
	}
	obj, success := RunTestPart(ping, "Ping", &results)
	if !success {
		return OverallResult(redisKey, redisName, results)
	}
	pong := obj.(string)

	if pong != "PONG" {
		results = append(results, SmokeTestResult{Name: "Pong", Result: false, Error: "No PONG reply from Redis"})
	} else {
		results = append(results, SmokeTestResult{Name: "Pong", Result: true})
	}

	return OverallResult(redisKey, redisName, results)
}
