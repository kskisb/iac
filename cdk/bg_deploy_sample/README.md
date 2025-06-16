# Welcome to your CDK Go project!

This is a blank project for CDK development with Go.

The `cdk.json` file tells the CDK toolkit how to execute your app.

## Useful commands

 * `cdk deploy`      deploy this stack to your default AWS account/region
 * `cdk diff`        compare deployed stack with current state
 * `cdk synth`       emits the synthesized CloudFormation template
 * `go test`         run unit tests


## ブルーグリーンデプロイ設定

`bg_deploy_sample/` に `.env`ファイルを作成し、次の内容を配置します。

```
ACCOUNT_ID=${ACCOUNT_ID} # 12桁のアカウントID
REGION=${REGION} # 例: ap-northeast-1
RESOURCE_NAME=${RESOURCE_NAME} # 値は任意
REPOSITORY_NAME=${REPOSITORY_NAME} # ECRのリポジトリ名
GITHUB_CONNECTION_ARN=${GITHUB_CONNECTION_ARN} # CodePipelineで接続を作成し、そのARNを設定する
GITHUB_REPOSITORY_OWNER=${GITHUB_REPOSITORY_OWNER} # アカウント名
GITHUB_REPOSITORY_NAME=${GITHUB_REPOSITORY_NAME} # GitHubのリポジトリ名
GITHUB_BRANCH_NAME=${GITHUB_BRANCH_NAME} # ブルーグリーンデプロイをするブランチ名
```