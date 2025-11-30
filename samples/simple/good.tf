variable "vpc_cidr" {}

module "alb" {
  source = "../"
  providers = {
    aws = aws.primary
  }
}

resource "aws_s3_bucket" "logs" {}
