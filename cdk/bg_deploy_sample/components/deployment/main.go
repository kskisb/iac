package deployment

import (
	"bg_deploy_sample/components/network"
	"bg_deploy_sample/components/service"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2/awscodebuild"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodedeploy"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodepipeline"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodepipelineactions"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Deployment struct {
	Application     awscodedeploy.EcsApplication
	DeploymentGroup awscodedeploy.EcsDeploymentGroup
	ServiceRole     awsiam.Role
	BuildProject    awscodebuild.PipelineProject
	Pipeline        awscodepipeline.Pipeline
}

func NewDeployment(stack constructs.Construct, network *network.Network, service *service.Service) *Deployment {
	resourceName := os.Getenv("RESOURCE_NAME")
	githubConnectionArn := os.Getenv("GITHUB_CONNECTION_ARN")
	repositoryOwner := os.Getenv("REPOSITORY_OWNER")
	repositoryName := os.Getenv("REPOSITORY_NAME")
	branchName := os.Getenv("BRANCH_NAME")

	// CodeDeploy設定
	codeDeployApp := awscodedeploy.NewEcsApplication(stack, jsii.String(resourceName+"-deployment"), &awscodedeploy.EcsApplicationProps{
		ApplicationName: jsii.String(resourceName + "-deployment"),
	})

	codeDeployRole := awsiam.NewRole(stack, jsii.String(resourceName+"-deploy-role"), &awsiam.RoleProps{
		RoleName:  jsii.String(resourceName + "-deploy-role"),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("codedeploy.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AWSCodeDeployRoleForECS")),
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonECS_FullAccess")),
		},
	})

	deploymentGroup := awscodedeploy.NewEcsDeploymentGroup(stack, jsii.String(resourceName+"-deployment-group"), &awscodedeploy.EcsDeploymentGroupProps{
		Application:         codeDeployApp,
		DeploymentGroupName: jsii.String(resourceName + "-deployment-group"),
		Service:             service.Service,
		BlueGreenDeploymentConfig: &awscodedeploy.EcsBlueGreenDeploymentConfig{
			BlueTargetGroup:  network.TargetGroup1,
			GreenTargetGroup: network.TargetGroup2,
			Listener:         network.Listener1,
			TestListener:     network.Listener2,
		},
		DeploymentConfig: awscodedeploy.EcsDeploymentConfig_LINEAR_10PERCENT_EVERY_1MINUTES(),
		Role:             codeDeployRole,
	})

	// CodeBuild設定
	codeBuildPolicy := awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: jsii.Strings(
			"logs:CreateLogGroup",
			"logs:CreateLogStream",
			"logs:PutLogEvents",
			"ssm:GetParameters",
		),
		Resources: jsii.Strings("*"),
		Effect:    awsiam.Effect_ALLOW,
	})

	codeBuildRole := awsiam.NewRole(stack, jsii.String(resourceName+"-codebuild-role"), &awsiam.RoleProps{
		RoleName:  jsii.String(resourceName + "-codebuild-role"),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("codebuild.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonEC2ContainerRegistryFullAccess")),
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonS3FullAccess")),
		},
	})

	codeBuildRole.AddToPolicy(codeBuildPolicy)

	source := awscodebuild.Source_GitHub(&awscodebuild.GitHubSourceProps{
		Owner:       jsii.String(repositoryOwner),
		Repo:        jsii.String(repositoryName),
		BranchOrRef: jsii.String(branchName),
	})

	codeBuildProject := awscodebuild.NewProject(stack, jsii.String(resourceName+"-codebuild-project"), &awscodebuild.ProjectProps{
		ProjectName: jsii.String(resourceName + "-codebuild-project"),
		Source:      source,
		Environment: &awscodebuild.BuildEnvironment{
			BuildImage:  awscodebuild.LinuxBuildImage_AMAZON_LINUX_2023_5(),
			ComputeType: awscodebuild.ComputeType_SMALL,
			Privileged:  jsii.Bool(true),
		},
		BuildSpec: awscodebuild.BuildSpec_FromSourceFilename(jsii.String("buildspec.yml")),
		Role:      codeBuildRole,
	})

	// CodePipeline設定
	artifactBucket := awss3.NewBucket(stack, jsii.String(resourceName+"-codepipeline-artifacts"), &awss3.BucketProps{
		BucketName: jsii.String(resourceName + "-codepipeline-artifacts"),
		// RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		// AutoDeleteObjects: jsii.Bool(true),
	})

	codePipelineRole := awsiam.NewRole(stack, jsii.String(resourceName+"-codepipeline-role"), &awsiam.RoleProps{
		RoleName:  jsii.String(resourceName + "-codepipeline-role"),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("codepipeline.amazonaws.com"), nil),
	})

	codePipeline := awscodepipeline.NewPipeline(stack, jsii.String(resourceName+"-codepipeline"), &awscodepipeline.PipelineProps{
		PipelineName:   jsii.String(resourceName + "-codepipeline"),
		ArtifactBucket: artifactBucket,
		Role:           codePipelineRole,
	})

	// Source
	sourceOutput := awscodepipeline.NewArtifact(jsii.String("SourceOutput"), nil)

	sourceAction := awscodepipelineactions.NewCodeStarConnectionsSourceAction(&awscodepipelineactions.CodeStarConnectionsSourceActionProps{
		ActionName:    jsii.String("SourceAction"),
		Owner:         jsii.String(repositoryOwner),
		Repo:          jsii.String(repositoryName),
		Branch:        jsii.String(branchName),
		ConnectionArn: jsii.String(githubConnectionArn),
		Output:        sourceOutput,
		Role:          codePipelineRole,
	})

	codePipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: jsii.String("Source"),
		Actions:   &[]awscodepipeline.IAction{sourceAction},
	})

	// Build
	buildOutput := awscodepipeline.NewArtifact(jsii.String("BuildOutput"), nil)

	buildAction := awscodepipelineactions.NewCodeBuildAction(&awscodepipelineactions.CodeBuildActionProps{
		ActionName: jsii.String("Build"),
		Project:    codeBuildProject,
		Input:      sourceOutput,
		Outputs:    &[]awscodepipeline.Artifact{buildOutput},
		Role:       codePipelineRole,
	})

	codePipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: jsii.String("Build"),
		Actions:   &[]awscodepipeline.IAction{buildAction},
	})

	// Approval
	approvalAction := awscodepipelineactions.NewManualApprovalAction(&awscodepipelineactions.ManualApprovalActionProps{
		ActionName: jsii.String("Approval"),
		Role:       codePipelineRole,
	})

	codePipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: jsii.String("Approval"),
		Actions:   &[]awscodepipeline.IAction{approvalAction},
	})

	// Deploy
	deployAction := awscodepipelineactions.NewCodeDeployEcsDeployAction(&awscodepipelineactions.CodeDeployEcsDeployActionProps{
		ActionName:                 jsii.String("Deploy"),
		DeploymentGroup:            deploymentGroup,
		AppSpecTemplateFile:        buildOutput.AtPath(jsii.String("appspec.yaml")),
		TaskDefinitionTemplateFile: buildOutput.AtPath(jsii.String("taskdef.json")),
		Role:                       codePipelineRole,
	})

	codePipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: jsii.String("Deploy"),
		Actions:   &[]awscodepipeline.IAction{deployAction},
	})

	return &Deployment{
		Application:     codeDeployApp,
		DeploymentGroup: deploymentGroup,
		ServiceRole:     codeDeployRole,
		BuildProject:    codeBuildProject,
	}
}
