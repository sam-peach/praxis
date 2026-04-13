# GitHub Actions OIDC — lets GitHub Actions assume an AWS role without storing
# long-lived credentials as secrets.

data "aws_iam_openid_connect_provider" "github" {
  # If the OIDC provider doesn't exist yet in your account, create it once with:
  #   aws iam create-open-id-connect-provider \
  #     --url https://token.actions.githubusercontent.com \
  #     --client-id-list sts.amazonaws.com \
  #     --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1
  # IAM is global, so this is shared across regions — if charity_ai already
  # created it, no action is needed here.
  url = "https://token.actions.githubusercontent.com"
}

resource "aws_iam_role" "github_deploy" {
  name = "${var.app_name}-github-deploy"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = data.aws_iam_openid_connect_provider.github.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
        StringLike = {
          # Restrict to the main branch of this repo only
          "token.actions.githubusercontent.com:sub" = "repo:${var.github_repo}:ref:refs/heads/main"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "github_deploy" {
  name = "${var.app_name}-github-deploy"
  role = aws_iam_role.github_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ECRAuth"
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken"]
        Resource = "*"
      },
      {
        Sid    = "ECRPush"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:CompleteLayerUpload",
          "ecr:InitiateLayerUpload",
          "ecr:PutImage",
          "ecr:UploadLayerPart",
          "ecr:BatchGetImage",
          "ecr:GetDownloadUrlForLayer",
        ]
        Resource = aws_ecr_repository.app.arn
      },
      {
        Sid    = "AppRunnerDeploy"
        Effect = "Allow"
        Action = [
          "apprunner:StartDeployment",
          "apprunner:DescribeService",
        ]
        Resource = aws_apprunner_service.app.arn
      },
    ]
  })
}
