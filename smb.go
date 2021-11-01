package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"os"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
)

type smbTest struct {
	path string
	key  string
	name string
}

func smbTestNew(s cfenv.Service, key string, name string) SmokeTest {
	mount := s.VolumeMounts[0]

	return &smbTest{
		path: mount["container_dir"],
		key: key,
		name: name,
	}
}

func (n *smbTest) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)

	if n.path == "" {
		results = append(results, SmokeTestResult{Name: "Load SMB Config", Result: false, Error: "SMB not configured"})
		return OverallResult(n.key, n.name, results)
	}

	write := func() (interface{}, error) {
	    filename := os.Getenv("SMB_FILE")
	    if (filename == "") {
	        return nil, fmt.Errorf("SMB_FILE env var not set")
	    }
	    
	    filePath := path.Join(n.path,os.Getenv("SMB_FILE"))
		data := []byte("test")
		err := ioutil.WriteFile(filePath, data, 0644)
		if err != nil {
			return nil, err
		}

		return true, nil
	}

	RunTestPart(write, "Write", &results)
	return OverallResult(n.key, n.name, results)
}

