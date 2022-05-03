#!/bin/bash

if ! command -v jq &>/dev/null; then
  helper::die "The command jq is not installed"
fi

if ! command -v gsht &>/dev/null; then
  helper::die "The command gsht is not installed"
fi

if [ ! -f "$config_json_path" ]; then
  helper::die "The configuration JSON file ${config_json_path} does not exist"
fi
