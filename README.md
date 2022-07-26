# AWS Signature Version 4 (SigV4) Github Action
[![Build](https://github.com/nexthink-cloud/aws-sigv4-action/actions/workflows/go.yml/badge.svg)](https://github.com/nexthink-cloud/aws-sigv4-action/actions/workflows/go.yml)

This Github Action allows you to call [Lambda Function URL](https://docs.aws.amazon.com/lambda/latest/dg/lambda-urls.html) with [AWS_IAM](https://docs.aws.amazon.com/lambda/latest/dg/urls-auth.html#urls-auth-iam) authentication mechanism by signing your request using the [AWS Signature v4 process](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html).

## Usage

To use this action, you need to have the necessary information to sign your request. When you send API requests to AWS, you sign the requests so that AWS can identify who sent them. You sign requests with your AWS access key, which consists of an access key ID, a secret access key and optionnally a session token. 

You can use this action after the [configure-aws-credentials](https://github.com/aws-actions/configure-aws-credentials) action. Here is an example:

1. Create a `.github/workflows/my-workflow.yml` file in your GitHub repo.
2. Add the following code to the `my-workflow.yml` file:

```yml
name: Test

# These permissions are needed to interact with GitHub's OIDC Token endpoint.
permissions:
  id-token: write
  contents: read
on:
  push:
jobs:
  test:
    name: Invoke Lambda Function URL with AWS_IAM auth.
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@2541b1294d2704b0964813337f33b291d3f8596b
        # Use of OIDC integration but can be anything documented here:
        # https://github.com/aws-actions/configure-aws-credentials
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@05b148adc31e091bafbaf404f745055d4d3bc9d2
        with:
          role-to-assume: ${{ secrets.my-role-to-assume }}
          aws-region: eu-west-1
        timeout-minutes: 1

      - name: Invoke Lambda function
        uses: nexthink-cloud/aws-sigv4-action@0.0.3
        with:
          lambda-url: https://y7imr2c6nw2cyyl7auyykesmuq0dbaag.lambda-url.eu-west-1.on.aws/event
          body: '{"Test": "result"}'
          method: POST
```