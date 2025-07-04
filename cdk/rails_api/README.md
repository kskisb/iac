# Twitter Clone インフラ

このインフラは [このリポジトリ](https://github.com/kskisb/rails_api) と連動しています。次の流れでセットアップできます。

## 概要

このプロジェクトは、Twitter CloneアプリケーションのためのAWSインフラをAWS CDK（Go）で構築します。

### インフラ構成

- **VPC**: プライベートおよびパブリックサブネット
- **ALB**: Application Load Balancer（SSL/TLS対応）
- **ECS Fargate**: コンテナ化されたRails APIアプリケーション
- **RDS**: PostgreSQLデータベース
- **ECR**: Dockerイメージレジストリ
- **Route53**: DNS管理
- **Certificate Manager**: SSL証明書

### アーキテクチャ

```
Internet
    |
  ALB (HTTPS)
    |
ECS Fargate (Rails API)
    |
RDS (PostgreSQL)
```

## 前提条件

- AWS CLI設定済み
- AWS CDK CLI インストール済み
- Go 1.18以降
- Docker
- 独自ドメイン（Route53で管理）
- [このリポジトリ](https://github.com/kskisb/rails_api) にて ECR リポジトリを作成済み

## 環境変数設定

プロジェクトルートに `.env` ファイルを作成してください：

```bash
ACCOUNT_ID=123456789
REGION=*********
RESOURCE_NAME=*********
REPOSITORY_NAME=*********
RAILS_MASTER_KEY=*********
DOMAIN_NAME=*********
DB_HOST=*********
DB_USERNAME=*********
DB_PASSWORD=*********
DB_PORT=5432
ALLOWED_ORIGIN=*********
```

## セットアップ

### 1. リポジトリのクローン

```bash
git clone https://github.com/kskisb/iac.git
cd rails_api
```

### 2. 依存関係のインストール

```bash
go mod download
```

### 3. 環境変数の設定

`rials_api/` に `.env` を作成し、適切な値を設定する。

### 4. CDKのブートストラップ（初回のみ）

```bash
cdk bootstrap
```

### 5. インフラのデプロイ

```bash
cdk deploy
```

## プロジェクト構造

```
rails_api/
├── rails_api.go          # メインのCDKスタック定義
├── components/           # インフラコンポーネント
│   ├── network/         # VPC、ALB、セキュリティグループ
│   ├── rds/            # RDSデータベース
│   └── service/        # ECS Fargate サービス
├── cdk.json            # CDK設定
├── go.mod              # Go モジュール定義
└── .env               # 環境変数（要作成）
```

## CDKコマンド

### デプロイ

```bash
# スタック全体をデプロイ
cdk deploy

# 変更差分を確認
cdk diff

# リソースの一覧表示
cdk list
```

### 削除

```bash
# スタック全体を削除
cdk destroy
```