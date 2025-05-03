package main

import (
    "os"
    "github.com/aws/aws-cdk-go/awscdk/v2"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
    "github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
    "github.com/aws/constructs-go/constructs/v10"
    "github.com/aws/jsii-runtime-go"
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

    // VPCの作成
    vpc := awsec2.NewVpc(stack, jsii.String("rails_api"), &awsec2.VpcProps{
      MaxAzs: jsii.Number(2),
      NatGateways: jsii.Number(0),
    })

    vpc.AddGatewayEndpoint(jsii.String("com.amazonaws.ap-northeast-1.s3"), &awsec2.GatewayVpcEndpointOptions{
      Service: awsec2.GatewayVpcEndpointAwsService_S3(),
    })

    vpc.AddInterfaceEndpoint(jsii.String("com.amazonaws.ap-northeast-1.ecr.api"), &awsec2.InterfaceVpcEndpointOptions{
      Service: awsec2.InterfaceVpcEndpointAwsService_ECR(),
    })

    vpc.AddInterfaceEndpoint(jsii.String("com.amazonaws.ap-northeast-1.ecr.dkr"), &awsec2.InterfaceVpcEndpointOptions{
      Service: awsec2.InterfaceVpcEndpointAwsService_ECR_DOCKER(),
    })

    // sg for ALB
    albSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("rails_api_sg_alb"), &awsec2.SecurityGroupProps{
      SecurityGroupName: jsii.String("rails_api_sg_alb"),
      Vpc: vpc,
      AllowAllOutbound: jsii.Bool(true),
    })
    albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(80)), jsii.String("http from anywhere"), jsii.Bool(false))

    // sg for ECS
    ecsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("rails_api_sg_ecs"), &awsec2.SecurityGroupProps{
      SecurityGroupName: jsii.String("rails_api_sg_ecs"),
      Vpc: vpc,
      AllowAllOutbound: jsii.Bool(true),
    })
    ecsSecurityGroup.AddIngressRule(albSecurityGroup, awsec2.Port_Tcp(jsii.Number(3000)), jsii.String("http from alb to rails"), jsii.Bool(false))

    repository := awsecr.Repository_FromRepositoryName(stack, jsii.String("rails_api_repository"), jsii.String("rails_api"))

    // ECSクラスターの作成
    cluster := awsecs.NewCluster(stack, jsii.String("rails_api_cluster"), &awsecs.ClusterProps{
      ClusterName: jsii.String("rails_api_cluster"),
      Vpc: vpc,
    })

    // タスク定義の作成
    taskDef := awsecs.NewFargateTaskDefinition(stack, jsii.String("rails_api_taskdef"), &awsecs.FargateTaskDefinitionProps{
      Family:         jsii.String("rails_api_taskdef"),
      MemoryLimitMiB: jsii.Number(1024),
      Cpu:            jsii.Number(512),
    })

    // コンテナの追加
    taskDef.AddContainer(jsii.String("rails_api_container"), &awsecs.ContainerDefinitionOptions{
      ContainerName: jsii.String("rails_api_container"),
      Image: awsecs.ContainerImage_FromEcrRepository(repository, jsii.String("latest")),
      PortMappings: &[]*awsecs.PortMapping{
        {
          ContainerPort: jsii.Number(3000),
          HostPort: jsii.Number(3000),
        },
      },
      Environment: &map[string]*string{
        "RAILS_ENV": jsii.String("production"),
        "RAILS_SERVE_STATIC_FILES": jsii.String("true"),
        "RAILS_MASTER_KEY": jsii.String(os.Getenv("RAILS_MASTER_KEY")),
      },
      Logging: awsecs.LogDrivers_AwsLogs(&awsecs.AwsLogDriverProps{
        StreamPrefix: jsii.String("rails-api"),
      }),
    })

    // Fargateサービスの作成
    service := awsecs.NewFargateService(stack, jsii.String("rails_api_service"), &awsecs.FargateServiceProps{
      ServiceName: jsii.String("rails_api_service"),
      Cluster:        cluster,
      TaskDefinition: taskDef,
      DesiredCount:   jsii.Number(1),
      AssignPublicIp: jsii.Bool(true),
      SecurityGroups: &[]awsec2.ISecurityGroup{ecsSecurityGroup},
    })

    // ALBの作成
    alb := awselasticloadbalancingv2.NewApplicationLoadBalancer(stack, jsii.String("rails_api_alb"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
      Vpc: vpc,
      InternetFacing: jsii.Bool(true),
      SecurityGroup: albSecurityGroup,
      LoadBalancerName: jsii.String("rails_api_alb"),
    })

    // ALBリスナーの作成
    listener := alb.AddListener(jsii.String("rails_api_listener"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
      Port: jsii.Number(80),
      Open: jsii.Bool(true),
    })

    // ALBリスナーにターゲットグループを追加
    listener.AddTargets(jsii.String("rails_api_target"), &awselasticloadbalancingv2.AddApplicationTargetsProps{
      Port: jsii.Number(3000),
      Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
      Targets: &[]awselasticloadbalancingv2.IApplicationLoadBalancerTarget{
        service.LoadBalancerTarget(&awsecs.LoadBalancerTargetOptions{
          ContainerName: jsii.String("rails_api_container"),
          ContainerPort: jsii.Number(3000),
        }),
      },
      HealthCheck: &awselasticloadbalancingv2.HealthCheck{
        Path: jsii.String("/up"),
        Interval: awscdk.Duration_Seconds(jsii.Number(60)),
        Timeout: awscdk.Duration_Seconds(jsii.Number(5)),
        HealthyThresholdCount: jsii.Number(2),
        UnhealthyThresholdCount: jsii.Number(5),
      },
      DeregistrationDelay: awscdk.Duration_Seconds(jsii.Number(30)),
    })

    return stack
}

func main() {
    defer jsii.Close()

    app := awscdk.NewApp(nil)

    NewRailsApiStack(app, "rails-api-stack", &RailsApiStackProps{
        awscdk.StackProps{
            Env: env(),
        },
    })

    app.Synth(nil)
}

func env() *awscdk.Environment {
    return nil
}
