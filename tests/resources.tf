resource "aws_acm_certificate" "foo" {
  domain_name       = "example.com"
  validation_method = "DNS"

  tags = {
    Environment = "test"
  }

  lifecycle {
    create_before_destroy = true
  }
}

            resource "aws_acm_certificate" "foo_correct" {
  domain_name       = "example.com"
  validation_method = "DNS"

  tags = {
    Environment = "test"
  }

  lifecycle {
    create_before_destroy = true
  }
}

    resource "aws_acm_certificate" "foo-incorrect" {
  domain_name       = "example.com"
  validation_method = "DNS"

  tags = {
    Environment = "test"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate" bar {
  domain_name       = "testing.example.com"
  validation_method = "EMAIL"

  validation_option {
    domain_name       = "testing.example.com"
    validation_domain = "example.com"
  }
}

resource aws_acm_certificate foo2 {
  domain_name       = "testing.example.com"
  validation_method = "EMAIL"

  validation_option {
    domain_name       = "testing.example.com"
    validation_domain = "example.com"
  }
}

            resource "aws_acm_certificate" bar2 {
  domain_name       = "testing.example.com"
  validation_method = "EMAIL"

  validation_option {
    domain_name       = "testing.example.com"
    validation_domain = "example.com"
  }
}


resource      "aws_acm_certificate" foobar3 {
  domain_name       = "testing.example.com"
  validation_method = "EMAIL"

  validation_option {
    domain_name       = "testing.example.com"
    validation_domain = "example.com"
  }
}


resource aws_acm_certificate "barfoo2" {
  domain_name       = "testing.example.com"
  validation_method = "EMAIL"

  validation_option {
    domain_name       = "testing.example.com"
    validation_domain = "example.com"
  }
}
