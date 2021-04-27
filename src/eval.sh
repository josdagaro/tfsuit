#!/bin/bash

get_objects() {
  local objects
  local tf_files
  local code
  local tf_objects
  tf_files=$(find_tf_files "$dir")
  objects=""

  while read -r tf_file; do
    code=$(<"$tf_file")
    tf_objects=$(echo "$code" | grep -oP "$1")
    objects+="
    ${tf_objects}"
  done < <(echo "$tf_files")

  echo "$objects"
}

trim_objects() {
  local objects
  local trimed_objects
  local object_type_identifier_length
  trimed_objects=
  mapfile -t objects <<<"$1"
  object_type_identifier_length="${#2}"

  for elem in "${objects[@]}"; do
    trimed_objects+="${elem:${object_type_identifier_length}}
"
  done

  echo "$trimed_objects"
}

exclude_exact_ignored_objects() {
  local objects
  local ignored_objects
  mapfile -t objects <<<"$1"
  mapfile -t ignored_objects <<<"$2"

  for object in "${objects[@]}"; do
    is_ignored=0

    for ignored_object in "${ignored_objects[@]}"; do
      if [ "$object" == "$ignored_object" ]; then
        is_ignored=1
      fi
    done

    if [ "$is_ignored" != 1 ]; then
      echo "$object"
    fi
  done
}

eval() {
  local objects_naming_convention_match_pattern_beginning
  local objects_naming_convention_match_pattern
  local objects_naming_convention_ignore_match_pattern
  local objects_naming_convention_ignore_exact
  local objects_match_pattern_1
  local objects_match_pattern_2
  local objects
  local compliant_objects
  local compliant_objects_json_array
  local not_compliant_objects
  local not_compliant_objects_json_array
  local ignored_objects
  objects_naming_convention_match_pattern=$(jq <"$config_json_path" -r .${1}.naming_conventions.match_pattern)
  objects_naming_convention_ignore_match_pattern=$(jq <"$config_json_path" -r .${1}.naming_conventions.ignore.match_pattern)
  objects_naming_convention_ignore_exact=$(jq <"$config_json_path" -r .${1}.naming_conventions.ignore.exact)
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
    vars_match_pattern_1='^(?!#*$)([\s]+)?variable\s+([a-z0-9_]+|"[a-z0-9_]+")'
    vars_match_pattern_2='variable\s+([a-z0-9_]+|"[a-z0-9_]+")'
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
