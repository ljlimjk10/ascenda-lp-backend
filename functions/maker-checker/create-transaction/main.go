package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vynious/ascenda-lp-backend/db"
	"github.com/vynious/ascenda-lp-backend/types"
	"github.com/vynious/ascenda-lp-backend/util"
)

var (
	DBService    *db.DBService
	requestBody  types.CreateTransactionBody
	responseBody types.TransactionResponseBody
	action       types.MakerAction
	err          error
)

func init() {

	// Initialise global variable DBService tied to Aurora
	DBService, err = db.SpawnDBService()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func CreateTransactionHandler(ctx context.Context, req *events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	defer DBService.CloseConn()

	role := ""

	if err := json.Unmarshal([]byte(req.Body), &requestBody); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 404,
			Body:       "Bad Request",
		}, nil
	}

	// Calls DB Service to create transaction
	txn, err := DBService.CreateTransaction(ctx, action, requestBody.MakerId, requestBody.Description)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "",
		}, nil
	}

	responseBody.Txn = *txn

	// Send emails seek checker's approval (Async)
	checkersEmail, err := DBService.GetCheckers(ctx, role)
	if err != nil {
		log.Println(err.Error())
	}
	if err = util.EmailCheckers(ctx, requestBody.ResourceType,
		checkersEmail); err != nil {
		log.Println(err.Error())
	}

	bod, err := json.Marshal(responseBody)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 201,
			Body:       err.Error(),
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 201,
		Body:       string(bod),
	}, nil
}

func main() {
	lambda.Start(CreateTransactionHandler)
}
