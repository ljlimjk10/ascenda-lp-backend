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
	"github.com/vynious/ascenda-lp-backend/types"
	"github.com/vynious/ascenda-lp-backend/util"
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

func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("i am entering the function")
	// Checking if userid and userlocation exists for logging purposes
	userId, err := util.GetCustomAttributeWithCognito("custom:userID", request.Headers["Authorization"])
	if err == nil {
		ctx = context.WithValue(ctx, "userId", userId)
	}
	userLocation, ok := request.Headers["CloudFront-Viewer-Country"]
	if ok {
		ctx = context.WithValue(ctx, "userLocation", userLocation)
	}
	bank, err := util.GetCustomAttributeWithCognito("custom:bank", request.Headers["Authorization"])
	if err == nil {
		ctx = context.WithValue(ctx, "bank", bank)
	}
	DB := DBService.GetBanksDB(request.Headers["Authorization"])

	roleName, exists := request.QueryStringParameters["roleName"]
	if !exists || roleName == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET",
			},
			Body: "Missing or empty RoleName query parameter",
		}, nil
	}

	roleRequestBody := types.GetRoleRequestBody{RoleName: roleName}
	role, err := db.RetrieveRoleWithRetrieveRoleRequestBody(ctx, DB, roleRequestBody)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return events.APIGatewayProxyResponse{
				StatusCode: 404,
				Headers: map[string]string{
					"Access-Control-Allow-Headers": "Content-Type",
					"Access-Control-Allow-Origin":  "*",
					"Access-Control-Allow-Methods": "GET",
				},
				Body: "Role not found",
			}, nil
		} else {
			log.Printf("Database error: %s", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Headers: map[string]string{
					"Access-Control-Allow-Headers": "Content-Type",
					"Access-Control-Allow-Origin":  "*",
					"Access-Control-Allow-Methods": "GET",
				},
				Body: "Internal server error",
			}, nil
		}
	}
	log.Printf("i have made it out of hell 1")
	responseBody, err := json.Marshal(role)
	if err != nil {
		log.Printf("JSON marshal error: %s", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET",
			},
			Body: "Error marshaling role into JSON",
		}, nil
	}
	log.Printf("i have made it out of hell 2")
	return events.APIGatewayProxyResponse{
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
	defer DBService.CloseConnections()
}
