package main

// import (
// 	"testing"

// 	"github.com/aws/aws-cdk-go/awscdk/v2"
// 	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
// 	"github.com/aws/jsii-runtime-go"
// )

// example tests. To run these tests, uncomment this file along with the
// example resource in bg_deploy_sample_test.go
// func TestBgDeploySampleStack(t *testing.T) {
// 	// GIVEN
// 	app := awscdk.NewApp(nil)

// 	// WHEN
// 	stack := NewBgDeploySampleStack(app, "MyStack", nil)

// 	// THEN
// 	template := assertions.Template_FromStack(stack, nil)

// 	template.HasResourceProperties(jsii.String("AWS::SQS::Queue"), map[string]interface{}{
// 		"VisibilityTimeout": 300,
// 	})
// }
