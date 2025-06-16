package network

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
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
	domainName := os.Getenv("DOMAIN_NAME")

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

	vpc.AddInterfaceEndpoint(jsii.String("com.amazonaws.ap-northeast-1.logs"), &awsec2.InterfaceVpcEndpointOptions{
		Service: awsec2.InterfaceVpcEndpointAwsService_CLOUDWATCH_LOGS(),
	})

	// sg for ALB
	albSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String(resourceName+"-sg-alb"), &awsec2.SecurityGroupProps{
		SecurityGroupName: jsii.String(resourceName + "-sg-alb"),
		Vpc:               vpc,
		AllowAllOutbound:  jsii.Bool(true),
	})
	albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(80)), jsii.String("http from anywhere"), jsii.Bool(false))
	albSecurityGroup.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(443)), jsii.String("https from anywhere"), jsii.Bool(false))

	// sg for ECS
	ecsSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String(resourceName+"-sg-ecs"), &awsec2.SecurityGroupProps{
		SecurityGroupName: jsii.String(resourceName + "-sg-ecs"),
		Vpc:               vpc,
		AllowAllOutbound:  jsii.Bool(true),
	})
	ecsSecurityGroup.AddIngressRule(albSecurityGroup, awsec2.Port_Tcp(jsii.Number(3000)), jsii.String("http from alb to rails"), jsii.Bool(false))

	// alb
	alb := awselasticloadbalancingv2.NewApplicationLoadBalancer(stack, jsii.String(resourceName+"-alb"), &awselasticloadbalancingv2.ApplicationLoadBalancerProps{
		LoadBalancerName: jsii.String(resourceName + "-alb"),
		Vpc:              vpc,
		InternetFacing:   jsii.Bool(true),
		SecurityGroup:    albSecurityGroup,
		IpAddressType:    awselasticloadbalancingv2.IpAddressType_IPV4,
	})

	// ALBのDNS名を取得
	hostedZone := awsroute53.HostedZone_FromLookup(stack, jsii.String("HostedZone"), &awsroute53.HostedZoneProviderProps{
		DomainName: jsii.String(domainName),
	})

	// ACM証明書の作成
	certificate := awscertificatemanager.NewCertificate(stack, jsii.String(resourceName+"-certificate"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String(domainName),
		Validation: awscertificatemanager.CertificateValidation_FromDns(hostedZone),
	})

	listener1 := alb.AddListener(jsii.String(resourceName+"-listener2"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:     jsii.Number(443),
		Protocol: awselasticloadbalancingv2.ApplicationProtocol_HTTPS,
		Certificates: &[]awselasticloadbalancingv2.IListenerCertificate{
			awselasticloadbalancingv2.ListenerCertificate_FromCertificateManager(certificate),
		},
	})

	listener2 := alb.AddListener(jsii.String(resourceName+"-listener1"), &awselasticloadbalancingv2.BaseApplicationListenerProps{
		Port:     jsii.Number(80),
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
			Path:     jsii.String("/up"),
			Interval: awscdk.Duration_Seconds(jsii.Number(60)),
			Timeout:  awscdk.Duration_Seconds(jsii.Number(30)),
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
			Path:     jsii.String("/up"),
			Interval: awscdk.Duration_Seconds(jsii.Number(60)),
			Timeout:  awscdk.Duration_Seconds(jsii.Number(30)),
		},
	})

	listener1.AddTargetGroups(jsii.String(resourceName+"-tg1"), &awselasticloadbalancingv2.AddApplicationTargetGroupsProps{
		TargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{targetGroup1},
	})

	listener2.AddTargetGroups(jsii.String(resourceName+"-tg2"), &awselasticloadbalancingv2.AddApplicationTargetGroupsProps{
		TargetGroups: &[]awselasticloadbalancingv2.IApplicationTargetGroup{targetGroup2},
	})

	// ALBのDNS名をRoute53に登録
	awsroute53.NewARecord(stack, jsii.String("ARecord"), &awsroute53.ARecordProps{
		Zone:   hostedZone,
		Target: awsroute53.RecordTarget_FromAlias(awsroute53targets.NewLoadBalancerTarget(alb, nil)),
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
