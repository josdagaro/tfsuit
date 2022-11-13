#!/bin/bash

set -eu

tfsuit() {
  (
    source helpers.sh
    source usage.sh
    source version.sh
    source inputs.sh
    source check_deps.sh
    source finder.sh
    source github.sh
    source providers/aws.sh

    # Initialization of variables for Terraform variables
    local compliant_variables
    local not_compliant_variables
    local variables_summary
    local variables_message
    # Initialization of variables for Terraform outputs
    local compliant_outputs
    local not_compliant_outputs
    local outputs_summary
    local outputs_message
    # Initialization of variables for Terraform modules
    local compliant_modules
    local not_compliant_modules
    local modules_summary
    local modules_message
    # Initialization of variables for Terraform AWS resources
    local compliant_aws_resources
    local not_compliant_aws_resources
    local aws_resources_summary
    local aws_resources_without_double_quotes_summary
    local aws_resources_message
    local aws_resources
    local remove_double_quotes_for_aws_resources
    local error_exists
    local message
    variables_message=""
    outputs_message=""
    modules_message=""
    aws_resources_message=""
    message="[ERROR]"
    error_exists=0

    # Terraform variables analysis
    echo "Processing variables..."

    variables_summary=$(finder::run \
      --context="vars" \
      --context-full-name="variable" \
      --obj-naming-convention-match-pattern-beginning="variable\s+" \
      --obj-match-pattern-1='^(?!#*$)([\s]+)?variable\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
      --obj-match-pattern-2='variable\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

    compliant_variables=$(echo "$variables_summary" | jq -r .compliant)
    not_compliant_variables=$(echo "$variables_summary" | jq -r .not_compliant)
    echo "compliant vars:"
    echo "$compliant_variables" | jq
    github::set_output "compliant_variables" "$(echo "$compliant_variables" | jq -rc)"
    echo "not compliant vars:"
    echo "$not_compliant_variables" | jq
    github::set_output "not_compliant_variables" "$(echo "$not_compliant_variables" | jq -rc)"

    if [ "${not_compliant_variables}" != "[]" ]; then
      variables_message="There are vars that doesn't complaint."
      error_exists=1
    fi

    # Terraform outputs analysis
    echo "processing outputs..."

    outputs_summary=$(finder::run \
      --context="outputs" \
      --context-full-name="output" \
      --obj-naming-convention-match-pattern-beginning="output\s+" \
      --obj-match-pattern-1='^(?!#*$)([\s]+)?output\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
      --obj-match-pattern-2='output\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

    compliant_outputs=$(echo "$outputs_summary" | jq -r .compliant)
    not_compliant_outputs=$(echo "$outputs_summary" | jq -r .not_compliant)
    echo "compliant outputs:"
    echo "$compliant_outputs" | jq
    github::set_output "compliant_outputs" "$(echo "$compliant_outputs" | jq -rc)"
    echo "not compliant outputs:"
    echo "$not_compliant_outputs" | jq
    github::set_output "not_compliant_outputs" "$(echo "$not_compliant_outputs" | jq -rc)"

    if [ "${not_compliant_outputs}" != "[]" ]; then
      outputs_message="There are outputs that doesn't complaint."
      error_exists=1
    fi

    # Terraform modules analysis
    echo "processing modules..."

    modules_summary=$(finder::run \
      --context="modules" \
      --context-full-name="module" \
      --obj-naming-convention-match-pattern-beginning="module\s+" \
      --obj-match-pattern-1='^(?!#*$)([\s]+)?module\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
      --obj-match-pattern-2='module\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

    compliant_modules=$(echo "$modules_summary" | jq -r .compliant)
    not_compliant_modules=$(echo "$modules_summary" | jq -r .not_compliant)
    echo "compliant modules:"
    echo "$compliant_modules" | jq
    github::set_output "compliant_modules" "$(echo "$compliant_modules" | jq -rc)"
    echo "not compliant modules:"
    echo "$not_compliant_modules" | jq
    github::set_output "not_compliant_modules" "$(echo "$not_compliant_modules" | jq -rc)"

    if [ "${not_compliant_modules}" != "[]" ]; then
      modules_message="There are modules that doesn't complaint."
      error_exists=1
    fi

    # Terraform aws resources analysis
    echo "processing AWS resources..."
    remove_double_quotes_for_aws_resources=$(jq <"$config_json_path" -r '.aws_resources.naming_conventions.remove_double_quotes')
    echo "remove double quotes for AWS resources: $remove_double_quotes_for_aws_resources"
    aws_resources=$(providers::aws::get_all_resources)
    aws_resources_summary='{'
    aws_resources_without_double_quotes_summary='{'

    while IFS= read -r aws_resource; do
      helper::spin
      local aws_resource_naming_convention_match_pattern_beginning
      local aws_resource_summary
      local aws_resource_without_double_quotes
      aws_resource_without_double_quotes=$(printf "%s\n" "$aws_resource" | sed -e "s/\\\"//g")

      # If the resource has double quotes in its name, they will be escaped...
      # E.g: "aws_acm_certificate" => \"aws_acm_certificate\"
      aws_resource_naming_convention_match_pattern_beginning=$(printf "%s\n" "$aws_resource" | sed -e "s/\"/\\\\\"/g")

      aws_resource_summary=$(finder::run \
        --context="aws_resources" \
        --context-full-name="resource $aws_resource" \
        --obj-naming-convention-match-pattern-beginning='resource\s+('"$aws_resource_naming_convention_match_pattern_beginning"')\s+' \
        --obj-match-pattern-1='^(?!#*$)([\s]+)?resource\s+('"$aws_resource_naming_convention_match_pattern_beginning"')\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
        --obj-match-pattern-2='resource\s+('"$aws_resource_naming_convention_match_pattern_beginning"')\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

      # TODO: run this process in parallel with the previous one for optimizing the execution time...
      # or, using a conditional based on the value of the variable $remove_double_quotes_for_aws_resources...
      # for performing this find or the previous one
      aws_resource_without_double_quotes_summary=$(finder::run \
        --context="aws_resources" \
        --context-full-name="resource $aws_resource_without_double_quotes" \
        --obj-naming-convention-match-pattern-beginning='resource\s+('"$aws_resource_without_double_quotes"')\s+' \
        --obj-match-pattern-1='^(?!#*$)([\s]+)?resource\s+('"$aws_resource_without_double_quotes"')\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
        --obj-match-pattern-2='resource\s+('"$aws_resource_without_double_quotes"')\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

      if [ "$aws_resources_summary" == '{' ]; then
        aws_resources_summary="$aws_resources_summary$aws_resource: $aws_resource_summary"
      else
        aws_resources_summary="$aws_resources_summary,$aws_resource: $aws_resource_summary"
      fi

      if [ "$aws_resources_without_double_quotes_summary" == '{' ]; then
        aws_resources_without_double_quotes_summary="$aws_resources_without_double_quotes_summary$aws_resource: $aws_resource_without_double_quotes_summary"
      else
        aws_resources_without_double_quotes_summary="$aws_resources_without_double_quotes_summary,$aws_resource: $aws_resource_without_double_quotes_summary"
      fi
    done <<<"$aws_resources"

    printf '\nAWS resources processed\n'
    aws_resources_summary="$aws_resources_summary}"
    aws_resources_without_double_quotes_summary="$aws_resources_without_double_quotes_summary}"
    helper::save_sample "aws-resources-summary.json" "$aws_resources_summary"
    helper::save_sample "aws-resources-without-double-quotes-summary.json" "$aws_resources_without_double_quotes_summary"

    # If the variable $remove_double_quotes_for_aws_resources is equal to "true"...
    # then the variable $aws_resources_summary is overwrited regarding the second find
    if [ "$remove_double_quotes_for_aws_resources" == "true" ]; then
      aws_resources_summary="$aws_resources_without_double_quotes_summary"
    fi

    printf '\nstructuring AWS resources...\n'
    aws_resources_summary=$(helper::convert_map_to_list_of_complaint_or_not_resources "$aws_resources_summary")
    printf '\nAWS resources ordered\n'
    compliant_aws_resources=$(echo "$aws_resources_summary" | jq -r .compliant)
    not_compliant_aws_resources=$(echo "$aws_resources_summary" | jq -r .not_compliant)
    echo "compliant aws resources:"
    echo "$compliant_aws_resources" | jq
    github::set_output "compliant_aws_resources" "$(echo "$compliant_aws_resources" | jq -rc)"
    echo "not compliant aws resources:"
    echo "$not_compliant_aws_resources" | jq
    github::set_output "not_compliant_aws_resources" "$(echo "$not_compliant_aws_resources" | jq -rc)"

    if [ "${not_compliant_aws_resources}" != "[]" ]; then
      aws_resources_message="There are aws resources that doesn't complaint."
      error_exists=1
    fi

    message+="
      $variables_message
      $outputs_message
      $modules_message
      $aws_resources_message
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
