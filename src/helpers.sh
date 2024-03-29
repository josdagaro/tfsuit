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

function helper::spin() {
  local pid=$1
  local spin='-\|/'
  local result=

  local i=0
  tput civis

  while kill -0 "$pid" 2>/dev/null; do
    local i=$(((i + 1) % ${#spin}))
    printf "%s" "${spin:$i:1}"
    echo -en "\033[1D"
    sleep .2
  done

  tput cnorm
  wait "$pid"
  return $?
}

function helper::save_sample() {
  local name="$1"
  local value="$2"

  if [ "$debug" -eq 1 ]; then
    echo "$value" | jq >"samples/$name"
  fi
}

function helper::get_json_elements_joined_by_comma() {
  local document="$1"
  local document_property="$2"
  local resources=""

  for row in $(echo "$document" | jq -r "$document_property | @base64"); do
    resource_element=$(echo "$row" | base64 --decode)
    # Escape double quotes
    resource_element=$(echo "$resource_element" | sed 's/"/\\"/g')

    if [ "$resources" == "" ]; then
      resources="\"$resource_element"\"
    else
      resources="$resources,\"$resource_element"\"
    fi
  done

  echo "$resources"
}
