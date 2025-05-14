package main

import (
    "github.com/aws/aws-cdk-go/awscdk/v2"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
    "github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
    "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
    "github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
    "github.com/aws/constructs-go/constructs/v10"
    "github.com/aws/jsii-runtime-go"
)

type RailsApiStackProps struct {
    awscdk.StackProps
    domainName *string
    railsMasterKey *string
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

    // VPCエンドポイントの作成
    vpc.AddGatewayEndpoint(jsii.String("com.amazonaws.ap-northeast-1.s3"), &awsec2.GatewayVpcEndpointOptions{
      Service: awsec2.GatewayVpcEndpointAwsService_S3(),
    })

    vpc.AddInterfaceEndpoint(jsii.String("com.amazonaws.ap-northeast-1.ecr.api"), &awsec2.InterfaceVpcEndpointOptions{
      Service: awsec2.InterfaceVpcEndpointAwsService_ECR(),
    })

    vpc.AddInterfaceEndpoint(jsii.String("com.amazonaws.ap-northeast-1.ecr.dkr"), &awsec2.InterfaceVpcEndpointOptions{
      Service: awsec2.InterfaceVpcEndpointAwsService_ECR_DOCKER(),
    })

    // ALB用のセキュリティグループ
    albSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("rails_api_sg_alb"), &awsec2.SecurityGroupProps{
      SecurityGroupName: jsii.String("rails_api_sg_alb"),
      Vpc: vpc,
      AllowAllOutbound: jsii.Bool(true),
    })
    albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(80)), jsii.String("http from anywhere"), jsii.Bool(false))
    albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(443)), jsii.String("https from anywhere"), jsii.Bool(false))

    // ECS用のセキュリティグループ
    ecsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("rails_api_sg_ecs"), &awsec2.SecurityGroupProps{
      SecurityGroupName: jsii.String("rails_api_sg_ecs"),
      Vpc: vpc,
      AllowAllOutbound: jsii.Bool(true),
    })
    ecsSecurityGroup.AddIngressRule(albSecurityGroup, awsec2.Port_Tcp(jsii.Number(3000)), jsii.String("http from alb to rails"), jsii.Bool(false))

    // ECRリポジトリの取得
    repository := awsecr.Repository_FromRepositoryName(stack, jsii.String("rails_api_repository"), jsii.String("rails_api"))

    // ECSクラスターの作成
    cluster := awsecs.NewCluster(stack, jsii.String("rails_api_cluster"), &awsecs.ClusterProps{
      ClusterName: jsii.String("rails_api_cluster"),
      Vpc: vpc,
    })

    // タスク定義の作成
    taskDef := awsecs.NewFargateTaskDefinition(stack, jsii.String("rails_api_taskdef"), &awsecs.FargateTaskDefinitionProps{
      Family:         jsii.String("rails_api_taskdef"),
      MemoryLimitMiB: jsii.Number(512),
      Cpu:            jsii.Number(256),
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
        "RAILS_MASTER_KEY": props.railsMasterKey,
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

    // ALBのDNS名を取得
    hostedZone := awsroute53.HostedZone_FromLookup(stack, jsii.String("HostedZone"), &awsroute53.HostedZoneProviderProps{
      DomainName: props.domainName,
    })

    // ACM証明書の作成
    certificate := awscertificatemanager.NewCertificate(stack, jsii.String("rails_api_certificate"), &awscertificatemanager.CertificateProps{
      DomainName: props.domainName,
      Validation: awscertificatemanager.CertificateValidation_FromDns(hostedZone),
    })

    // ALBの作成
    alb := awselasticloadbalancingv2.NewApplicationLoadBalancer(stack, jsii.String("rails_api_alb"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
      Vpc: vpc,
      InternetFacing: jsii.Bool(true),
      SecurityGroup: albSecurityGroup,
      LoadBalancerName: jsii.String("rails-api-alb"),
    })

    // ALBリスナーの作成(http)
    alb.AddListener(jsii.String("rails_api_http_listener"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
      Port: jsii.Number(80),
      DefaultAction: awselasticloadbalancingv2.ListenerAction_Redirect(&awselasticloadbalancingv2.RedirectOptions{
        Protocol: jsii.String("HTTPS"),
        Port: jsii.String("443"),
      }),
    })

    // ALBリスナーの作成(https)
    httpsListener := alb.AddListener(jsii.String("rails_api_https_listener"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
      Port: jsii.Number(443),
      Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
      Certificates: &[]awselasticloadbalancingv2.IListenerCertificate{
        awselasticloadbalancingv2.ListenerCertificate_FromCertificateManager(certificate),
      },
    })

    // ALBリスナーにターゲットグループを追加
    httpsListener.AddTargets(jsii.String("rails_api_target"), &awselasticloadbalancingv2.AddApplicationTargetsProps{
      TargetGroupName: jsii.String("rails-api-target"),
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
        Interval: awscdk.Duration_Seconds(jsii.Number(300)),
        Timeout: awscdk.Duration_Seconds(jsii.Number(120)),
      },
    })

    // ALBのDNS名をRoute53に登録
    awsroute53.NewARecord(stack, jsii.String("ARecord"), &awsroute53.ARecordProps{
      Zone: hostedZone,
      Target: awsroute53.RecordTarget_FromAlias(awsroute53targets.NewLoadBalancerTarget(alb, nil)),
    })

    return stack
}

/*
起動・削除コマンド

cdk deploy \
  --context domain_name=${domain_name} \
  --context rails_master_key=${rails_master_key} \
  --context account_id=${account_id} \
  --context region=${region:-ap-northeast-1}

  cdk destroy \
  --context domain_name=${domain_name} \
  --context rails_master_key=${rails_master_key} \
  --context account_id=${account_id} \
  --context region=${region:-ap-northeast-1}
*/

func main() {
    defer jsii.Close()

    app := awscdk.NewApp(nil)

    domainName := app.Node().TryGetContext(jsii.String("domain_name")).(string)
    railsMasterKey := app.Node().TryGetContext(jsii.String("rails_master_key")).(string)

    NewRailsApiStack(app, "rails-api-stack", &RailsApiStackProps{
        StackProps: awscdk.StackProps{
            Env: env(app),
        },
        domainName: jsii.String(domainName),
        railsMasterKey: jsii.String(railsMasterKey),
    })

    app.Synth(nil)
}

func env(app awscdk.App) *awscdk.Environment {
  accountId := app.Node().TryGetContext(jsii.String("account_id")).(string)
  region := app.Node().TryGetContext(jsii.String("region")).(string)
  return &awscdk.Environment{
    Account: jsii.String(accountId),
    Region: jsii.String(region),
  }
}
