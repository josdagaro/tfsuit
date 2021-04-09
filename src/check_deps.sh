#!/usr/bin/env bash

if ! command -v jq &>/dev/null; then
  die "The command jq is not installed"
fi

if ! command -v gsht &>/dev/null; then
  die "The command gsht is not installed"
fi

[ ! -f "$config_json_path" ] && die "The configuration JSON file ${config_json_path} does not exist"
