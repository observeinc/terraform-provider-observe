# Enables publication of release artifacts to internal (publicly accessible) Terraform registry at terraform.observeinc.com.

resource "github_actions_variable" "aws_release_role" {
  repository = local.repository

  variable_name = "AWS_ROLE_ARN"
  value         = aws_iam_role.github_actions_release.arn
}

# https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services
resource "aws_iam_role" "github_actions_release" {
  name = "${local.repository}-gha-release"

  inline_policy {
    name   = "registry-write"
    policy = data.aws_iam_policy_document.registry_write.json
  }

  managed_policy_arns = []
  assume_role_policy  = data.aws_iam_policy_document.github_actions_assume_role.json

  tags = {
    Principal  = "GitHub Actions"
    Repository = "${local.organization}/${local.repository}"
  }
}

data "aws_s3_bucket" "terraform_registry" {
  bucket = "observeinc-terraform-registry"
}

data "aws_iam_policy_document" "registry_write" {
  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
    ]

    resources = ["${data.aws_s3_bucket.terraform_registry.arn}/*"]
  }

  statement {
    actions   = ["s3:ListBucket"]
    resources = [data.aws_s3_bucket.terraform_registry.arn]
  }
}

data "aws_iam_openid_connect_provider" "github_actions" {
  url = "https://token.actions.githubusercontent.com"
}

locals {
  oidc_claim_prefix = trimprefix("https://", data.aws_iam_openid_connect_provider.github_actions.url)
}

data "aws_iam_policy_document" "github_actions_assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github_actions.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_claim_prefix}:sub"
      values   = ["repo:${local.organization}/${local.repository}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_claim_prefix}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}
