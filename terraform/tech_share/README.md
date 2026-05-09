# Tech Share API - Terraform Infrastructure

Rails API（`tech_share_api`）を AWS 上に ECS Fargate でデプロイするための Terraform コード。

## 概要

**目標**：NAT ゲートウェイを使わず、VPC Endpoint 経由で秘密を安全に管理しながら、Rails API を ECS Fargate 上で運用できるインフラ構築。

**主要コンポーネント**
- **Compute**：ECS Fargate（`tech-share-api` コンテナ）、ALB（ロードバランサー）
- **Database**：RDS PostgreSQL 15（`db.t4g.micro`、20GB gp3）
- **Networking**：VPC（10.0.0.0/16）、Public/Private Subnet、VPC Endpoint（NAT 不使用）
- **Secrets Management**：AWS Secrets Manager（DB パスワード、Rails Master Key）
- **Logging**：CloudWatch Logs（retention 1日）
- **IAM**：最小権限ポリシー

## ファイル構成

```
iac/terraform/tech_share/
├── main.tf              # メインリソース定義
├── variables.tf         # 変数定義
├── terraform.tfvars     # 変数値（具体値）
├── provider.tf          # Terraform 設定・AWS プロバイダ
└── README.md            # このファイル
```

## 前提条件

- Terraform >= 1.5
- AWS CLI 設定済み（profile `default` または環境変数）
- ECR リポジトリ作成済み（`techshare-api`）
- Rails イメージを ECR に push 済み

## デプロイ手順

### 1. 変数の準備

[terraform.tfvars](terraform.tfvars) に必要な値が設定されているか確認：

```hcl
region   = "ap-northeast-1"
image_url = "879381244555.dkr.ecr.ap-northeast-1.amazonaws.com/techshare-api:latest"
```

### 2. Secrets Manager に Rails Master Key を設定

`main.tf` の `rails_master_key` 変数に環境変数で値を渡します（コードにハードコード不可）。

```bash
cd iac/terraform/tech_share
export TF_VAR_rails_master_key="<your-rails-master-key>"
```

### 3. Terraform 実行

```bash
# 初期化
terraform init

# プラン確認
terraform plan -out=plan.tfplan

# 適用
terraform apply "plan.tfplan"
```

**初回 apply にかかる時間**：約 10〜15 分（RDS、ALB、VPC Endpoint 作成）

### 4. リソース削除

テスト後にクリーンアップ：

```bash
terraform destroy -auto-approve
```

## アーキテクチャ

### ネットワーク

```
┌─────────────────────────────────────────────────────┐
│ VPC (10.0.0.0/16)                                   │
│                                                     │
│  ┌─────────────────┐      ┌─────────────────┐     │
│  │ Public Subnet-a │      │ Public Subnet-c │     │
│  │ (10.0.1.0/24)   │      │ (10.0.2.0/24)   │     │
│  └────────┬────────┘      └────────┬────────┘     │
│           │                        │               │
│    ┌──────▼────────────────────────▼──────┐        │
│    │ ALB (Load Balancer) - Port 80        │        │
│    └──────┬────────────────────────────────┘        │
│           │                                         │
│  ┌────────▼──────────────────┐                     │
│  │ Target Group (Port 3000)  │                     │
│  └────────┬──────────────────┘                     │
│           │                                         │
│  ┌────────▼────────────────────────────┐           │
│  │ ECS Service                         │           │
│  │ (Desired Count = 1)                 │           │
│  └────────┬────────────────────────────┘           │
│           │                                         │
│  ┌────────▼──────────────────────────────────┐     │
│  │ ECS Task Definition (Fargate)            │     │
│  │ - CPU: 256, Memory: 512 MB              │     │
│  │ - Container: tech-share-api (Port 3000) │     │
│  └────────┬──────────────────────────────────┘     │
│           │                                         │
│  ┌────────▼─────────────────────────────────────┐  │
│  │ Private Subnet-a & c                        │  │
│  │ ECS Task + RDS PostgreSQL                   │  │
│  │ - RDS Endpoint: rds-instance.region.rds     │  │
│  │ - Port: 5432                                │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
│  VPC Endpoints (Interface Type):                   │
│  - secretsmanager.ap-northeast-1                   │
│  - ecr.api.ap-northeast-1                          │
│  - ecr.dkr.ap-northeast-1                          │
│  - logs.ap-northeast-1                             │
│  - ssm.ap-northeast-1                              │
│  - s3 (Gateway Type)                               │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Secrets Manager

| シークレット名 | 用途 | 取得方法 |
|---|---|---|
| `tech-share-production-db-password` | RDS パスワード | `random_password` で自動生成 |
| `tech-share-production-rails-master-key` | Rails Master Key | `TF_VAR_rails_master_key` 環境変数で渡す |

両方ともタスク実行時に ECS が取得し、コンテナに環境変数として注入します。

## セキュリティ機能

### 1. Secrets Manager 統合

- DB パスワードとRails Master Key を平文でコード・state に保存しない
- ECS タスク実行時にランタイムメモリから取得

### 2. IAM ポリシー（最小権限）

**タスク実行ロール** (`ecs_task_execution_role`):
- `secretsmanager:GetSecretValue`（両シークレット ARN に限定）
- `logs:CreateLogStream`, `logs:PutLogEvents`（CloudWatch ログ）
- ECR イメージ pull

**タスクロール** (`ecs_task_role`):
- `secretsmanager:GetSecretValue`（DB パスワードのみ）
- アプリケーションが必要とするその他の権限

### 3. VPC エンドポイント

- NAT ゲートウェイ不使用でコスト削減
- プライベートサブネット内のタスクが AWS 管理サービスへアクセス可能
- 専用 SG で TCP/443（HTTPS）を許可

## 環境変数・設定

### ECS タスク定義で使用される環境変数

| 環境変数 | ソース | 用途 |
|---|---|---|
| `RAILS_ENV` | 静的 | `production` |
| `RAILS_LOG_TO_STDOUT` | 静的 | CloudWatch Logs 出力 |
| `DB_HOST` | RDS エンドポイント | データベース接続 |
| `DB_NAME` | `techshare` | データベース名 |
| `DB_USERNAME` | `postgres` | DB ユーザー |
| `DB_PASSWORD` | Secrets Manager | DB パスワード（実行時注入） |
| `RAILS_MASTER_KEY` | Secrets Manager | Rails credentials 復号化（実行時注入） |

## 動作確認

### ヘルスチェックエンドポイント

ALB は `GET /up` に対して HTTP 200 を期待します。

```bash
# ALB DNS を取得
aws elbv2 describe-load-balancers --region ap-northeast-1 \
  --query 'LoadBalancers[?LoadBalancerName==`tech-share-production-alb`].DNSName' \
  --output text

# ヘルスチェック確認
curl http://<ALB-DNS>/up
```

### ログ確認

```bash
# CloudWatch Logs を追跡
aws logs tail /ecs/tech-share-production --follow --region ap-northeast-1
```

### リソース状態確認

```bash
# ECS サービス
aws ecs describe-services \
  --cluster tech-share-production-cluster \
  --services tech-share-production-service \
  --region ap-northeast-1

# タスク一覧
aws ecs list-tasks \
  --cluster tech-share-production-cluster \
  --region ap-northeast-1

# RDS インスタンス
aws rds describe-db-instances \
  --db-instance-identifier tech-share-production-postgres \
  --region ap-northeast-1 \
  --query 'DBInstances[0].[DBInstanceIdentifier,DBInstanceStatus,Endpoint.Address]' \
  --output text
```

## 費用見積もり

短時間テスト（10 分程度）：約 $0.2〜$0.5  
月額（本番向け）：
- RDS（`db.t4g.micro`, 20GB）：約 $30
- ALB：約 $20
- ECS Fargate（1 タスク常時）：約 $15
- VPC Endpoint（5 個）：約 $40
- その他（CloudWatch, ECR など）：約 $10
- **合計概算**：約 $115 〜 $125 / 月

## 参考資料

- [Terraform AWS Provider ドキュメント](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [AWS ECS Fargate ベストプラクティス](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking.html)
- [AWS Secrets Manager 統合ガイド](https://docs.aws.amazon.com/secretsmanager/latest/userguide/)

---

**作成日**：2026-05-09
**Terraform Version**：>= 1.5
**AWS Provider Version**：~> 5.0
