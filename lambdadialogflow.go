// Package lambdadialogflow simplifies writing dialogflow fulfillemts running on AWS Lambda/Serverless
package lambdadialogflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang/protobuf/jsonpb"
	df "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

// Agent contains the original dialogflow request and convenient methods to construct a response
type Agent struct {
	req *df.WebhookRequest
	res *df.WebhookResponse
}

// WebhookHandler handles one dialogflow request
type WebhookHandler func(*Agent)

var (
	handlerMap = make(map[string]WebhookHandler)
)

// Request returns the  dialogflow request
func (w *Agent) Request() *df.WebhookRequest {
	return w.req
}

// Response returns the  dialogflow response
func (w *Agent) Response() *df.WebhookResponse {
	return w.res
}

// Action returns the action from the dialogflow request
func (w *Agent) Action() string {
	return w.req.QueryResult.Action
}

// Session returns the session id for this request
func (w *Agent) Session() string {
	return w.req.Session
}

// GetStringParam returns a string parameter
func (w *Agent) GetStringParam(name string) string {
	return w.req.QueryResult.Parameters.GetFields()[name].GetStringValue()
}

// GetNumberParam returns a float64 parameter
func (w *Agent) GetNumberParam(name string) float64 {
	return w.req.QueryResult.Parameters.GetFields()[name].GetNumberValue()
}

// Say lets the agent return a message to the user
func (w *Agent) Say(someText string) {
	w.res.FulfillmentText = someText
}

// SetContext is used to set the output context
func (w *Agent) SetContext(contextname string, lifetime int32) {
	ctx := &df.Context{Name: contextname, LifespanCount: lifetime}
	w.res.OutputContexts = append(w.res.OutputContexts, ctx)
}

// Register a new webhook handler for an action
func Register(action string, handler WebhookHandler) {
	handlerMap[action] = handler
}

// newAgent creates a new agent based on the webhook request from dialogflow
func newAgent(webhookRequest *df.WebhookRequest) (*Agent, error) {
	w := &Agent{req: webhookRequest, res: &df.WebhookResponse{}}
	return w, nil
}

// HandleRequest handles the dialogflow request coming in via the lambda api gateway
func HandleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	webhookRequest := &df.WebhookRequest{}
	err := jsonpb.Unmarshal(strings.NewReader(req.Body), webhookRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400},
			fmt.Errorf("unable to decode webhook request: %v", err)
	}

	w, err := newAgent(webhookRequest)

	webhookHandler := handlerMap[w.Action()]
	if webhookHandler == nil {
		return events.APIGatewayProxyResponse{StatusCode: 404},
			fmt.Errorf("no handler defined for action: %v", w.Action())
	}

	webhookHandler(w)

	var buf bytes.Buffer
	body, err := json.Marshal(w.res)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}
	json.HTMLEscape(&buf, body)

	resp := events.APIGatewayProxyResponse{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	return resp, err
}

// Start listening on requests
func Start() {
	lambda.Start(HandleRequest)
}
