#!/bin/bash

set -eu

tfsuit() {
  (
    source helpers.sh
    source usage.sh
    source version.sh
    source inputs.sh
    source check_deps.sh
    source evaluator.sh

    # Initialization of variables for TF variables
    local compliant_vars
    local not_compliant_vars
    local vars_sum
    local vars_message
    # Initialization of variables for TF outputs
    local compliant_outputs
    local not_compliant_outputs
    local outputs_sum
    local outputs_message
    # Initialization of variables for TF modules
    local compliant_mods
    local not_compliant_mods
    local mods_sum
    local mods_message
    local error_exists
    local message
    vars_message=""
    outputs_message=""
    mods_message=""
    message="[ERROR]"
    error_exists=0
    vars_sum=$(evaluator::eval --context="vars" --context-full-name="variable" --obj-naming-convention-match-pattern-beginning="variable\s+" --obj-match-pattern-1='^(?!#*$)([\s]+)?variable\s+([a-z0-9_]+|"[a-z0-9_]+")' --obj-match-pattern-2='variable\s+([a-z0-9_]+|"[a-z0-9_]+")')
    compliant_vars=$(echo "$vars_sum" | jq -r .compliant)
    not_compliant_vars=$(echo "$vars_sum" | jq -r .not_compliant)
    echo "compliant vars:"
    echo "$compliant_vars" | jq
    echo "::set-output name=compliant_vars::$(echo "$compliant_vars" | jq -rc)"
    echo "not compliant vars:"
    echo "$not_compliant_vars" | jq
    echo "::set-output name=not_compliant_vars::$(echo "$not_compliant_vars" | jq -rc)"

    if [ "${not_compliant_vars}" != "[]" ]; then
      vars_message="There are vars that doesn't complaint."
      error_exists=1
    fi

    outputs_sum=$(evaluator::eval --context="outputs" --context-full-name="output" --obj-naming-convention-match-pattern-beginning="output\s+" --obj-match-pattern-1='^(?!#*$)([\s]+)?output\s+([a-z0-9_]+|"[a-z0-9_]+")' --obj-match-pattern-2='output\s+([a-z0-9_]+|"[a-z0-9_]+")')
    compliant_outputs=$(echo "$outputs_sum" | jq -r .compliant)
    not_compliant_outputs=$(echo "$outputs_sum" | jq -r .not_compliant)
    echo "compliant outputs:"
    echo "$compliant_outputs" | jq
    echo "::set-output name=compliant_outputs::$(echo "$compliant_outputs" | jq -rc)"
    echo "not compliant outputs:"
    echo "$not_compliant_outputs" | jq
    echo "::set-output name=not_compliant_outputs::$(echo "$not_compliant_outputs" | jq -rc)"

    if [ "${not_compliant_outputs}" != "[]" ]; then
      outputs_message="There are outputs that doesn't complaint."
      error_exists=1
    fi

    mods_sum=$(evaluator::eval --context="modules" --context-full-name="module" --obj-naming-convention-match-pattern-beginning="module\s+" --obj-match-pattern-1='^(?!#*$)([\s]+)?module\s+([a-z0-9_]+|"[a-z0-9_]+")' --obj-match-pattern-2='module\s+([a-z0-9_]+|"[a-z0-9_]+")')
    compliant_mods=$(echo "$mods_sum" | jq -r .compliant)
    not_compliant_mods=$(echo "$mods_sum" | jq -r .not_compliant)
    echo "compliant modules:"
    echo "$compliant_mods" | jq
    echo "::set-output name=compliant_modules::$(echo "$compliant_mods" | jq -rc)"
    echo "not compliant modules:"
    echo "$not_compliant_mods" | jq
    echo "::set-output name=not_compliant_modules::$(echo "$not_compliant_mods" | jq -rc)"

    if [ "${not_compliant_mods}" != "[]" ]; then
      mods_message="There are modules that doesn't complaint."
      error_exists=1
    fi

    message+="
      $vars_message
      $outputs_message
      $mods_message
    "

    if [ "$error_exists" -eq 1 ] && [ "$fail_on_not_compliant" -eq 1 ]; then
      if [ -n "$docs_link" ]; then
        message+=" Please, check the related docs: $docs_link"
      fi

      helper::die "$message" 1
    fi
  )
}

tfsuit "$@"
