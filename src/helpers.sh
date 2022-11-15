#!/bin/bash

helper::die() {
  echo "$1"
  exit "${2:-0}"
}

helper::convert_array_to_json_array() {
  local array
  local json_array
  mapfile -t array <<<"$1"
  json_array="["

  for elem in "${array[@]}"; do
    if [ -n "$elem" ]; then
      elem=$(echo "$elem" | sed -e 's/^[[:space:]]*//')
      elem="${elem//\"/\\\"}"
      json_array="$json_array\"$elem\","
    fi
  done

  if [ "$json_array" != "[" ]; then
    json_array="${json_array::${#json_array}-1}"
  fi

  json_array+="]"
  echo "$json_array"
}

helper::convert_json_array_to_array() {
  local json_array
  json_array="$1"

  for row in $(echo "$json_array" | jq -r '.[]'); do
    echo "$row"
  done
}

helper::find_tf_files() {
  local dir
  local command_find
  local result
  local tfsuitignore_file_name
  tfsuitignore_file_name=".tfsuitignore"
  dir="$1"
  command_find="#!/usr/bin/env bash
  find ${dir} "

  if [ -f "$tfsuitignore_file_name" ] && [ ! -s "$tfsuitignore_file_name" ]; then
    command_find+="-type d \( "

    while IFS= read -r line; do
      if [ -d "$line" ]; then
        command_find+=" -name '${line}' -o"
      elif [ -f "$line" ]; then # TODO: Add a grep to allow regular expressions like *.json
        command_find+=" -path '${line}' -o"
      else
        helper::die "Directory or file ${line} doesn't exists"
      fi
    done <"$tfsuitignore_file_name"

    command_find="${command_find::${#command_find}-2}\) -prune -false -o "
  fi

  command_find+="-name '*.tf'"
  echo "$command_find" >/tmp/tfsuit_find.sh
  result=$(bash /tmp/tfsuit_find.sh)
  rm -f /tmp/tfsuit_find.sh
  echo "$result"
}

# Inputs for loader (spin)
sp='/-\|'
printf ' '

helper::spin() {
  printf '\b%.1s' "$sp"
  sp=${sp#?}${sp%???}
}

helper::save_sample() {
  local name
  local value
  name="$1"
  value="$2"

  if [ "$debug" -eq 1 ]; then
    echo "$value" | jq >"samples/$name"
  fi
}

helper::convert_map_to_list_of_complaint_or_not_resources() {
  local compliant_resources
  local not_compliant_resources
  local resources
  local keys
  local found_compliant_resources
  local found_not_compliant_resources
  resources="$1"

  compliant_resources='['
  not_compliant_resources='['
  keys=$(echo "$resources" | jq 'keys[]')
  helper::save_sample "aws-resources-keys.txt" "$keys"

  while IFS= read -r key; do
    key=$(printf "%s\n" "$key" | sed -e "s/\"/\\\\\"/g")
    found_compliant_resources=$(echo "$resources" | jq ".$key.compliant")
    # TODO: The error could be here
    #found_compliant_resources=$(helper::convert_json_array_to_array "$found_compliant_resources")
    #echo "$found_compliant_resources" | jq >"samples/test/$key.txt"

    for row in $(echo "$found_compliant_resources" | jq -r '.[] | @base64'); do
      row=$(echo "$row" | base64 --decode)
      row=$(printf "%s\n" "$row" | sed -e "s/\"/\\\\\"/g")
      compliant_resources="${compliant_resources}\"resource $key $row\","
    done

    #if [ -n "$found_compliant_resources" ] && [ "$found_compliant_resources" != "" ] && [ "$found_compliant_resources" != "[]" ]; then
    #  while IFS= read -r found_compliant_resource; do
    #    compliant_resources="${compliant_resources}resource $key $found_compliant_resource,"
    #  done <<<"$found_compliant_resources"
    #fi

    found_not_compliant_resources=$(echo "$resources" | jq ".$key.not_compliant")
    # TODO: The error could be here
    #found_not_compliant_resources=$(helper::convert_json_array_to_array "$found_not_compliant_resources")
    #echo "$found_not_compliant_resources" | jq >"samples/test/$key-not.txt"

    for row in $(echo "$found_not_compliant_resources" | jq -r '.[] | @base64'); do
      row=$(echo "$row" | base64 --decode)
      row=$(printf "%s\n" "$row" | sed -e "s/\"/\\\\\"/g")
      not_compliant_resources="${not_compliant_resources}\"resource $key $row\","
    done

    #while IFS= read -r found_not_compliant_resource; do
    #  not_compliant_resources="${not_compliant_resources}resource $key $found_not_compliant_resource,"
    #done <<<"$found_not_compliant_resources"
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
