#!/bin/bash

# Actions pass inputs as $INPUT_<input name> environmet variables
[[ ! -z "$INPUT_DIR" ]] && dir_flag="-dir=${INPUT_DIR}"
[[ ! -z "$INPUT_CONFIG_JSON_PATH" ]] && config_json_path_flag="-c=${INPUT_CONFIG_JSON_PATH}"
[[ ! -z "$INPUT_FAIL_ON_NOT_COMPLIANT" ]] && fail_on_not_compliant_flag="-f"

echo "flags:
  ${dir_flag}
  ${config_json_path_flag}
  ${fail_on_not_compliant_flag}
"