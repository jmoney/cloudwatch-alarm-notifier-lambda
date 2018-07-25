// Copyright 2018 Jonathan Monette
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jmoney8080/go-gadget-slack"
)

var (
	// Info Logger
	Info *log.Logger
	// Warning Logger
	Warning *log.Logger
	// Error Logger
	Error *log.Logger

	slackClient               *slack.Client
	slackAttachmentsChunkSize int
	slackMonitorChannel       string
)

// CloudWatchAlarmEvent the cloudwatch event on the SNS event
type CloudWatchAlarmEvent struct {
	AlarmName        string                      `json:"AlarmName"`
	AlarmDescription string                      `json:"AlarmDescription"`
	AWSAccountID     string                      `json:"AWSAccountId"`
	NewStateValue    string                      `json:"NewStateValue"`
	NewStateReason   string                      `json:"NewStateReason"`
	StateChangeTime  string                      `json:"StateChangeTime"`
	Region           string                      `json:"Region"`
	OldStateValue    string                      `json:"OldStateValue"`
	Trigger          CloudWatchAlarmEventTrigger `json:"Trigger"`
}

// CloudWatchAlarmEventTrigger trigger hash from the CloudWatchAlarm Event
type CloudWatchAlarmEventTrigger struct {
	Period             int     `json:"Period"`
	EvaluationPeriods  int     `json:"EvaluationPeriods"`
	ComparisonOperator string  `json:"ComparisonOperator"`
	Threshold          float32 `json:"Threshold"`
}

func init() {

	Info = log.New(os.Stdout,
		"[INFO]: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(os.Stdout,
		"[WARNING]: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(os.Stderr,
		"[ERROR]: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	slackClient = slack.New(http.Client{Timeout: 10 * time.Second}, os.Getenv("SLACK_WEBHOOK"))
	slackAttachmentsChunkSize = 100
	slackMonitorChannel = os.Getenv("SLACK_MONITOR_CHANNEL")
}

func main() {
	lambda.Start(HandleRequest)
}

// HandleRequest function that the lambda runtime service calls
func HandleRequest(ctx context.Context, event events.SNSEvent) error {
	slackAttachments := []slack.Attachment{}

	for _, eventRecord := range event.Records {
		cloudWatchAlarmEvent := CloudWatchAlarmEvent{}
		json.NewDecoder(strings.NewReader(eventRecord.SNS.Message)).Decode(&cloudWatchAlarmEvent)

		color := "good"
		if cloudWatchAlarmEvent.NewStateValue == "ALARM" {
			color = "danger"
		} else if cloudWatchAlarmEvent.NewStateValue == "INSUFFICIENT_DATA" {
			color = "warning"
		}

		slackAttachment := slack.Attachment{
			Color:      color,
			Title:      eventRecord.SNS.Subject,
			Text:       cloudWatchAlarmEvent.NewStateReason,
			Footer:     os.Getenv("AWS_LAMBDA_FUNCTION_NAME"),
			FooterIcon: "https://d1d05r7k0qlw4w.cloudfront.net/dist-cbe91c5a8477701757ff6752aae4c6f892018972/img/favicon.ico",
			Ts:         time.Now().UnixNano() / int64(time.Second),
			AttachmentField: []slack.AttachmentField{
				{
					Title: "AccountID",
					Value: cloudWatchAlarmEvent.AWSAccountID,
					Short: true,
				},
				{
					Title: "Region",
					Value: cloudWatchAlarmEvent.Region,
					Short: true,
				},
				{
					Title: "Period",
					Value: fmt.Sprintf("%v", cloudWatchAlarmEvent.Trigger.Period),
					Short: true,
				},
				{
					Title: "Threshold",
					Value: fmt.Sprintf("%v", cloudWatchAlarmEvent.Trigger.Threshold),
					Short: true,
				},
				{
					Title: "Evaluated Periods",
					Value: fmt.Sprintf("%v", cloudWatchAlarmEvent.Trigger.EvaluationPeriods),
					Short: true,
				},
				{
					Title: "Comparison Operator",
					Value: cloudWatchAlarmEvent.Trigger.ComparisonOperator,
					Short: true,
				},
			},
		}
		slackAttachments = append(slackAttachments, slackAttachment)
	}

	if len(slackAttachments) != 0 {
		// Here we are chunking up the attachments.  Slack only allows 100 attachments in one post. While that'd be insane and absurd to do, it's a known limit
		// we can easily account for in the code
		for i := 0; i < len(slackAttachments); i += slackAttachmentsChunkSize {
			end := i + slackAttachmentsChunkSize
			if end > len(slackAttachments) {
				end = len(slackAttachments)
			}

			chunkedSlackAttachments := slackAttachments[i:end]
			payload := slack.Payload{
				Channel:     slackMonitorChannel,
				Attachments: chunkedSlackAttachments,
			}
			resp, err := (*slackClient).Send(payload)
			if err != nil {
				Error.Println(err)
			} else {
				Info.Println(resp)
			}
		}
	} else {
		Warning.Println("No Slack Sent")
	}
	return nil
}
