variable "Bad-Name" {}

module "Alb-Bad" {
  source = "../"
  providers = {
    aws = aws.secondary
  }
}

resource "aws_s3_bucket" "LOGS-BUCKET" {}
