#!/usr/bin/env bash

die() {
  echo "$1"
  exit "${2:-0}"
}

convert_array_to_json_array() {
  local array
  local json_array
  array="$1"
  json_array="["

  for elem in "${array[@]}"; do
    json_array="$json_array\"$elem\","
  done

  json_array="${json_array::-1}]"
  return "$json_array"
}
