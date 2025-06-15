package network

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Network struct {
	Vpc              awsec2.IVpc
	AlbSecurityGroup awsec2.ISecurityGroup
	EcsSecurityGroup awsec2.ISecurityGroup
	Alb              awselasticloadbalancingv2.ApplicationLoadBalancer
	Listener1        awselasticloadbalancingv2.ApplicationListener
	Listener2        awselasticloadbalancingv2.ApplicationListener
	TargetGroup1     awselasticloadbalancingv2.ApplicationTargetGroup
	TargetGroup2     awselasticloadbalancingv2.ApplicationTargetGroup
}

func NewNetwork(stack constructs.Construct) *Network {
	resourceName := os.Getenv("RESOURCE_NAME")

	// VPCの作成
	vpc := awsec2.NewVpc(stack, jsii.String(resourceName+"-vpc"), &awsec2.VpcProps{
		VpcName:                      jsii.String(resourceName + "-vpc"),
		MaxAzs:                       jsii.Number(2),
		NatGateways:                  jsii.Number(0),
		RestrictDefaultSecurityGroup: jsii.Bool(false),
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
	albSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String(resourceName+"-sg-alb"), &awsec2.SecurityGroupProps{
		SecurityGroupName: jsii.String(resourceName + "-sg-alb"),
		Vpc:               vpc,
		AllowAllOutbound:  jsii.Bool(true),
	})
	albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(80)), jsii.String("http from anywhere"), jsii.Bool(false))
	albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(8080)), jsii.String("http(8080) from anywhere"), jsii.Bool(false))

	// sg for ECS
	ecsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String(resourceName+"-sg-ecs"), &awsec2.SecurityGroupProps{
		SecurityGroupName: jsii.String(resourceName + "-sg-ecs"),
		Vpc:               vpc,
		AllowAllOutbound:  jsii.Bool(true),
	})
	ecsSecurityGroup.AddIngressRule(albSecurityGroup, awsec2.Port_Tcp(jsii.Number(80)), jsii.String("http from anywhere"), jsii.Bool(false))
	ecsSecurityGroup.AddIngressRule(albSecurityGroup, awsec2.Port_Tcp(jsii.Number(8080)), jsii.String("http(8080) from anywhere"), jsii.Bool(false))

	// alb
	alb := awselasticloadbalancingv2.NewApplicationLoadBalancer(stack, jsii.String(resourceName+"-alb"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
		LoadBalancerName: jsii.String(resourceName + "-alb"),
		Vpc:              vpc,
		InternetFacing:   jsii.Bool(true),
		SecurityGroup:    albSecurityGroup,
		IpAddressType:    awselasticloadbalancingv2.IpAddressType_IPV4,
	})

	listener1 := alb.AddListener(jsii.String(resourceName+"-listener1"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:     jsii.Number(80),
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		Open:     jsii.Bool(true),
	})

	listener2 := alb.AddListener(jsii.String(resourceName+"-listener2"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:     jsii.Number(8080),
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		Open:     jsii.Bool(true),
	})

	// tg1
	targetGroup1 := awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String(resourceName+"-tg1"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{
		TargetGroupName: jsii.String(resourceName + "-tg1"),
		Vpc:             vpc,
		Port:            jsii.Number(80),
		Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		TargetType:      awselasticloadbalancingv2.TargetType_IP,
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
			Path:     jsii.String("/"),
			Interval: awscdk.Duration_Seconds(jsii.Number(300)),
			Timeout:  awscdk.Duration_Seconds(jsii.Number(120)),
		},
	})

	// tg2 (Blue/Greenデプロイ用)
	targetGroup2 := awselasticloadbalancingv2.NewApplicationTargetGroup(stack, jsii.String(resourceName+"-tg2"), &awselasticloadbalancingv2.ApplicationTargetGroupProps{
		TargetGroupName: jsii.String(resourceName + "-tg2"),
		Vpc:             vpc,
		Port:            jsii.Number(80),
		Protocol:        awselasticloadbalancingv2.ApplicationProtocol_HTTP,
		TargetType:      awselasticloadbalancingv2.TargetType_IP,
		HealthCheck: &awselasticloadbalancingv2.HealthCheck{
			Path:     jsii.String("/"),
			Interval: awscdk.Duration_Seconds(jsii.Number(300)),
			Timeout:  awscdk.Duration_Seconds(jsii.Number(120)),
		},
	})

	listener1.AddTargetGroups(jsii.String(resourceName+"-tg1"), &awselasticloadbalancingv2.AddApplicationTargetGroupsProps{
		TargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{targetGroup1},
	})

	listener2.AddTargetGroups(jsii.String(resourceName+"-tg2"), &awselasticloadbalancingv2.AddApplicationTargetGroupsProps{
		TargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{targetGroup2},
	})

	return &Network{
		Vpc:              vpc,
		AlbSecurityGroup: albSecurityGroup,
		EcsSecurityGroup: ecsSecurityGroup,
		Alb:              alb,
		Listener1:        listener1,
		Listener2:        listener2,
		TargetGroup1:     targetGroup1,
		TargetGroup2:     targetGroup2,
	}
}
