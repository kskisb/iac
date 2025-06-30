package rds

import (
	"os"
	"rails_api/components/network"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type RDS struct {
	Instance awsrds.DatabaseInstance
}

func NewRDS(stack constructs.Construct, network *network.Network) *RDS {
	vpc := network.Vpc
	rdsSecurityGroup := network.RdsSecurityGroup
	resourceName := os.Getenv("RESOURCE_NAME")
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")

	// DB サブネットグループの作成
	subnetGroup := awsrds.NewSubnetGroup(stack, jsii.String(resourceName+"-subnet-group"), &awsrds.SubnetGroupProps{
		Description: jsii.String("Subnet group for RDS"),
		Vpc:         vpc,
		VpcSubnets: &awsec2.SubnetSelection{
			SubnetType: awsec2.SubnetType_PRIVATE_ISOLATED,
		},
	})

	// PostgreSQL パラメータグループの作成
	parameterGroup := awsrds.NewParameterGroup(stack, jsii.String(resourceName+"-parameter-group"), &awsrds.ParameterGroupProps{
		Engine: awsrds.DatabaseInstanceEngine_Postgres(&awsrds.PostgresInstanceEngineProps{
			Version: awsrds.PostgresEngineVersion_VER_16_4(),
		}),
		Parameters: &map[string]*string{
			"shared_preload_libraries": jsii.String("pg_stat_statements"),
		},
	})

	// RDSインスタンスの作成（無料利用枠対応）
	instance := awsrds.NewDatabaseInstance(stack, jsii.String(resourceName+"-database"), &awsrds.DatabaseInstanceProps{
		DatabaseName:       jsii.String("rails_api_production"),
		InstanceIdentifier: jsii.String("database-1"),
		Engine: awsrds.DatabaseInstanceEngine_Postgres(&awsrds.PostgresInstanceEngineProps{
			Version: awsrds.PostgresEngineVersion_VER_16_4(),
		}),
		InstanceType:   awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE3, awsec2.InstanceSize_MICRO), // t3.micro（無料利用枠）
		Vpc:            vpc,
		SecurityGroups: &[]awsec2.ISecurityGroup{rdsSecurityGroup},
		SubnetGroup:    subnetGroup,
		Credentials:    awsrds.Credentials_FromPassword(jsii.String(dbUsername), awscdk.SecretValue_UnsafePlainText(jsii.String(dbPassword))),

		// ストレージ設定（無料利用枠：20GB）
		AllocatedStorage:    jsii.Number(20),
		StorageType:         awsrds.StorageType_GP3,
		StorageEncrypted:    jsii.Bool(true),
		MaxAllocatedStorage: jsii.Number(1000), // 自動スケーリング上限

		// バックアップ設定
		BackupRetention:        awscdk.Duration_Days(jsii.Number(7)),
		DeleteAutomatedBackups: jsii.Bool(true),
		DeletionProtection:     jsii.Bool(false),

		// メンテナンス設定
		AutoMinorVersionUpgrade: jsii.Bool(true),

		// マルチAZ設定（無料利用枠のため無効）
		MultiAz: jsii.Bool(false),

		// パラメータグループ
		ParameterGroup: parameterGroup,

		// モニタリング設定
		MonitoringInterval:        awscdk.Duration_Seconds(jsii.Number(0)), // Performance Insights無効
		EnablePerformanceInsights: jsii.Bool(false),

		// ログ設定
		CloudwatchLogsExports: &[]*string{},

		// 削除時の設定
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	return &RDS{
		Instance: instance,
	}
}
