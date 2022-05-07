#!/bin/bash

# It goes to the page https://raw.githubusercontent.com/hashicorp/terraform-provider-aws/main/internal/provider/provider.go ...
# for getting the file in format RAW and extract the available resources using a regular expressions
providers::aws::get_all_resources() {
  local raw_file_url
  local raw_content
  local match_pattern_1
  local match_pattern_2
  local resources
  # Using this regular expression for catching all the lines betweek the brackets:
  # DataSourcesMap: map[string]*schema.Resource{ ... }
  match_pattern_1='(?<=DataSourcesMap: map\[string\]\*schema\.Resource\{)\n((?:.*?|\n)*?)(?=\})'
  # Using this regular expression for catching all resources names...
  # which are between double quotes: "aws_acm_certificate"
  match_pattern_2='\"[a-z0-9_]+\"'
  raw_file_url="https://raw.githubusercontent.com/hashicorp/terraform-provider-aws/main/internal/provider/provider.go"
  raw_content=$(curl -s "$raw_file_url")
  raw_content=$(echo "$raw_content" | pcregrep -oM "$match_pattern_1" | pcregrep -oM "$match_pattern_2")
  # Remove all tabulations in each line
  resources=$(echo "$raw_content" | sed 's/\t//g')
  echo "$resources"
}
