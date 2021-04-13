#!/bin/bash

get_vars() {
  local vars
  local tf_files
  local code
  local tf_vars
  tf_files=$(find_tf_files "$dir")
  vars=""

  while read -r tf_file; do
    code=$(cat "$tf_file")
    tf_vars=$(echo "$code" | grep -oP "$1")
    vars+="
    ${tf_vars}"
  done < <(echo "$tf_files")

  echo "$vars"
}

eval_vars() {
  local vars_naming_convention_match_pattern_beginning
  local vars_naming_convention_match_pattern
  local vars_match_pattern_1
  local vars_match_pattern_2
  local vars
  local compliant_vars
  local compliant_vars_json_array
  local not_compliant_vars
  local not_compliant_vars_json_array
  vars_naming_convention_match_pattern=$(cat "$config_json_path" | jq -r .vars.naming_conventions.match_pattern)

  if [ "$vars_naming_convention_match_pattern" != "null" -a ! -z "$vars_naming_convention_match_pattern" ]; then
    vars_naming_convention_match_pattern_beginning="variable\s+"
    vars_match_pattern_1="^(?!#*$)([\s]+)?variable\s+[a-z0-9_]+"
    vars_match_pattern_2="variable\s+[a-z0-9_]+"
    vars_naming_convention_match_pattern="${vars_naming_convention_match_pattern_beginning}${vars_naming_convention_match_pattern}"
    vars=$(get_vars "$vars_match_pattern_1")
    compliant_vars=$(echo "$vars" | grep -oE "$vars_naming_convention_match_pattern")
    not_compliant_vars=$(echo "$vars" | grep -ovE "$vars_naming_convention_match_pattern" | grep -oP "$vars_match_pattern_1" | grep -oE "$vars_match_pattern_2")
  else
    compliant_vars=""
    not_compliant_vars=""
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

  echo "{
    \"compliant\": ${compliant_vars_json_array},
    \"not_compliant\": ${not_compliant_vars_json_array}
  }"
}
