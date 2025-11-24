module "services" {
  source = "./ecs"
}

resource "aws_s3_bucket" "logs" {
  bucket = "example"
}
