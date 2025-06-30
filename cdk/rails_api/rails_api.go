package main

import (
	"os"

	"rails_api/components/network"
	"rails_api/components/rds"
	"rails_api/components/service"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/joho/godotenv"
)

type RailsApiStackProps struct {
	awscdk.StackProps
}

func NewRailsApiStack(scope constructs.Construct, id string, props *RailsApiStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	network := network.NewNetwork(stack)

	rds.NewRDS(stack, network)

	service.NewService(stack, network)

	return stack
}

func main() {
	defer jsii.Close()

	// 環境変数読み込み
	godotenv.Load()

	app := awscdk.NewApp(nil)

	NewRailsApiStack(app, "rails-api-stack", &RailsApiStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("ACCOUNT_ID")),
		Region:  jsii.String(os.Getenv("REGION")),
	}
}
