package main

import (
	"os"
)

type me struct {
}

func meTestNew() SmokeTest {
	return &me{}
}

func (m *me) run() SmokeTestResult {
	name := "Me"
	sitetype := os.Getenv("TYPE")
	sitename := os.Getenv("SITE")
	if (sitetype != "" && sitename != "") {
		name = sitetype + "\n" + "[" + sitename + "]"
	}
	return SmokeTestResult{Key: "me", Name: name, Result: true}
}
