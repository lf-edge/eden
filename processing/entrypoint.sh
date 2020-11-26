#!/bin/sh
set -e

exec 3>&1
exec 1>&2

do_help() {
cat <<__EOT__
Usage: docker run itmoeve/processing -i file -o file svg - to process output of perf script into svg
Usage: docker run itmoeve/processing -i file -o https://GIT_LOGIN:GIT_TOKEN@github.com/GIT_REPO -b branch git
__EOT__
  exit 0
}

do_svg() {
  ./stackcollapse-perf.pl <"$INPUT_DATA" >temp
  ./flamegraph.pl temp >"$OUTPUT_DATA"
}

do_git() {
  git clone -b "$BRANCH" "$OUTPUT_DATA" ./repo
  cd ./repo/"$BRANCH"
  cp "$INPUT_DATA" .
  git config --global user.email "eden_processing@example.com"
  git config --global user.name "PROCESSING"
  git add .
  git commit -m "processing-results"
  git push
}

# Lets' parse global options first
while true; do
  case "$1" in
  -i*) #shellcheck disable=SC2039
    INPUT_DATA="${1/-i/}"
    if [ -z "$INPUT_DATA" ]; then
      INPUT_DATA="$2"
      shift
    fi
    shift
    ;;
  -o*) #shellcheck disable=SC2039
    OUTPUT_DATA="${1/-o/}"
    if [ -z "$OUTPUT_DATA" ]; then
      OUTPUT_DATA="$2"
      shift
    fi
    shift
    ;;
  -b*) #shellcheck disable=SC2039
    BRANCH="${1/-b/}"
    if [ -z "$BRANCH" ]; then
      BRANCH="$2"
      shift
    fi
    shift
    ;;
  *)
    break
    ;;
  esac
done

# If we were not told to do anything, print help and exit with success
[ $# -eq 0 ] && do_help

ACTION="do_$1"

"$ACTION" "$@"
