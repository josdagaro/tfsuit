variable "Bad-Name" {}
module "Alb-Bad"     { source = "../" }
resource "aws_s3_bucket" "LOGS-BUCKET" {}
