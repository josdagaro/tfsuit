#!/usr/bin/env bash

# Initialize variables
help=0
debug=0
verbose=0
version=0
dir=
recursive=0
config_json_path=
extra_args=

getopt --test 2>/dev/null

if [[ $? -ne 4 ]]; then
  echo "GNU's enhanced getopt is required to run this script"
  echo "You can usually find this in the util-linux package"
  echo "On MacOS/OS X see homebrew's package: http://brewformulas.org/Gnu-getopt"
  echo "For anyone else, build from source: http://frodo.looijaard.name/project/getopt"
  exit 1
fi

SHORT=hDVvd:rc:
LONG=help,debug,verbose,version,dir:,recursive,config-json-path:

PARSED=$(getopt --options ${SHORT} \
  --longoptions ${LONG} \
  --name "$0" \
  -- "$@") # Pass all the args to this script to getopt

if [[ $? -ne 0 ]]; then
  # e.g. $? == 1
  #  then getopt has complained about wrong arguments to stdout
  exit 2
fi

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
