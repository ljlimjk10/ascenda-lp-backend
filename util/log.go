package util

import (
	"log"
	"net/http"
	"regexp"
	"time"

	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/vynious/ascenda-lp-backend/types"
)

func CreateLogEntry(customLog types.Log) error {
	// Specify your AWS credentials and region here
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("ap-southeast-1"),
		Credentials: credentials.NewStaticCredentials("AKIAUAWJVJ7IQGRU6FGJ", "tsJfzFep6wKQWX+DSOM20xeLQtiZYQdBkhs2IL3N", ""),
	})
	if err != nil {
		return err
	}
	log.Printf("Connected to log db")
	// Create a DynamoDB client
	svc := dynamodb.New(sess)

	// Filter PII in the action field
	filteredAction := filterPII(customLog.Action)

	// Generate a UUID for the log ID
	logID := uuid.New().String()

	// Set the Timestamp field to the current time
	customLog.Timestamp = time.Now().UTC()
	log.Printf("Putting in db")
	input := &dynamodb.PutItemInput{
		TableName: aws.String("logs"),
		Item: map[string]*dynamodb.AttributeValue{
			"log_id": {
				S: aws.String(logID),
			},
			"UserID": {
				S: aws.String(customLog.UserId),
			},
			"Type": {
				S: aws.String(customLog.Type),
			},
			"Action": {
				S: aws.String(filteredAction), // Redact PII in the action field
			},
			"UserLocation": {
				S: aws.String(customLog.UserLocation),
			},
			"Timestamp": {
				S: aws.String(customLog.Timestamp.Format(time.RFC3339)),
			},
			"TTL": {
				S: aws.String(customLog.TTL),
			},
		},
	}

	_, err = svc.PutItem(input)
	if err != nil {
		log.Printf("Error creating log entry: %v", err)
		return err
	}
	log.Printf("putted")
	return nil
}

func filterPII(message string) string {
	// Custom logic to redact PII from the message
	// Redact email addresses
	re := regexp.MustCompile(`[\w\.\-]+@[a-zA-Z0-9\-]+\.[a-zA-Z0-9\-\.]+`)
	filteredMessage := re.ReplaceAllString(message, "[REDACTED_EMAIL]")

	// Redact user names
	userNames := []string{"John", "Doe", "Jane", "Smith"} // Add more user names as needed
	for _, name := range userNames {
		filteredMessage = strings.ReplaceAll(filteredMessage, name, "[REDACTED_NAME]")
	}

	return filteredMessage
}

func HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract user ID, action type, and action description from the request
	userID := request.QueryStringParameters["UserID"]
	actionType := request.QueryStringParameters["ActionType"]
	// actionDescription := request.QueryStringParameters["ActionDescription"]

	// ip := request.RequestContext.Identity.SourceIP

	// userLocation, err := GetLocationFromIP(ip)
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }

	// filteredDescription, err := FilterPIIWithMacie(actionDescription)
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }
	// Create a log entry
	logs := types.Log{
		LogId:  "unique_log_id",
		UserId: userID,
		Action: actionType,
		Type:   "user_action",
		// Action:       actionDescription,
		UserLocation: "here",
		Timestamp:    time.Now(),
		TTL:          "",
	}
	// Store the log entry in DynamoDB
	err := CreateLogEntry(logs)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Log entry created successfully",
	}, nil
}
