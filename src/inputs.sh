#!/bin/bash

set -eu

# Initialize variables
help=0
debug=0
verbose=0
version=0
dir=
recursive=0
config_json_path=
extra_args=("${dummy_arg}") # Because set -u does not allow undefined variables to be used

echo "All pre-getopt arguments: $*"
getopt --test >/dev/null

if [[ $? -ne 4 ]]; then
  echo "I'm sorry, 'getopt --test' failed in this environment"
  exit 1
fi

SHORT=hDVvd:rc:
LONG=help,debug,verbose,version,dir:,recursive,config-json-path:

PARSED=$(getopt --options ${SHORT} \
  --longoptions ${LONG} \
  --name "$0" \
  -- "$@") # Pass all the args to this script to getopt

eval set -- "${PARSED}"

while [[ $# -gt 0 ]]; do
  case "$1" in
  -h | --help)
    help=1
    ;;
  -D | --debug)
    debug=1
    ;;
  -V | --verbose)
    verbose=1
    ;;
  -v | --version)
    version=1
    ;;
  -d | --dir)
    dir="$2"
    shift
    ;;
  -r | --recursive)
    recursive=1
    shift
    ;;
  -c | --config-json-path)
    config_json_path="$2"
    shift
    ;;
  --)
    shift
    extra_args=("$@")
    break
    ;;
  esac
  shift
done

extra_args=("${extra_args[@]/${dummy_arg}/}")
