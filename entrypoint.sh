#!/bin/bash

set -eu

dir_flag=
config_json_path_flag=
fail_on_not_compliant_flag=
docs_link_flag=

# Actions pass inputs as $INPUT_<input name> environmet variables
[[ -n "$INPUT_DIR" ]] && dir_flag="-d=${INPUT_DIR}"
[[ -n "$INPUT_CONFIG_JSON_PATH" ]] && config_json_path_flag="-c=${INPUT_CONFIG_JSON_PATH}"
[[ -n "$INPUT_FAIL_ON_NOT_COMPLIANT" ]] && fail_on_not_compliant_flag="-f"
[[ -n "$INPUT_DOCS_LINK" ]] && docs_link_flag="-dl=${INPUT_DOCS_LINK}"

echo "inputs:
  ${dir_flag}
  ${config_json_path_flag}
  ${fail_on_not_compliant_flag}
  ${docs_link_flag}
"

tfsuit --version
tfsuit "$dir_flag" "$config_json_path_flag" "$fail_on_not_compliant_flag" "$docs_link_flag"
tfsuit_exit_code=$?
exit "$tfsuit_exit_code"
