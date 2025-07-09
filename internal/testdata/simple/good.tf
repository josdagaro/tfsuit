variable "vpc_cidr" {}
module "alb" { source = "../" }
resource "aws_s3_bucket" "logs" {}
