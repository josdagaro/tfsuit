#!/bin/bash

get_vars() {
  local vars
  local tf_files
  local code
  local tf_vars
  tf_files=$(find_tf_files "$dir")
  vars=""

  while read -r tf_file; do
    code=$(<"$tf_file")
    tf_vars=$(echo "$code" | grep -oP "$1")
    vars+="
    ${tf_vars}"
  done < <(echo "$tf_files")

  echo "$vars"
}

trim_vars() {
  local vars
  local trimed_vars
  trimed_vars=
  mapfile -t vars <<<"$1"

  for elem in "${vars[@]}"; do
    trimed_vars+="${elem:9}
"
  done

  echo "$trimed_vars"
}

exclude_exact_ignored_vars() {
  local vars
  local ignored_vars
  mapfile -t vars <<<"$1"
  mapfile -t ignored_vars <<<"$2"

  for var in "${vars[@]}"; do
    is_ignored=0

    for ignored_var in "${ignored_vars[@]}"; do
      if [ "$var" == "$ignored_var" ]; then
        is_ignored=1
      fi
    done

    if [ "$is_ignored" != 1 ]; then
      echo "$var"
    fi
  done
}

eval_vars() {
  local vars_naming_convention_match_pattern_beginning
  local vars_naming_convention_match_pattern
  local vars_naming_convention_ignore_match_pattern
  local vars_naming_convention_ignore_exact
  local vars_match_pattern_1
  local vars_match_pattern_2
  local vars
  local compliant_vars
  local compliant_vars_json_array
  local not_compliant_vars
  local not_compliant_vars_json_array
  local ignored_vars
  vars_naming_convention_match_pattern=$(jq <"$config_json_path" -r .vars.naming_conventions.match_pattern)
  vars_naming_convention_ignore_match_pattern=$(jq <"$config_json_path" -r .vars.naming_conventions.ignore.match_pattern)
  vars_naming_convention_ignore_exact=$(jq <"$config_json_path" -r .vars.naming_conventions.ignore.exact)

  if [ "$vars_naming_convention_match_pattern" != "null" -a ! -z "$vars_naming_convention_match_pattern" ]; then
    vars_naming_convention_match_pattern_beginning="variable\s+"
    vars_match_pattern_1="^(?!#*$)([\s]+)?variable\s+[a-z0-9_]+"
    vars_match_pattern_2="variable\s+[a-z0-9_]+"
    vars_naming_convention_match_pattern="${vars_naming_convention_match_pattern_beginning}${vars_naming_convention_match_pattern}"
    vars=$(get_vars "$vars_match_pattern_1")
    compliant_vars=$(echo "$vars" | grep -oE "$vars_naming_convention_match_pattern")
    not_compliant_vars=$(echo "$vars" | grep -vE "$vars_naming_convention_match_pattern" | grep -oP "$vars_match_pattern_1" | grep -oE "$vars_match_pattern_2")
  else
    compliant_vars=""
    not_compliant_vars=""
  fi

  if [ "$vars_naming_convention_ignore_match_pattern" != "null" -a ! -z "$vars_naming_convention_ignore_match_pattern" ]; then
    # Do something...
    ignored_vars="[]"
  elif [ "$vars_naming_convention_ignore_exact" != "null" -a ! -z "$vars_naming_convention_ignore_exact" ]; then
    ignored_vars=$(convert_json_array_to_array "$vars_naming_convention_ignore_exact")
  else
    ignored_vars="[]"
  fi

  if [ -z "$compliant_vars" ]; then
    compliant_vars_json_array="[]"
  else
    compliant_vars=$(trim_vars "$compliant_vars")
    compliant_vars=$(exclude_exact_ignored_vars "$compliant_vars" "$ignored_vars")
    compliant_vars_json_array=$(convert_array_to_json_array "$compliant_vars")
  fi

  if [ -z "$not_compliant_vars" ]; then
    not_compliant_vars_json_array="[]"
  else
    not_compliant_vars=$(trim_vars "$not_compliant_vars")
    not_compliant_vars=$(exclude_exact_ignored_vars "$not_compliant_vars" "$ignored_vars")
    not_compliant_vars_json_array=$(convert_array_to_json_array "$not_compliant_vars")
  fi

  echo "{
    \"compliant\": ${compliant_vars_json_array},
    \"not_compliant\": ${not_compliant_vars_json_array}
  }"
}
