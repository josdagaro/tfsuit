resource "aws_iam_role" "app" {
  name = "app-role"
}

data "aws_region" "current" {}
