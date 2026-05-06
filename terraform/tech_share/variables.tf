variable "region" {
  description = "AWS region"
  type        = string
  default     = "ap-northeast-1"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "tech-share"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "image_url" {
  description = "ECR image URL"
  type        = string
  # デフォルトなし — terraform.tfvars で値を渡す
}