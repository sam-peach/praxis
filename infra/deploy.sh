#!/usr/bin/env bash
# Usage: ./infra/deploy.sh
# Builds the Docker image, pushes it to ECR, and triggers an App Runner redeployment.
# Run `terraform apply` first to create the infrastructure, then use this script for updates.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_NAME="sme-prototype"

REGION=$(terraform -chdir="$SCRIPT_DIR" output -raw region)
ECR_URL=$(terraform -chdir="$SCRIPT_DIR" output -raw ecr_url)
ACCOUNT="${ECR_URL%%.*}"

echo "==> Authenticating with ECR ($REGION)"
aws ecr get-login-password --region "$REGION" \
  | docker login --username AWS --password-stdin "$ACCOUNT.dkr.ecr.$REGION.amazonaws.com"

echo "==> Building image (linux/amd64 for App Runner)"
docker buildx build --platform linux/amd64 -t "$APP_NAME:amd64" --load "$REPO_ROOT"

echo "==> Pushing to ECR"
docker tag "$APP_NAME:amd64" "$ECR_URL:latest"
docker push "$ECR_URL:latest"

echo "==> Triggering App Runner deployment"
SERVICE_ARN=$(aws apprunner list-services --region "$REGION" \
  --query "ServiceSummaryList[?ServiceName=='$APP_NAME'].ServiceArn" \
  --output text)
aws apprunner start-deployment --service-arn "$SERVICE_ARN" --region "$REGION" > /dev/null

echo ""
echo "Deployment triggered. App URL:"
terraform -chdir="$SCRIPT_DIR" output -raw app_url
echo ""
echo "Check status: aws apprunner describe-service --service-arn $SERVICE_ARN --region $REGION --query 'Service.Status' --output text"
