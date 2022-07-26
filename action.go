package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	EnvAWSAccessKeyID     = "AWS_ACCESS_KEY_ID"
	EnvAWSSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	EnvAWSSessionToken    = "AWS_SESSION_TOKEN"
	EnvAWSRegion          = "AWS_REGION"
)

const awsRegionRegExp = `(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d`

var (
	lambdaURL     = flag.String("lambda-url", "", "The lambda function URL, should be https://<id>.lambda-url.<region>.on.aws/something.")
	requestBody   = flag.String("body", "", "The body associated with the request (POST request).")
	requestMethod = flag.String("method", "GET", "HTTP Method used to call the Lambda function.")
	headerList    = flag.String("headers", "", "List of Headers")
)

func main() {
	flag.Parse()

	var credentials aws.Credentials

	if *lambdaURL == "" {
		fmt.Fprintln(os.Stderr, "lambda-url is required")
		os.Exit(1)
	}

	awsRegion := os.Getenv(EnvAWSRegion)
	var err error
	if awsRegion == "" {
		fmt.Fprintln(os.Stdout, "AWS region is not specified, try to guess from lambda URL")
		// Try to extract region from function URL => https://<id>.lambda-url.<region>.on.aws/
		awsRegion, err = guessAWSRegion(*lambdaURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	awsAccessKeyID := os.Getenv(EnvAWSAccessKeyID)
	if awsAccessKeyID == "" {
		fmt.Fprintf(os.Stderr, "%s env variable is required\n", EnvAWSAccessKeyID)
		os.Exit(1)
	}

	awsSecretAccessKey := os.Getenv(EnvAWSSecretAccessKey)
	if awsSecretAccessKey == "" {
		fmt.Fprintf(os.Stderr, "%s env variable is required\n", EnvAWSSecretAccessKey)
		os.Exit(1)
	}

	awsSessionToken := os.Getenv(EnvAWSSessionToken)
	if awsSessionToken == "" {
		credentials = aws.Credentials{AccessKeyID: awsAccessKeyID, SecretAccessKey: awsSecretAccessKey}
	} else {
		credentials = aws.Credentials{AccessKeyID: awsAccessKeyID, SecretAccessKey: awsSecretAccessKey, SessionToken: awsSessionToken}
	}

	req, bodyHash := buildRequest(*lambdaURL, *requestMethod, awsRegion, *requestBody)
	req.Body = ioutil.NopCloser(strings.NewReader(*requestBody))

	signer := v4.NewSigner()
	signer.SignHTTP(context.Background(), credentials, req, bodyHash, "lambda", awsRegion, time.Now())

	client := &http.Client{Timeout: time.Duration(5) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HTTP error %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to decode response body %s\n", err)
	}

	fmt.Printf("status code: %s, response: %s", resp.Status, string(respBody))

	// Github Action outputs
	fmt.Printf(`::set-output name=status::%s`, resp.Status)
	fmt.Print("\n")
	fmt.Printf(`::set-output name=code::%d`, resp.StatusCode)
	fmt.Print("\n")
	fmt.Printf(`::set-output name=message::%s`, string(respBody))
	fmt.Print("\n")
}

func buildRequest(lambdaURL, requestMethod, region, requestBody string) (*http.Request, string) {
	reader := strings.NewReader(requestBody)
	return buildRequestWithBodyReader(lambdaURL, requestMethod, region, reader)
}

func buildRequestWithBodyReader(lambdaURL, requestMethod, region string, requestBody io.Reader) (*http.Request, string) {

	req, err := http.NewRequest(requestMethod, lambdaURL, requestBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building the http request %s\n", err)
		os.Exit(1)
	}
	headers := strings.Split(*headerList, "\n")
	for _, header := range headers {
		key := strings.Trim(strings.Split(header, ":")[0], " ")
		value := strings.Trim(strings.Split(header, ":")[1], " ")
		req.Header.Add(key, value)
	}

	// req.Header.Add("Content-Type", "application/json")
	// req.Header.Add("Accept", "*")

	h := sha256.New()
	_, _ = io.Copy(h, requestBody)
	payloadHash := hex.EncodeToString(h.Sum(nil))

	return req, payloadHash
}

func guessAWSRegion(lambdaURL string) (string, error) {
	u, _ := url.Parse(lambdaURL)
	r := regexp.MustCompile(awsRegionRegExp)

	result := r.FindStringSubmatch(u.Hostname())
	if result == nil {
		return "", errors.New("lambda function URL is malformed, impossible to guess AWS region")
	}
	return result[0], nil
}
