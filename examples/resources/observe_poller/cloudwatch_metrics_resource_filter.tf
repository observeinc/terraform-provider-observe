cloudwatch_metrics {
  region          = "us-west-2"
  assume_role_arn = aws_iam_role.cloudwatch_role.arn

  query {
    namespace = "AWS/EC2"
    resource_filter {
      tag_filter {
        key    = "Environment"
        values = ["Staging", "Production"]
      }
    }
  }
}
