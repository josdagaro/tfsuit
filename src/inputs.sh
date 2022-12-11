#!/bin/bash

# Initialize variables
help=0
debug=0
version=0
fail_on_not_compliant=0
dir=
config_json_path=
docs_link=
set_github_actions_outputs=0

for arg in "$@"; do
  case $arg in
  -h | --help)
    help=1
    ;;
  -D | --debug)
    debug=1
    ;;
  -v | --version)
    version=1
    ;;
  -d=* | --dir=*)
    dir="${arg#*=}"
    ;;
  -c=* | --config-json-path=*)
    config_json_path="${arg#*=}"
    ;;
  -f | --fail-on-not-compliant)
    fail_on_not_compliant=1
    ;;
  -dl=* | --docs-link=*)
    docs_link="${arg#*=}"
    ;;
  -gh | --github-actions)
    set_github_actions_outputs=1
    ;;
  esac
  shift
done

if [[ "$version" -eq 1 ]]; then
  helper::die "$(version)"
fi

if [[ "$help" -eq 1 ]]; then
  helper::die "$(version)"
fi

if [ -z "$dir" ] || [ -z "$config_json_path" ]; then
  usage::show
fi
