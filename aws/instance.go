package aws

import (
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/crypto/ssh"
)

type instance struct {
	runInstancesInput *ec2.RunInstancesInput
	ec2               *ec2.EC2

	instanceId  *string
	keyPairName *string
	privateKey  ssh.Signer
	terminate   chan bool

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
	i.createKeyPair()
	result, err := i.ec2.RunInstances(i.runInstancesInput)
	if err != nil {
		return nil, err
	}
	i.instanceId = result.Instances[0].InstanceId

	go i.StreamConsole()
	return i.instanceId, err
}

func (i *instance) RunSshCommand(cmd string) error {
	sshConfig := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(i.privateKey),
		},
		// We have a long timeout since this is also possibly waiting for SSH to be available on the
		// host
		Timeout: 5 * time.Minute,
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", *i.PrivateIp()), sshConfig)
	if err != nil {
		i.logger.Warn("Failed to dial: ", err)
		return err
	}

	session, err := connection.NewSession()
	if err != nil {
		i.logger.Warn("Failed to create SSH session: ", err)
		return err
	}
	bytes, err := session.CombinedOutput(cmd)
	if err != nil {
		i.logger.Warn("Failed to run command: ", err)
		return err
	}
	i.logger.WithFields(log.Fields{
		"source": "ssh",
	}).Info(string(bytes))

	return nil
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
	i.ec2.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: i.keyPairName,
	})
}

func (i *instance) PrivateIp() *string {
	instance, _ := i.describeInstance()
	return instance.PublicIpAddress //TODO(kmg) PrivateIpAddress
}

func (i *instance) Forget() {
	i.terminate <- true
}

func (i *instance) StreamConsole() {
	// TODO(kmg): add support for only logging new content in the console
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
				time.Sleep(2 * time.Second)
				continue
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

// Create a private key for accessing this instance
func (i *instance) createKeyPair() {
	i.keyPairName = aws.String("SomeRandomKeyPairName")

	resp, _ := i.ec2.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: i.keyPairName,
	})

	log.Info("Instance private key is: ", *resp.KeyMaterial)
	privateKey, _ := pem.Decode([]byte(*resp.KeyMaterial))
	i.privateKey, _ = ssh.ParsePrivateKey(privateKey.Bytes)
}
