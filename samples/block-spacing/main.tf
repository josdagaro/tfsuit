variable "foo" { type = string }
variable "bar" {
  type = number
}
resource "aws_s3_bucket" "logs" {
  bucket = "sample"
}

module "demo" {}
