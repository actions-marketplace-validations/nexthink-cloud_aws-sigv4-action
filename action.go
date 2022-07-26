package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	EnvAWSAccessKeyID     = "aws-access-key-id"
	EnvAWSSecretAccessKey = "aws-secret-access-key"
	EnvAWSSessionToken    = "aws-session-token"
	EnvAWSRegion          = "region"
	EnvLambdaFunctionURL  = "lambda-function-url"
	EnvLambdaFunctionBody = "body"
)

const awsRegionRegExp = `(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d`

func main() {
	var credentials aws.Credentials
	lambdaURL := os.Getenv(EnvLambdaFunctionURL)
	if lambdaURL == "" {
		fmt.Fprintf(os.Stderr, "%s env variable is required\n", EnvLambdaFunctionURL)
		os.Exit(1)
	}

	body := os.Getenv(EnvLambdaFunctionBody)

	awsRegion := os.Getenv(EnvAWSRegion)
	var err error
	if awsRegion == "" {
		fmt.Fprintln(os.Stdout, "AWS region is not specified, try to guess from lambda URL")
		// Try to extract region from function URL => https://<id>.lambda-url.<region>.on.aws/
		awsRegion, err = guessAWSRegion(lambdaURL)
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

	req, bodyHash := buildRequest(lambdaURL, awsRegion, body)

	signer := v4.NewSigner()
	signer.SignHTTP(context.Background(), credentials, req, bodyHash, "lambda", awsRegion, time.Now())

	client := &http.Client{Timeout: time.Duration(5) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HTTP error %s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to decode response body %s", err)
	}

	fmt.Printf("status code: %s, response: %s", resp.Status, string(respBody))
}

func buildRequest(lambdaURL, region, body string) (*http.Request, string) {
	reader := strings.NewReader(body)
	return buildRequestWithBodyReader(lambdaURL, region, reader)
}

func buildRequestWithBodyReader(lambdaURL, region string, body io.Reader) (*http.Request, string) {
	req, _ := http.NewRequest(http.MethodPost, lambdaURL, body)
	req.Header.Add("Content-Type", "application/json")

	h := sha256.New()
	_, _ = io.Copy(h, body)
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
