// Package lambdadialogflow simplifies writing dialogflow fulfillemts running on AWS Lambda/Serverless
package lambdadialogflow

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang/protobuf/jsonpb"
	_structpb "github.com/golang/protobuf/ptypes/struct"
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

func (w *Agent) getField(name string) *_structpb.Value {
	f := w.req.QueryResult.Parameters.GetFields()[name]
	if f != nil {
		return f
	}
	return w.req.OriginalDetectIntentRequest.Payload.GetFields()[name]
}

// GetStringParam returns a string parameter
func (w *Agent) GetStringParam(name string) string {
	f := w.getField(name)
	if f != nil {
		return f.GetStringValue()
	}
	return ""
}

// GetNumberParam returns a float64 parameter
func (w *Agent) GetNumberParam(name string) float64 {
	f := w.getField(name)
	if f != nil {
		return f.GetNumberValue()
	}
	return 0
}

// AddPayload adds a strint/value to the response payload
func (w *Agent) AddPayload(name, value string) {
	stringValue := &_structpb.Value{
		Kind: &_structpb.Value_StringValue{
			StringValue: value,
		},
	}

	if w.Response().Payload != nil && w.Response().Payload.Fields != nil {
		w.Response().Payload.Fields[name] = stringValue
	} else {
		w.Response().Payload = &_structpb.Struct{
			Fields: map[string]*_structpb.Value{
				name: stringValue,
			},
		}
	}
}

// AddJSONPayloadBase64 base64 encodes the value before adding it as payload
func (w *Agent) AddJSONPayloadBase64(name, value string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(value))
	w.AddPayload(name, encoded)
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
	marshaler := &jsonpb.Marshaler{}
	err = marshaler.Marshal(&buf, w.res)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

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
