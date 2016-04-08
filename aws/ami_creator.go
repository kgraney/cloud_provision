package aws

import (
	"encoding/base64"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type AmiCreator struct {
	ImageFile string
	AmiName   string
	AmiSize   int64
	DryRun    bool
	VpcId     string
	SubnetId  string

	resources Resources
	svc       *ec2.EC2
}

type Resources struct {
	SecurityGroupId  string
	CopierInstanceId string
	TargetVolumeId   string
}

func (c *AmiCreator) LogFatal(errs ...interface{}) {
	log.Error("Fatal error! Performing AWS cleanup.", errs)
	c.Cleanup()
	log.Fatal(errs)
}

func (c *AmiCreator) RecordResource(resourceId *string, target *string) {
	*target = *resourceId
	_, err := c.svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{resourceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("service"),
				Value: aws.String("ami-creation"),
			},
		},
	})
	log.Info("Creating tags on resource: ", *resourceId)
	if err != nil {
		log.Warn("Could not create tags for resource ", resourceId, err)
		return
	}
}

func (c *AmiCreator) Cleanup() {
	log.Info("Cleaning up resources..")

	log.Info("Terminating instance ", c.resources.CopierInstanceId)
	c.svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(c.resources.CopierInstanceId)},
	})

	c.svc.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(c.resources.CopierInstanceId)},
	})

	log.Info("Deleting security group ", c.resources.SecurityGroupId)
	c.svc.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(c.resources.SecurityGroupId),
	})
}

func (c *AmiCreator) StreamConsole(instanceId *string, terminate chan bool) {
	var lastConsoleUpdate time.Time

	instanceLogger := log.WithFields(log.Fields{
		"instanceId": *instanceId,
	})

	for {
		select {
		case <-terminate:
			return
		default:
			resp, err := c.svc.GetConsoleOutput(&ec2.GetConsoleOutputInput{
				InstanceId: instanceId,
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
					for _, line := range strings.Split(decodedString, "\n") {
						instanceLogger.WithFields(log.Fields{
							"source": "console",
						}).Info(line)
					}
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *AmiCreator) Create() {
	log.Info("Creating an AMI with ", c.ImageFile)

	awsConfig := aws.NewConfig().WithRegion("us-east-1")
	awsSession := session.New(awsConfig)

	copierInstanceTerminated := make(chan bool)

	c.svc = ec2.New(awsSession)
	c.CreateSecurityGroup()
	c.CreateInstance()
	go c.StreamConsole(&c.resources.CopierInstanceId, copierInstanceTerminated)

	log.Info("Waiting for copier instance to be running")
	c.svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(c.resources.CopierInstanceId)},
	})

	log.Info("Waiting just because")
	time.Sleep(120 * time.Second)
	c.Cleanup()

	copierInstanceTerminated <- true
}

func (c *AmiCreator) CreateSecurityGroup() *string {
	sgInput := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("ThisShouldBeARandomlyGeneratedName"),
		Description: aws.String("Security group created by cloud_provision script"),
		DryRun:      aws.Bool(c.DryRun),
		VpcId:       aws.String(c.VpcId),
	}

	sgOutput, err := c.svc.CreateSecurityGroup(sgInput)
	if err != nil {
		c.LogFatal(err)
	}

	log.Info("Created security group ", *sgOutput.GroupId)
	c.RecordResource(sgOutput.GroupId, &c.resources.SecurityGroupId)
	return sgOutput.GroupId
}

func (c *AmiCreator) CreateInstance() {
	runResult, err := c.svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String("ami-fce3c696"), // Ubuntu
		InstanceType:     aws.String("t2.micro"),
		MinCount:         aws.Int64(1),
		MaxCount:         aws.Int64(1),
		SecurityGroupIds: []*string{aws.String(c.resources.SecurityGroupId)},
		SubnetId:         aws.String(c.SubnetId),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					VolumeSize:          aws.Int64(100 + c.AmiSize),
					VolumeType:          aws.String("gp2"),
				},
			},
			{
				DeviceName: aws.String("/dev/xvdb"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					Encrypted:           aws.Bool(false),
					VolumeSize:          aws.Int64(c.AmiSize),
					VolumeType:          aws.String("gp2"),
				},
			}},
	})

	if err != nil {
		c.LogFatal("Could not create instance ", err)
		return
	}
	c.RecordResource(runResult.Instances[0].InstanceId, &c.resources.CopierInstanceId)

	result, _ := c.svc.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String("blockDeviceMapping"),
		InstanceId: aws.String(c.resources.CopierInstanceId),
	})
	log.Info(result.BlockDeviceMappings)
}
