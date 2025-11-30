resource "aws_s3_bucket" "logs" {
  bucket = "example"
}

data "aws_region" "current" {}
