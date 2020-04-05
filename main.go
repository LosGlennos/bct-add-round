package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"strconv"
)

type Round struct {
	PlayerName string `json:"playerName"`
	Round string `json:"round"`
	Points int `json:"points"`
}

func HandleRequest(ctx context.Context, round Round) (string, error) {
	endpointCfg := aws.NewConfig().
		WithRegion("eu-west-1").
		WithCredentialsChainVerboseErrors(true)

	s := session.Must(session.NewSession())
	dynamoClient := dynamodb.New(s, endpointCfg)

	tableName := "BeerCartingTour"

	_, err := dynamoClient.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"PlayerName": {
				S: aws.String(round.PlayerName),
			},
			"Round": {
				S: aws.String(round.Round),
			},
			"Points": {
				N: aws.String(strconv.Itoa(round.Points)),
			},
		},
	})
	if err != nil {
		return "", err
	}

	return "Saved", nil
}

func main() {
	lambda.Start(HandleRequest)
}

