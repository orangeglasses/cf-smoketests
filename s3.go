package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
)

type CredBucket struct {
	URI        string `json:"uri"`
	Name       string `json:"name"`
	Bucket     string `json:"bucket"`
	Region     string `json:"region"`
	Versioning bool   `json:"versioning"`
}

type Credentials struct {
	InsecureSkipVerify bool         `json:"insecure_skip_verify"`
	AccessKeyID        string       `json:"access_key_id"`
	SecretAccessKey    string       `json:"secret_access_key"`
	Buckets            []CredBucket `json:"buckets"`
	Endpoint           string       `json:"endpoint"`
	PathStyleAccess    bool         `json:"pathStyleAccess"`
}

type s3Test struct {
	Client *s3.S3
	Bucket string
}

func s3TestNew(env *cfenv.App) SmokeTest {
	s3Services, err := env.Services.WithTag("s3-bucket")
	if err != nil {
		fmt.Println("smoketest app not bound to an s3 service")
		return nil
	}

	creds := s3Services[0].Credentials
	bucketMap := creds["buckets"].([]map[string]interface{})[0]

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	httpClient := http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}

	sess, err := session.NewSession(&aws.Config{
		HTTPClient:       &httpClient,
		Credentials:      credentials.NewStaticCredentials(creds["access_key_id"].(string), creds["secret_access_key"].(string), ""),
		Endpoint:         aws.String(creds["endpoint"].(string)),
		Region:           aws.String(bucketMap["region"].(string)),
		S3ForcePathStyle: aws.Bool(creds["pathStyleAccess"].(bool)),
	},
	)

	return &s3Test{
		Client: s3.New(sess),
		Bucket: bucketMap["bucket"].(string),
	}
}

func (t *s3Test) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)

	filename := path.Join("./", "s3testfile")

	//create test file
	write := func() (interface{}, error) {
		data := []byte("test")
		err := ioutil.WriteFile(filename, data, 0644)
		if err != nil {
			return nil, err
		}

		return true, nil
	}

	upload := func() (interface{}, error) {
		upFile, err := os.Open(filename)
		if err != nil {
			return false, err
		}
		defer upFile.Close()

		upFileInfo, _ := upFile.Stat()
		var fileSize int64 = upFileInfo.Size()
		fileBuffer := make([]byte, fileSize)
		upFile.Read(fileBuffer)
		_, err = t.Client.PutObject(&s3.PutObjectInput{
			Bucket:             aws.String(t.Bucket),
			Key:                aws.String("s3testfile"),
			ACL:                aws.String("private"),
			Body:               bytes.NewReader(fileBuffer),
			ContentDisposition: aws.String("attachment"),
			ContentLength:      aws.Int64(int64(len(fileBuffer))),
			ContentType:        aws.String(http.DetectContentType(fileBuffer)),
		})
		return true, nil
	}

	RunTestPart(write, "Create local testfile", &results)
	RunTestPart(upload, "Upload file to S3", &results)
	return OverallResult("nfs", "NFS", results)
}
