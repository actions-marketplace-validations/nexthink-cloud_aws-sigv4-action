package main

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/stretchr/testify/assert"
)

var testCredentials = aws.Credentials{AccessKeyID: "AKID", SecretAccessKey: "SECRET", SessionToken: "SESSION"}

func TestSignRequest(t *testing.T) {
	req, body := buildRequest("https://some-id.lambda-url.eu-west-1.on.aws/", "POST", "eu-west-1", "{}")
	signer := v4.NewSigner()
	err := signer.SignHTTP(context.Background(), testCredentials, req, body, "lambda", "eu-west-1", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
	expectedDate := "19700101T000000Z"
	expectedSig := "AWS4-HMAC-SHA256 Credential=AKID/19700101/eu-west-1/lambda/aws4_request, SignedHeaders=content-length;content-type;host;x-amz-date;x-amz-security-token, Signature=a4a65a93c8c2746e2a594ecbe77c0adfcc934735a0c46260fb265c4a8d9ab811"

	q := req.Header
	if e, a := expectedSig, q.Get("Authorization"); e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
	if e, a := expectedDate, q.Get("X-Amz-Date"); e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func BenchmarkSignRequest(b *testing.B) {
	signer := v4.NewSigner()
	req, bodyHash := buildRequest("https://some-id.lambda-url.eu-west-1.on.aws/", "POST", "eu-west-1", "{}")
	for i := 0; i < b.N; i++ {
		signer.SignHTTP(context.Background(), testCredentials, req, bodyHash, "lambda", "eu-west-1", time.Now())
	}
}

func TestGuessAWSRegion(t *testing.T) {
	tests := []struct {
		url            string
		expectedRegion string
	}{
		{"https://some-id.lambda-url.eu-west-1.on.aws/", "eu-west-1"},
		{"https://dejkfjklwejflewfkl.lambda-url.eu-west-3.on.aws/test", "eu-west-3"},
		{"https://dejkfjklwejflewfkl.lambda-url.us-east-1.on.aws/", "us-east-1"},
		{"https://dejkfjklwejflewfkl.lambda-url.eu-central-1.on.aws/", "eu-central-1"},
		{"https://dejkfjklwejflewfkl.lambda-url.eu-south-1.on.aws/", "eu-south-1"},
	}

	for _, test := range tests {
		region, err := guessAWSRegion(test.url)
		assert.Nil(t, err, "should not be any error")
		assert.Equal(t, test.expectedRegion, region, "unexpected region")
	}
}

func TestMalformedLambdaURL(t *testing.T) {
	malformedURL := "https://some-id.lambda-url.eu-us-2.on.aws/"
	region, err := guessAWSRegion(malformedURL)
	assert.Empty(t, region)
	assert.EqualError(t, err, "lambda function URL is malformed, impossible to guess AWS region")
}
