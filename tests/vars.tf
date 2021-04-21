variable "vpc_cidr" {}
variable vpc_cidr_virginia {}

variable vpc_cidr1_virginia {
  type    = string
  default = ""
}

variable vpc_cidr2_virginia {
  type    = string
  default = ""
}

variable cidr_vpc_virginia {}
variable cidr_vpc {}
variable ec2_nlb_0_name_virginia {}
variable nlb_0_ec2_name {}
variable nlb_0_name {}
variable route53_domain {}



variable tags {}

variable "env" {
  description = "The environment where the infrastructure is setting up"
  type        = string
}
