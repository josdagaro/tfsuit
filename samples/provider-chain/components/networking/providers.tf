terraform {
  required_providers {
    aws = {
      configuration_aliases = [
        aws.virginia,
        aws.ohio
      ]
    }
  }
}
