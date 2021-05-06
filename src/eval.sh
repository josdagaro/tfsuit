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
  local context
  local objects_naming_convention_match_pattern_beginning
  local objects_match_pattern_1
  local objects_match_pattern_2

  for arg in "$@"; do
    case $arg in
    --context=*)
      context="${arg#*=}"
      ;;
    --obj-naming-convention-match-pattern-beginning=*)
      objects_naming_convention_match_pattern_beginning="${arg#*=}"
      ;;
    --obj-match-pattern-1=*)
      objects_match_pattern_1="${arg#*=}"
      ;;
    --obj-match-pattern-2=*)
      objects_match_pattern_2="${arg#*=}"
      ;;
    esac
    shift
  done

  local objects_naming_convention_match_pattern
  local objects_naming_convention_ignore_match_pattern
  local objects_naming_convention_ignore_exact
  local objects
  local compliant_objects
  local compliant_objects_json_array
  local not_compliant_objects
  local not_compliant_objects_json_array
  local ignored_objects
  objects_naming_convention_match_pattern=$(jq <"$config_json_path" -r --arg ctx "$context" '.[$ctx].naming_conventions.match_pattern')
  objects_naming_convention_ignore_match_pattern=$(jq <"$config_json_path" -r --arg ctx "$context" '.[$ctx].naming_conventions.ignore.match_pattern')
  objects_naming_convention_ignore_exact=$(jq <"$config_json_path" -r --arg ctx "$context" '.[$ctx].naming_conventions.ignore.exact')

  if [ "$objects_naming_convention_match_pattern" != "null" -a ! -z "$objects_naming_convention_match_pattern" ]; then
    objects_naming_convention_match_pattern="${objects_naming_convention_match_pattern_beginning}${objects_naming_convention_match_pattern}"
    objects=$(get_objects "$objects_match_pattern_1")
    compliant_objects=$(echo "$objects" | grep -oE "$objects_naming_convention_match_pattern")
    not_compliant_objects=$(echo "$objects" | grep -vE "$objects_naming_convention_match_pattern" | grep -oP "$objects_match_pattern_1" | grep -oE "$objects_match_pattern_2")
  else
    compliant_objects=""
    not_compliant_objects=""
  fi

  if [ "$objects_naming_convention_ignore_match_pattern" != "null" -a ! -z "$objects_naming_convention_ignore_match_pattern" ]; then
    # Do something...
    ignored_objects="[]"
  elif [ "$objects_naming_convention_ignore_exact" != "null" -a ! -z "$objects_naming_convention_ignore_exact" ]; then
    ignored_objects=$(convert_json_array_to_array "$objects_naming_convention_ignore_exact")
  else
    ignored_objects="[]"
  fi

  if [ -z "$compliant_objects" ]; then
    compliant_objects_json_array="[]"
  else
    compliant_objects=$(trim_objects "$compliant_objects")
    compliant_objects=$(exclude_exact_ignored_objects "$compliant_objects" "$ignored_objects")
    compliant_objects_json_array=$(convert_array_to_json_array "$compliant_objects")
  fi

  if [ -z "$not_compliant_objects" ]; then
    not_compliant_objects_json_array="[]"
  else
    not_compliant_objects=$(trim_objects "$not_compliant_objects")
    not_compliant_objects=$(exclude_exact_ignored_objects "$not_compliant_objects" "$ignored_objects")
    not_compliant_objects_json_array=$(convert_array_to_json_array "$not_compliant_objects")
  fi

  echo "{
    \"compliant\": ${compliant_objects_json_array},
    \"not_compliant\": ${not_compliant_objects_json_array}
  }"
}
