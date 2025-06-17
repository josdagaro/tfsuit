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

    # Inicialización de variables para Terraform variables
    local compliant_variables
    local not_compliant_variables
    local variables_summary
    local variables_message
    # Inicialización de variables para Terraform outputs
    local compliant_outputs
    local not_compliant_outputs
    local outputs_summary
    local outputs_message
    # Inicialización de variables para Terraform módulos
    local compliant_modules
    local not_compliant_modules
    local modules_summary
    local modules_message
    # Inicialización de variables para Terraform recursos
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

    # Análisis de variables de Terraform
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

    # Análisis de outputs de Terraform
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

    # Análisis de módulos de Terraform
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

    # Validación de las variables pasadas a los módulos
    echo "Evaluando argumentos de módulos para variables con sufijo de región..."

    grep -Poz 'module\s+"[a-zA-Z0-9_-]+"\s*\{[^}]+\}' *.tf | \
    while IFS= read -r -d '' module_block; do
      module_name=$(echo "$module_block" | grep -Po 'module\s+"\K[^"]+')
      
      # Extraer la región del nombre del módulo, e.g., _virginia
      region=$(echo "$module_name" | grep -Po '_\K[a-z0-9]+$')
      
      # Buscar los argumentos que usan variables
      echo "$module_block" | grep -Po '\s+\w+\s+=\s+var\.\w+' | while read -r line; do
        varname=$(echo "$line" | grep -Po 'var\.\K\w+')
        argname=$(echo "$line" | awk -F= '{print $1}' | tr -d ' ')

        # Si el argumento empieza con associate_ se espera que la variable tenga sufijo de región
        if [[ "$argname" == associate_* ]] && [[ "$varname" != *_$region ]]; then
          echo "[ERROR] En el módulo '$module_name': la variable '$varname' usada para '$argname' debería terminar en '_$region'"
          error_exists=1
        fi
      done
    done

    # Análisis de recursos de Terraform
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
    # Si falla, imprime el contenido crudo
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

    if [ "$error_exists" -eq 1 ] && [ "$fail_on_not_compliant" -eq 1 ]; then
      if [ -n "$docs_link" ]; then
        message+=" Please, check the related docs: $docs_link"
      fi

      helper::die "$message" 1
    fi
  )
}

tfsuit "$@"
