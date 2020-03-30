package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
)

type PlayerDetails struct {
	PlayerName string `json:"playerName"`
}

func HandleRequest(ctx context.Context, credentials PlayerDetails) (string, error) {
	return "Hello", nil
}

func main() {
	lambda.Start(HandleRequest)
}

