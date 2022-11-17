#!/bin/bash

providers::convert_map_to_list_of_complaint_or_not_resources() {
  local compliant_resources
  local not_compliant_resources
  local resources
  local keys
  local found_compliant_resources
  local found_not_compliant_resources
  local key_with_escaped_double_quotes
  resources="$1"

  compliant_resources='['
  not_compliant_resources='['
  keys=$(echo "$resources" | jq 'keys[]')
  helper::save_sample "aws-resources-keys.txt" "$keys"

  while IFS= read -r key; do
    key_with_escaped_double_quotes=$(printf "%s\n" "$key" | sed -e "s/\"/\\\\\"/g")
    key=$(printf "%s\n" "$key" | sed -e "s/\\\"//g")
    found_compliant_resources=$(echo "$resources" | jq ".$key.compliant")

    for row in $(echo "$found_compliant_resources" | jq -r '.[] | @base64'); do
      row=$(echo "$row" | base64 --decode)
      row=$(printf "%s\n" "$row" | sed -e "s/\"/\\\\\"/g")
      compliant_resources="${compliant_resources}\"resource $key_with_escaped_double_quotes $row\","
    done

    found_not_compliant_resources=$(echo "$resources" | jq ".$key.not_compliant")

    for row in $(echo "$found_not_compliant_resources" | jq -r '.[] | @base64'); do
      row=$(echo "$row" | base64 --decode)
      row=$(printf "%s\n" "$row" | sed -e "s/\"/\\\\\"/g")
      not_compliant_resources="${not_compliant_resources}\"resource $key_with_escaped_double_quotes $row\","
    done
  done <<<"$keys"

  compliant_resources=${compliant_resources::-1}
  compliant_resources="$compliant_resources]"
  not_compliant_resources=${not_compliant_resources::-1}
  not_compliant_resources="$not_compliant_resources]"
  helper::save_sample "aws-compliant-resources.json" "$compliant_resources"
  helper::save_sample "aws-not-compliant-resources.json" "$not_compliant_resources"

  echo "{
    \"compliant\": $compliant_resources,
    \"not_compliant\": $not_compliant_resources
  }"
}
