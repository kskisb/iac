package service

import (
	"rails_api/components/network"

	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Service struct {
	Repository    awsecr.IRepository
	Cluster       awsecs.Cluster
	TaskDef       awsecs.FargateTaskDefinition
	Service       awsecs.FargateService
	ExecutionRole awsiam.IRole
	TaskRole      awsiam.IRole
}

func NewService(stack constructs.Construct, network *network.Network) *Service {
	vpc := network.Vpc
	sg := network.EcsSecurityGroup
	targetGroup1 := network.TargetGroup1

	resourceName := os.Getenv("RESOURCE_NAME")
	repositoryName := os.Getenv("REPOSITORY_NAME")
	railsMasterKey := os.Getenv("RAILS_MASTER_KEY")

	repository := awsecr.Repository_FromRepositoryName(stack, jsii.String(resourceName+"-repository"), jsii.String(repositoryName))

	cluster := awsecs.NewCluster(stack, jsii.String(resourceName+"-cluster"), &awsecs.ClusterProps{
		ClusterName: jsii.String(resourceName + "-cluster"),
		Vpc:         vpc,
	})

	taskRole := awsiam.NewRole(stack, jsii.String(resourceName+"-task-role"), &awsiam.RoleProps{
		RoleName:  jsii.String(resourceName + "-task-role"),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ecs-tasks.amazonaws.com"), nil),
	})

	executionRole := awsiam.NewRole(stack, jsii.String(resourceName+"-execution-role"), &awsiam.RoleProps{
		RoleName:  jsii.String(resourceName + "-execution-role"),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ecs-tasks.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AmazonECSTaskExecutionRolePolicy")),
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("CloudWatchLogsFullAccess")),
		},
	})

	taskDef := awsecs.NewFargateTaskDefinition(stack, jsii.String(resourceName+"-taskdef"), &awsecs.FargateTaskDefinitionProps{
		Family:         jsii.String(resourceName + "-taskdef"),
		Cpu:            jsii.Number(256),
		MemoryLimitMiB: jsii.Number(512),
		TaskRole:       taskRole,
		ExecutionRole:  executionRole,
	})

	railsContainer := taskDef.AddContainer(jsii.String("rails"), &awsecs.ContainerDefinitionOptions{
		ContainerName:        jsii.String("rails"),
		Image:                awsecs.ContainerImage_FromEcrRepository(repository, jsii.String("latest")),
		Cpu:                  jsii.Number(256),
		MemoryReservationMiB: jsii.Number(512),
		Essential:            jsii.Bool(true),
		Environment: &map[string]*string{
			"TZ":                       jsii.String("Asia/Tokyo"),
			"RAILS_ENV":                jsii.String("production"),
			"RAILS_SERVE_STATIC_FILES": jsii.String("true"),
			"RAILS_MASTER_KEY":         jsii.String(railsMasterKey),
			"DB_HOST":                  jsii.String(os.Getenv("DB_HOST")),
			"DB_USERNAME":              jsii.String(os.Getenv("DB_USERNAME")),
			"DB_PASSWORD":              jsii.String(os.Getenv("DB_PASSWORD")),
			"DB_PORT":                  jsii.String(os.Getenv("DB_PORT")),
		},
		Logging: awsecs.LogDrivers_AwsLogs(&awsecs.AwsLogDriverProps{
			LogGroup: awslogs.NewLogGroup(stack, jsii.String(resourceName+"-log-group"), &awslogs.LogGroupProps{
				LogGroupName:  jsii.String("/aws/ecs/" + resourceName + "-log-group"),
				RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
				Retention:     awslogs.RetentionDays_ONE_WEEK,
			}),
			StreamPrefix: jsii.String(resourceName + "-rails"),
		}),
	})

	railsContainer.AddPortMappings(&awsecs.PortMapping{
		Name:          jsii.String("rails"),
		ContainerPort: jsii.Number(3000),
		HostPort:      jsii.Number(3000),
		Protocol:      awsecs.Protocol_TCP,
	})

	service := awsecs.NewFargateService(stack, jsii.String(resourceName+"-service"), &awsecs.FargateServiceProps{
		ServiceName:            jsii.String(resourceName + "-service"),
		Cluster:                cluster,
		TaskDefinition:         taskDef,
		DesiredCount:           jsii.Number(1),
		AssignPublicIp:         jsii.Bool(false),
		HealthCheckGracePeriod: awscdk.Duration_Seconds(jsii.Number(3600)),
		SecurityGroups:         &[]awsec2.ISecurityGroup{sg},
		// DeploymentController: &awsecs.DeploymentController{
		// 	Type: awsecs.DeploymentControllerType_CODE_DEPLOY,
		// },
	})

	targetGroup1.AddTarget(service.LoadBalancerTarget(&awsecs.LoadBalancerTargetOptions{
		ContainerName: railsContainer.ContainerName(),
		ContainerPort: jsii.Number(3000),
	}))

	return &Service{
		Repository:    repository,
		Cluster:       cluster,
		TaskDef:       taskDef,
		Service:       service,
		ExecutionRole: executionRole,
		TaskRole:      taskRole,
	}
}
