package main

import (
	"fmt"
	"io/ioutil"
	"path"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
)

type nfsTest struct {
	path string
}

func nfsTestNew(env *cfenv.App) SmokeTest {
	// TODO: replace with searching on tag basis, possibly resulting in multiple returns in case of multiple matches.
	nfsServices, err := env.Services.WithTag("nfs")
	if err != nil {
		fmt.Println(err.Error())
		return &nfsTest{}
	}
	mount := nfsServices[0].Volume_Mounts[0]

	return &nfsTest{
		path: mount["container_dir"],
	}
}

func (n *nfsTest) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)

	if n.path == "" {
		results = append(results, SmokeTestResult{Name: "Load NFS Config", Result: false, Error: "NFS not configured"})
		return OverallResult("nfs", "NFS", results)
	}

	write := func() (interface{}, error) {
		filename := path.Join(n.path, "prodsmoketestfile")
		data := []byte("test")
		err := ioutil.WriteFile(filename, data, 0644)
		if err != nil {
			return nil, err
		}

		return true, nil
	}

	RunTestPart(write, "Write", &results)
	return OverallResult("nfs", "NFS", results)
}
