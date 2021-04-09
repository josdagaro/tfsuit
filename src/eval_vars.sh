#!/bin/bash

set -eu

read_vars() {
  local vars
  $dir
  return "$vars"
}

eval_vars() {
  local vars_naming_convention_match_pattern_beginning
  local vars_naming_convention_match_pattern
  local vars_match_pattern
  local vars
  local compliant_vars
  local compliant_vars_json_array
  local not_compliant_vars
  local not_compliant_vars_json_array
  local vars_sum
  config_vars_naming_convention_match_pattern=$(cat "$config_json_path" | jq -r .vars.naming_conventions.match_pattern)

  if [ "$vars_naming_convention_match_pattern" != "null" ]; then
    vars_naming_convention_match_pattern_beginning="variable\s+"
    vars_match_pattern="variable\s+[a-z0-9_]+"
    vars_naming_convention_match_pattern="${vars_naming_convention_match_pattern_beginning}${vars_naming_convention_match_pattern}"
    vars=$(read_vars)
    compliant_vars=$(echo "$vars" | grep -oE "$vars_naming_convention_match_pattern")
    not_compliant_vars=$(echo "$vars" | grep -ovE "$vars_naming_convention_match_pattern" | grep -oE "$vars_match_pattern")
  fi

  if [ -z "$compliant_vars" ]; then
    compliant_vars_json_array="[]"
  else
    compliant_vars_json_array=$(convert_array_to_json_array "$compliant_vars")
  fi

  if [ -z "$not_compliant_vars" ]; then
    not_compliant_vars_json_array="[]"
  else
    not_compliant_vars_json_array=$(convert_array_to_json_array "$not_compliant_vars")
  fi

  return "{
    \"compliant\": $(echo ${compliant_vars_json_array}),
    \"not_compliant\": $(echo ${not_compliant_vars}),
  }"
}
