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
    # Initialization of variables for Terraform resources
    local compliant_resources
    local not_compliant_resources
    local resources_summary
    local resources_message
    local remove_double_quotes_for_resources
    local error_exists
    local message
    variables_message=""
    outputs_message=""
    modules_message=""
    resources_message=""
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

    # Terraform resources analysis
    echo "processing resources..."
    remove_double_quotes_for_resources=$(jq <"$config_json_path" -r '.resources.naming_conventions.remove_double_quotes')
    echo "remove double quotes for resources: $remove_double_quotes_for_resources"

    resources_summary=$(finder::run \
      --context="resources" \
      --context-full-name="resource" \
      --obj-naming-convention-match-pattern-beginning='resource\s+"([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")"\s+' \
      --obj-match-pattern-1='^(?!#*$)([\s]+)?resource\s+"([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")"\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
      --obj-match-pattern-2='resource\s+"([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")"\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

    resource_without_double_quotes_summary=$(finder::run \
      --context="resources" \
      --context-full-name="resource" \
      --obj-naming-convention-match-pattern-beginning='resource\s+([a-zA-Z0-9_-]+)\s+' \
      --obj-match-pattern-1='^(?!#*$)([\s]+)?resource\s+([a-zA-Z0-9_-]+)\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")' \
      --obj-match-pattern-2='resource\s+([a-zA-Z0-9_-]+)\s+([a-zA-Z0-9_-]+|"[a-zA-Z0-9_-]+")')

    if [ "$remove_double_quotes_for_resources" == "true" ]; then
      compliant_resources="[$(helper::get_json_elements_joined_by_comma "$resource_without_double_quotes_summary" .compliant[])]"
      not_compliant_resources="[$(helper::get_json_elements_joined_by_comma "$resource_without_double_quotes_summary" .not_compliant[])"
      resources_summary_compliant=$(helper::get_json_elements_joined_by_comma "$resources_summary" .compliant[])

      if [ "$resources_summary_compliant" != "" ]; then
        if [ "$not_compliant_resources" != "[" ]; then
          not_compliant_resources="$not_compliant_resources,$resources_summary_compliant"
        else
          not_compliant_resources="[$resources_summary_compliant"
        fi
      fi

      resources_summary_not_compliant=$(helper::get_json_elements_joined_by_comma "$resources_summary" .not_compliant[])

      if [ "$resources_summary_not_compliant" != "" ]; then
        if [ "$not_compliant_resources" != "[" ]; then
          not_compliant_resources="$not_compliant_resources,$resources_summary_not_compliant"
        else
          not_compliant_resources="[$resources_summary_not_compliant"
        fi
      fi
    else
      compliant_resources="[$(helper::get_json_elements_joined_by_comma "$resources_summary" .compliant[])]"
      not_compliant_resources="[$(helper::get_json_elements_joined_by_comma "$resources_summary" .not_compliant[])"
      resource_without_double_quotes_summary_compliant=$(helper::get_json_elements_joined_by_comma "$resource_without_double_quotes_summary" .compliant[])

      if [ "$resource_without_double_quotes_summary_compliant" != "" ]; then
        if [ "$not_compliant_resources" != "[" ]; then
          not_compliant_resources="$not_compliant_resources,$resource_without_double_quotes_summary_compliant"
        else
          not_compliant_resources="[$resource_without_double_quotes_summary_compliant"
        fi
      fi

      resource_without_double_quotes_summary_not_compliant=$(helper::get_json_elements_joined_by_comma "$resource_without_double_quotes_summary" .not_compliant[])

      if [ "$resource_without_double_quotes_summary_not_compliant" != "" ]; then
        if [ "$not_compliant_resources" != "[" ]; then
          not_compliant_resources="$not_compliant_resources,$resource_without_double_quotes_summary_not_compliant"
        else
          not_compliant_resources="[$resource_without_double_quotes_summary_not_compliant"
        fi
      fi
    fi

    not_compliant_resources="$not_compliant_resources]"
    echo "compliant resources:"
    echo "$compliant_resources" | jq
    github::set_output "compliant_resources" "$(echo "$compliant_resources" | jq -rc)"
    echo "not compliant resources:"
    # If it fails, just print the content as raw
    echo "$not_compliant_resources" | jq || echo "$not_compliant_resources"
    github::set_output "not_compliant_resources" "$(echo "$not_compliant_resources" | jq -rc)"

    if [ "${not_compliant_resources}" != "[]" ]; then
      resources_message="There are resources that doesn't complaint."
      error_exists=1
    fi

    message+="
      $variables_message
      $outputs_message
      $modules_message
      $resources_message
    "
    echo "validating required variables for resources and modules..."

    required_vars_json=$(jq -r '.required_variables_per_object' "$config_json_path")

    # Validar recursos
    jq -r '.resources // {} | to_entries[] | "\(.key)=\(.value | @csv)"' <<< "$required_vars_json" | while IFS='=' read -r resource_type required_vars_csv; do
      required_vars=($(echo "$required_vars_csv" | tr -d '"' | tr ',' ' '))
      echo "Validating $resource_type for required variables: ${required_vars[*]}"
      
      for file in $(find . -name "*.tf"); do
        if grep -q "resource\s\+\"$resource_type\"" "$file"; then
          for var in "${required_vars[@]}"; do
            if ! grep -q "$var\s*=" "$file"; then
              echo "[ERROR] Resource '$resource_type' in file '$file' is missing required variable '$var'"
              github::set_output "missing_resource_variables" "[ERROR] Resource '$resource_type' in file '$file' is missing required variable '$var'"
              error_exists=1
            fi
          done
        fi
      done
    done

    # Validar módulos
    declare -A required_simple_variables
    declare -A required_complex_variables

    while IFS='=' read -r module_pattern required_vars_csv; do
      required_simple_variables["$module_pattern"]=$(echo "$required_vars_csv" | tr -d '"' | tr ',' ' ')
    done < <(
      jq -r '
        .modules
        | to_entries[]
        | "\(.key)=\(.value | map(select(type == "string")) | @csv)"
        ' <<< "$required_vars_json"
    )

    while IFS='=' read -r module_pattern nested_json; do
      required_complex_variables["$module_pattern"]="$nested_json"
    done < <(
      jq -r '
        .modules
        | to_entries[]
        | select(.value | any(type == "object"))
        | "\(.key)=\(.value | map(select(type == "object")) | @json)"
        ' <<< "$required_vars_json"
    )

    for module_pattern in "${!required_simple_variables[@]}"; do
      required_vars=(${required_simple_variables[$module_pattern]})
      
      echo "Validating modules matching '$module_pattern' for required variable references: ${required_vars[*]}"
      
      for file in $(find . -name "*.tf"); do
        module_blocks=$(awk '/module\s+"/{flag=1} /}/{flag=0} flag' "$file")
        module_names=$(grep -oP 'module\s+"[^"]+"' "$file" | cut -d'"' -f2)
        echo "$module_blocks"
        echo "$module_names"
        for mod in $module_names; do
          echo "segundo for"
          if [[ "$mod" =~ $module_pattern ]]; then
            echo "valida modulo"
            block=$(awk "/module[[:space:]]+\"$mod\"[[:space:]]*{/,/^}/" "$file")

            for var in "${required_vars[@]}"; do
              pattern="var.${var}"
              echo "tercer for"
              if ! [[ "$block" =~ $pattern ]]; then
                echo "entra al if de falla"
                echo "[ERROR] Module '$mod' in file '$file' is missing reference to '${var}'"
                github::set_output "missing_module_variables" "[ERROR] Module '$mod' in file '$file' is missing reference to '${var}'"
                error_exists=1
              fi
              echo "sale if de falla"
            done
          fi
        done
      done
    done

    for module_pattern in "${!required_complex_variables[@]}"; do
      nested_json="${required_complex_variables[$module_pattern]}"
      echo "Validating modules matching '$module_pattern' for required complex variable references: $nested_json"

      for file in $(find . -name "*.tf"); do
        module_blocks=$(awk '/module\s+"/{flag=1} /}/{flag=0} flag' "$file")
        module_names=$(grep -oP 'module\s+"[^"]+"' "$file" | cut -d'"' -f2)

        for mod in $module_names; do
          if [[ "$mod" =~ $module_pattern ]]; then
            block=$(awk "/module[[:space:]]+\"$mod\"[[:space:]]*{/,/^}/" "$file")

            jq -c '.[]' <<< "$nested_json" | while read -r complex_entry; do
              jq -r 'to_entries[] | "\(.key)=\(.value[])"' <<< "$complex_entry" | while IFS='=' read -r outer_attr varname; do
                outer_attr_actual=$(echo "$block" | grep -oP '^\s*\K\w+(?=\s*=\s*\[)')
                varnames_actual=$(echo "$block" | grep -oP 'var\.\w+' | sed 's/var\.//' | sort -u)

                if ! [[ "$outer_attr" =~ $outer_attr_actual ]]; then
                  echo "[ERROR] Module '$mod' in file '$file' is missing '$outer_attr'"
                  github::set_output "missing_module_variables" "[ERROR] Module '$mod' in file '$file' is missing '$outer_attr'"
                  error_exists=1
                fi

                varname=$(echo "$varname" | tr -d '\r' | xargs)

                if ! echo "$varnames_actual" | grep -Eq "$varname"; then
                  echo "[ERROR] Variable '$outer_attr' of the module '$mod' is missing '$varname'"
                  github::set_output "missing_module_variables" "[ERROR] Variable '$outer_attr' of the module '$mod' is missing '$varname'"
                  error_exists=1
                fi
              done
            done
          fi
        done
      done
    done

    if [ "$error_exists" -eq 1 ] && [ "$fail_on_not_compliant" -eq 1 ]; then
      if [ -n "$docs_link" ]; then
        message+=" Please, check the related docs: $docs_link"
      fi

      helper::die "$message" 1
    fi
  )
}

tfsuit "$@"
