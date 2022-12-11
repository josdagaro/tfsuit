#!/bin/bash

# In this script we validate the required dependencies for tfsuit...
# so, that means the needed commands and resources like required files

if ! command -v jq &>/dev/null; then
  helper::die "The command jq is not installed"
fi

if ! command -v gsht &>/dev/null; then
  helper::die "The command gsht is not installed"
fi

if ! command -v curl &>/dev/null; then
  helper::die "The command curl is not installed"
fi

if ! command -v pcregrep &>/dev/null; then
  helper::die "The command pcregrep is not installed"
fi

if [ ! -f "$config_json_path" ]; then
  helper::die "The configuration JSON file ${config_json_path} does not exist"
fi
