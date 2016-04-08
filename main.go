package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/kgraney/cloud_provision/aws"
	"github.com/kgraney/cloud_provision/lib"
)

func main() {
	app := cli.NewApp()
	app.Name = "cloud_provision"
	app.Usage = "Provisioning custom infrastructure in the public cloud"
	app.Version = "0.0.1"

	providers := []cloud_provision.CloudProvider{
		aws.AwsProvider{},
	}

	for _, provider := range providers {
		for _, command := range provider.GetCommands() {
			app.Commands = append(app.Commands, command)
		}
	}

	app.Run(os.Args)
}
