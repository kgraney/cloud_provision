package aws

import (
	"encoding/base64"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ConsoleStreamer struct {
	InstanceId string
}

func (s ConsoleStreamer) Run() {
	var lastConsoleUpdate time.Time
	var terminate chan bool
	var lastLogged string

	awsConfig := aws.NewConfig().WithRegion("us-east-1")
	awsSession := session.New(awsConfig)
	ec2Session := ec2.New(awsSession)

	instanceLogger := log.WithFields(log.Fields{
		"instanceId": s.InstanceId,
	})

	for {
		select {
		case <-terminate:
			return
		default:
			resp, err := ec2Session.GetConsoleOutput(&ec2.GetConsoleOutputInput{
				InstanceId: aws.String(s.InstanceId),
			})
			if err != nil {
				instanceLogger.Warning("Error getting console output: ", err)
			}
			if lastConsoleUpdate != *resp.Timestamp {
				lastConsoleUpdate = *resp.Timestamp
				instanceLogger.Info("Console updated @ ", resp.Timestamp)
				if resp.Output != nil {
					decodedBytes, _ := base64.StdEncoding.DecodeString(*resp.Output)
					decodedString := string(decodedBytes[:])
					fmt.Printf(DeduplicatedLogStream(lastLogged, decodedString))
					lastLogged = decodedString
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func DeduplicatedLogStream(previous, current string) string {
	if previous == "" || current == "" {
		return current
	}
	for i := 1; i <= len(previous); i++ {
		fmt.Printf("%d -- %s -- %s\n", i, previous[len(previous)-i:], current[:i])
		if previous[len(previous)-i:] == current[:i] {
			return current[i:]
		}
	}
	return current
}
