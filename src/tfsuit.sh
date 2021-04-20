#!/bin/bash

set -eu

tfsuit() {
  (
    source helpers.sh
    source usage.sh
    source version.sh
    source inputs.sh
    source check_deps.sh
    source eval_vars.sh

    local compliant_vars
    local not_compliant_vars
    local vars_sum
    local vars_message
    vars_sum=$(eval_vars)
    compliant_vars=$(echo "$vars_sum" | jq -r .compliant)
    not_compliant_vars=$(echo "$vars_sum" | jq -r .not_compliant)
    echo "compliant vars:"
    echo "$compliant_vars" | jq
    echo "::set-output name=compliant_vars::$(echo ${compliant_vars} | jq -rc)"
    echo "not compliant vars:"
    echo "$not_compliant_vars" | jq
    echo "::set-output name=not_compliant_vars::$(echo ${not_compliant_vars} | jq -rc)"

    if [ "${not_compliant_vars}" != "[]" -a "$fail_on_not_compliant" -eq 1 ]; then
      vars_message="[ERROR] There are vars that doesn't complaint."

      if [ ! -z "$docs_link" ]; then
        vars_message+=" Please, check the related docs: ${docs_link}"
      fi

      die "$vars_message" 1
    fi
  )
}

tfsuit "$@"
