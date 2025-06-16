package service

import (
	"bg_deploy_sample/components/network"

	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Service struct {
	NginxRepository awsecr.IRepository
	Cluster         awsecs.Cluster
	TaskDef         awsecs.FargateTaskDefinition
	Service         awsecs.FargateService
	ExecutionRole   awsiam.IRole
	TaskRole        awsiam.IRole
}

type Network struct {
	Vpc           awsec2.Vpc
	SecurityGroup awsec2.SecurityGroup
	TargetGroup1  awselasticloadbalancingv2.ApplicationTargetGroup
}

func NewService(stack constructs.Construct, network *network.Network) *Service {
	vpc := network.Vpc
	sg := network.EcsSecurityGroup
	targetGroup1 := network.TargetGroup1

	resourceName := os.Getenv("RESOURCE_NAME")
	repositoryName := os.Getenv("REPOSITORY_NAME")

	nginxRepository := awsecr.Repository_FromRepositoryName(stack, jsii.String(resourceName+"-repository"), jsii.String(repositoryName))

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

	nginxContainer := taskDef.AddContainer(jsii.String("nginx"), &awsecs.ContainerDefinitionOptions{
		ContainerName:        jsii.String("nginx"),
		Image:                awsecs.ContainerImage_FromEcrRepository(nginxRepository, jsii.String("latest")),
		Cpu:                  jsii.Number(256),
		MemoryReservationMiB: jsii.Number(512),
		Essential:            jsii.Bool(true),
		Environment: &map[string]*string{
			"TZ": jsii.String("Asia/Tokyo"),
		},
		// Logging: awsecs.LogDrivers_AwsLogs(&awsecs.AwsLogDriverProps{
		// 	LogGroup: awslogs.NewLogGroup(stack, jsii.String(resourceName+"-log-group"), &awslogs.LogGroupProps{
		// 		LogGroupName:  jsii.String("/aws/ecs/" + resourceName + "log-group"),
		// 		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		// 		Retention:     awslogs.RetentionDays_ONE_WEEK,
		// 	}),
		// 	StreamPrefix: jsii.String(resourceName + "-nginx"),
		// }),
	})

	nginxContainer.AddPortMappings(&awsecs.PortMapping{
		Name:          jsii.String("nginx"),
		ContainerPort: jsii.Number(80),
		HostPort:      jsii.Number(80),
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
		DeploymentController: &awsecs.DeploymentController{
			Type: awsecs.DeploymentControllerType_CODE_DEPLOY,
		},
	})

	targetGroup1.AddTarget(service.LoadBalancerTarget(&awsecs.LoadBalancerTargetOptions{
		ContainerName: nginxContainer.ContainerName(),
		ContainerPort: jsii.Number(80),
	}))

	return &Service{
		NginxRepository: nginxRepository,
		Cluster:         cluster,
		TaskDef:         taskDef,
		Service:         service,
		ExecutionRole:   executionRole,
		TaskRole:        taskRole,
	}
}
