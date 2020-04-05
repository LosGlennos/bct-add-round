package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"errors"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat/go-jwx/jwk"
	"log"
	"os"
	"strconv"
)

type Handler struct {
	dynamoClient     *dynamodb.DynamoDB
	identityProvider *cognito.CognitoIdentityProvider
	WellKnownJWKs    *jwk.Set
}

type Event struct {
	Headers Headers `json:"headers"`
	Body    Body    `json:"body"`
}

type Headers struct {
	Authorization string `json:"authorization"`
}

type Body struct {
	PlayerName string `json:"playerName"`
	Round      string `json:"round"`
	Points     int    `json:"points"`
}

func HandleRequest(ctx context.Context, event Event) (string, error) {
	handler := newHandler()

	err := handler.authenticate(event.Headers)
	if err != nil {
		return "", err
	}

	err = handler.saveRound(event.Body)
	if err != nil {
		return "", err
	}

	return "Saved", nil
}

func newHandler() Handler {
	dynamoClient := getDynamoClient()
	cognitoClient := getCognitoClient()

	return Handler{
		dynamoClient:     dynamoClient,
		identityProvider: cognitoClient,
	}
}

func getCognitoClient() *cognito.CognitoIdentityProvider {
	conf := &aws.Config{
		Region: aws.String("eu-west-1"),
	}

	s, err := session.NewSession(conf)
	if err != nil {
		panic(err)
	}

	identityProvider := cognito.New(s)

	return identityProvider
}

func getDynamoClient() *dynamodb.DynamoDB {
	endpointCfg := aws.NewConfig().
		WithRegion("eu-west-1").
		WithCredentialsChainVerboseErrors(true)

	s := session.Must(session.NewSession())
	dynamoClient := dynamodb.New(s, endpointCfg)
	return dynamoClient
}

func (handler *Handler) saveRound(round Body) error {
	tableName := "BeerCartingTour"
	_, err := handler.dynamoClient.PutItem(&dynamodb.PutItemInput{
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
	return err
}

func (handler *Handler) authenticate(headers Headers) error{
	token := headers.Authorization[6:]
	_, err := handler.ParseAndVerifyJWT(token)
	if err != nil {
		return err
	}
	return nil
}

func (handler *Handler) ParseAndVerifyJWT(t string) (*jwt.Token, error) {
	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		keys := handler.WellKnownJWKs.LookupKeyID(token.Header["kid"].(string))
		if len(keys) == 0 {
			log.Println("Failed to look up JWKs")
			return nil, errors.New("could not find matching `kid` in well known tokens")
		}
		key, err := keys[0].Materialize()
		if err != nil {
			log.Printf("Failed to create public key: %s", err)
			return nil, err
		}
		rsaPublicKey := key.(*rsa.PublicKey)
		return rsaPublicKey, nil
	})

	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			err = claims.Valid()
			if err == nil {
				if claims.VerifyAudience(os.Getenv("COGNITO_APP_CLIENT_ID"), false) {
					return token, nil
				} else {
					err = errors.New("token audience does not match client id")
					log.Println("Invalid audience for id token")
				}
			} else {
				log.Println("Invalid claims for id token")
				log.Println(err)
			}
		}
	} else {
		log.Println("Invalid token:", err)
	}

	return nil, err
}

func (handler *Handler) getWellKnownJWTKs() error {
	var buffer bytes.Buffer
	buffer.WriteString("https://cognito-idp.eu-west-1.amazonaws.com/")
	buffer.WriteString(os.Getenv("COGNITO_USER_POOL_ID"))
	buffer.WriteString("/.well-known/jwks.json")
	wkjwksURL := buffer.String()
	buffer.Reset()

	set, err := jwk.Fetch(wkjwksURL)
	if err == nil {
		handler.WellKnownJWKs = set
	} else {
		log.Println("There was a problem getting the well known JSON web token key set")
		log.Println(err)
	}
	return err
}

func main() {
	lambda.Start(HandleRequest)
}
