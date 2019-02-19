package main

type me struct {
}

func meTestNew() SmokeTest {
	return &me{}
}

func (m *me) run() SmokeTestResult {
	return SmokeTestResult{Key: "me", Name: "Me", Result: true}
}
