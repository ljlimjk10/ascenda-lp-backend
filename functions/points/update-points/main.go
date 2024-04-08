package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vynious/ascenda-lp-backend/db"
	"github.com/vynious/ascenda-lp-backend/types"
)

var (
	DB      *db.DBService
	err     error
	headers = map[string]string{
		"Access-Control-Allow-Headers": "Content-Type",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "PUT",
	}
)

func init() {
	log.Printf("INIT")
	DB, err = db.SpawnDBService()
}

func main() {
	// we are simulating a lambda behind an ApiGatewayV2
	lambda.Start(handler)

	defer DB.CloseConn()
}

func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	// Checking if userid and userlocation exists for logging purposes
	// userId, ok := request.Headers["userId"]
	// if ok {
	// 	ctx = context.WithValue(ctx, "userId", userId)
	// }
	userLocation, ok := request.Headers["CloudFront-Viewer-Country"]
	if ok {
		ctx = context.WithValue(ctx, "userLocation", userLocation)
	}
	req := types.UpdatePointsRequestBody{}
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    headers,
			Body:       errors.New("invalid request. malformed request found").Error(),
		}, nil
	}

	if req.ID == "" || req.NewBalance == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    headers,
			Body:       errors.New("bad request. id or new_balance not found").Error(),
		}, nil
	}
	log.Printf("UpdatePoints %s", req.ID)

	pointsRecord, err := DB.UpdatePoints(ctx, req)
	if pointsRecord == nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    headers,
			Body:       err.Error(),
		}, nil
	}

	obj, _ := json.Marshal(pointsRecord)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(obj),
	}, nil
}
