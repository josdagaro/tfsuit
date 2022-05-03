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
