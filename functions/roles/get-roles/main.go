package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/vynious/ascenda-lp-backend/db"
	"gorm.io/gorm"
)

var (
	DBService *db.DBService
	RDSClient *rds.Client
	err       error
)

func init() {
	DBService, err = db.SpawnDBService()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	roles, err := db.RetrieveAllRolesWithUsers(ctx, DBService)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Headers: map[string]string{
					"Access-Control-Allow-Headers": "Content-Type",
					"Access-Control-Allow-Origin":  "*",
					"Access-Control-Allow-Methods": "GET",
				},
				Body: "Role(s) not found",
			}, nil
		} else {
			log.Printf("Database error: %s", err)
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Access-Control-Allow-Headers": "Content-Type",
					"Access-Control-Allow-Origin":  "*",
					"Access-Control-Allow-Methods": "GET",
				},
				Body: `{"message": "Internal server error"}`,
			}, nil
		}
	}

	responseBody, err := json.Marshal(roles)
	if err != nil {
		log.Printf("JSON marshal error: %s", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET",
			},
			Body: `{"message": "Error marshaling roles into JSON"}`,
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Headers": "Content-Type",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	// we are simulating a lambda behind an ApiGatewayV2
	lambda.Start(handler)
	defer DBService.CloseConn()
}
