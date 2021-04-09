#!/bin/bash

set -eu

tfsuit() {
  (
    source check-deps.sh
    source helpers.sh
    source version.sh
    source inputs.sh

    if [[ "$version" -eq 1 ]]; then
      die "$(version)"
    fi

    if [[ "$help" -eq 1 ]]; then
      die "$(version)"
    fi

    local config_var_naming_convention_beginning_of_match_pattern
    local config_var_naming_convention_match_pattern
    config_var_naming_convention_match_pattern=$(cat "$config_json_path" | jq .vars.naming_conventions.match_pattern)

    if [ "$config_var_naming_convention_match_pattern" != "null" ]; then
      config_var_naming_convention_beginning_of_match_pattern="variable\\s+"
      config_var_naming_convention_match_pattern="${config_var_naming_convention_beginning_of_match_pattern}${config_var_naming_convention_match_pattern}"
      
    elif [ "$" ]; then
    fi
  )
}

tfsuit "$@"
