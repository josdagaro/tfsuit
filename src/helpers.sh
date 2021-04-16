#!/bin/bash

die() {
  echo "$1"
  exit "${2:-0}"
}

convert_array_to_json_array() {
  local array
  local json_array
  mapfile -t array <<<"$1"
  json_array="["

  for elem in "${array[@]}"; do
    json_array="$json_array\"$elem\","
  done

  if [ "$json_array" != "[" ]; then
    json_array="${json_array::${#json_array}-1}"
  fi

  json_array+="]"
  echo "$json_array"
}

convert_json_array_to_array() {
  local json_array
  json_array="$1"

  for row in $(echo "$json_array" | jq -r '.[]'); do
    echo "$row"
  done
}

find_tf_files() {
  local dir
  local command_find
  local result
  dir="$1"
  command_find="#!/usr/bin/env bash
  find ${dir} "

  if [ -f ".tfsuitignore" ]; then
    command_find+="-type d \( "

    while read -r ignored; do
      command_find+=" -name '${ignored}' -o"
    done < <(cat .tfsuitignore)

    command_find="${command_find::${#command_find}-2}\) -prune -false -o "
  fi

  command_find+="-name '*.tf'"
  echo "$command_find" >/tmp/tfsuit_find.sh
  result=$(bash /tmp/tfsuit_find.sh)
  rm -f /tmp/tfsuit_find.sh
  echo "$result"
}
