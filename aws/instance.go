package aws

import (
	"encoding/base64"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type instance struct {
	runInstancesInput *ec2.RunInstancesInput
	ec2               *ec2.EC2

	instanceId *string
	terminate  chan bool

	logger log.FieldLogger
}

func NewInstance(service *ec2.EC2, input *ec2.RunInstancesInput) *instance {
	instance := new(instance)
	instance.ec2 = service
	instance.runInstancesInput = input
	instance.logger = log.WithFields(log.Fields{})
	instance.terminate = make(chan bool)
	return instance
}

func (i *instance) Start() (*string, error) {
	// TODO(kmg): assert that only one instance is being launched
	i.logger.Info("Starting an instance from AMI ", *i.runInstancesInput.ImageId)
	result, err := i.ec2.RunInstances(i.runInstancesInput)
	if err != nil {
		return nil, err
	}
	i.instanceId = result.Instances[0].InstanceId

	go i.StreamConsole()
	return i.instanceId, err
}

func (i *instance) WaitUntilRunning() {
	i.ec2.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{i.instanceId},
	})
}

func (i *instance) WaitUntilTerminated() {
	i.ec2.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{i.instanceId},
	})
}

func (i *instance) Terminate() {
	i.ec2.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{i.instanceId},
	})
}

func (i *instance) PrivateIp() *string {
	instance, _ := i.describeInstance()
	return instance.PrivateIpAddress
}

func (i *instance) Forget() {
	i.terminate <- true
}

func (i *instance) StreamConsole() {
	var lastConsoleUpdate time.Time

	i.logger = log.WithFields(log.Fields{
		"instanceId": *i.instanceId,
	})

	for {
		select {
		case <-i.terminate:
			return
		default:
			resp, err := i.ec2.GetConsoleOutput(&ec2.GetConsoleOutputInput{
				InstanceId: i.instanceId,
			})
			if err != nil {
				i.logger.Warning("Error getting console output: ", err)
			}
			if lastConsoleUpdate != *resp.Timestamp {
				lastConsoleUpdate = *resp.Timestamp
				i.logger.Info("Console updated @ ", resp.Timestamp)
				if resp.Output != nil {
					decodedBytes, _ := base64.StdEncoding.DecodeString(*resp.Output)
					decodedString := string(decodedBytes[:])
					i.logger.WithFields(log.Fields{
						"source": "console",
					}).Info(decodedString)
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (i *instance) describeInstance() (*ec2.Instance, error) {
	i.logger.Info("Describing instance")

	result, err := i.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{&ec2.Filter{
			Name:   aws.String("instance-id"),
			Values: []*string{i.instanceId},
		},
		},
	})

	return result.Reservations[0].Instances[0], err
}
