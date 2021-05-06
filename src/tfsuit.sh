#!/bin/bash

set -eu

tfsuit() {
  (
    source helpers.sh
    source usage.sh
    source version.sh
    source inputs.sh
    source check_deps.sh
    source eval.sh

    local compliant_vars
    local not_compliant_vars
    local vars_sum
    local vars_message
    local compliant_outputs
    local not_compliant_outputs
    local outputs_sum
    local outputs_message
    local error_exists
    local message
    message="[ERROR]"
    error_exists=0
    vars_sum=$(eval --context="vars" --context-full-name="variable" --obj-naming-convention-match-pattern-beginning="variable\s+" --obj-match-pattern-1='^(?!#*$)([\s]+)?variable\s+([a-z0-9_]+|"[a-z0-9_]+")' --obj-match-pattern-2='variable\s+([a-z0-9_]+|"[a-z0-9_]+")')
    compliant_vars=$(echo "$vars_sum" | jq -r .compliant)
    not_compliant_vars=$(echo "$vars_sum" | jq -r .not_compliant)
    echo "compliant vars:"
    echo "$compliant_vars" | jq
    echo "::set-output name=compliant_vars::$(echo ${compliant_vars} | jq -rc)"
    echo "not compliant vars:"
    echo "$not_compliant_vars" | jq
    echo "::set-output name=not_compliant_vars::$(echo ${not_compliant_vars} | jq -rc)"

    if [ "${not_compliant_vars}" != "[]" ]; then
      vars_message="There are vars that doesn't complaint."
      error_exists=1
    fi

    outputs_sum=$(eval --context="outputs" --context-full-name="output" --obj-naming-convention-match-pattern-beginning="output\s+" --obj-match-pattern-1='^(?!#*$)([\s]+)?output\s+([a-z0-9_]+|"[a-z0-9_]+")' --obj-match-pattern-2='output\s+([a-z0-9_]+|"[a-z0-9_]+")')
    compliant_outputs=$(echo "$outputs_sum" | jq -r .compliant)
    not_compliant_outputs=$(echo "$outputs_sum" | jq -r .not_compliant)
    echo "compliant outputs:"
    echo "$compliant_outputs" | jq
    echo "::set-output name=compliant_outputs::$(echo ${compliant_outputs} | jq -rc)"
    echo "not compliant outputs:"
    echo "$not_compliant_outputs" | jq
    echo "::set-output name=not_compliant_outputs::$(echo ${not_compliant_outputs} | jq -rc)"

    if [ "${not_compliant_outputs}" != "[]" ]; then
      outputs_message="There are outputs that doesn't complaint."
      error_exists=1
    fi

    message+="
      $vars_message
      $outputs_message
    "

    if [ "$fail_on_not_compliant" -eq 1 ]; then
      if [ ! -z "$docs_link" ]; then
        message+=" Please, check the related docs: ${docs_link}"
      fi

      die "$message" 1
    fi
  )
}

tfsuit "$@"
