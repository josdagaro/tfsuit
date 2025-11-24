module "backend" {
  source = "./modules/backend"
  providers = {
    aws = aws.virginia
  }
}
