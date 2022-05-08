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

resource "aws_api_gateway_account" "demo" {
  cloudwatch_role_arn = aws_iam_role.cloudwatch.arn
}

resource "aws_iam_role" "cloudwatch" {
  name = "api_gateway_cloudwatch_global"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "cloudwatch" {
  name = "default"
  role = aws_iam_role.cloudwatch.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:DescribeLogGroups",
                "logs:DescribeLogStreams",
                "logs:PutLogEvents",
                "logs:GetLogEvents",
                "logs:FilterLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}

resource "aws_codedeploy_app" example {
  compute_platform = "Lambda"
  name             = "example"
}

resource aws_codedeploy_app example2 {
  compute_platform = "Lambda"
  name             = "example"
}

resource aws_codedeploy_app "example3" {
  compute_platform = "Lambda"
  name             = "example"
}
