data "aws_caller_identity" "current" {}

// Get AWS AccountID associated to Observe tenant
data "observe_cloud_info" "current" {}

data "observe_workspace" "this" {
  name = "Default"
}

data "observe_datastream" "this" {
  workspace = data.observe_workspace.this.oid
  name      = "Default"
}

resource "aws_iam_role" "cloudwatch_role" {
  name_prefix = "observe-metrics"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = "sts:AssumeRole",
        Effect = "Allow",
        Principal = {
          AWS = [
            "arn:aws:iam::${data.observe_cloud_info.current.account_id}:root",
          ]
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = data.observe_datastream.this.id
          }
        }
      }
    ]
  })
}

resource "aws_iam_policy" "cloudwatch_policy" {
  name_prefix = "cloudwatch-policy"
  description = "Policy to allow CloudWatchMetrics poller"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = [
          "cloudwatch:GetMetricData",
          "cloudwatch:ListMetrics",
          "tag:GetResources",
        ],
        Effect   = "Allow",
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "attach_policy" {
  role       = aws_iam_role.cloudwatch_role.name
  policy_arn = aws_iam_policy.cloudwatch_policy.arn
}

resource "observe_poller" "this" {
  workspace = data.observe_workspace.this.oid
  name      = "Example Poller"
  interval  = "2m"

  datastream = data.observe_datastream.this.oid

  cloudwatch_metrics {
    region          = "us-west-2"
    assume_role_arn = aws_iam_role.cloudwatch_role.arn

    query {
      namespace = "AWS/EC2"
    }
  }
}
