package aws

import (
	"github.com/codegangsta/cli"
	"github.com/kgraney/cloud_provision/lib"
)

type AwsProvider struct {
}

var _ cloud_provision.CloudProvider = AwsProvider{}

func (p AwsProvider) GetCommands() []cli.Command {
	return []cli.Command{{
		Name:  "aws",
		Usage: "Provision VMs to Amazon Web Services",
		Subcommands: []cli.Command{
			{
				Name:  "create-ami",
				Usage: "Create an AMI",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "image-file",
						Usage: "Image file to create from",
						Value: "",
					},
					cli.StringFlag{
						Name:  "ami-name",
						Usage: "Name of the AMI to create",
						Value: "",
					},
					cli.IntFlag{
						Name:  "ami-size",
						Usage: "Size of the AMI to create (in GB)",
						Value: 80,
					},
					cli.StringFlag{
						Name:  "vpc-id",
						Usage: "The Id of the VPC to use for image creation",
						Value: "vpc-6eda800a",
					},
					cli.StringFlag{
						Name:  "subnet-id",
						Usage: "The Id of the subnet to use for image creation",
						Value: "subnet-3441cd42",
					},
				},
				Action: func(c *cli.Context) {
					creator := AmiCreator{
						ImageFile: c.String("image-file"),
						AmiName:   c.String("ami-name"),
						AmiSize:   int64(c.Int("ami-size")),
						VpcId:     c.String("vpc-id"),
						SubnetId:  c.String("subnet-id"),
					}
					creator.Create()
				},
			},
		},
	}}
}
