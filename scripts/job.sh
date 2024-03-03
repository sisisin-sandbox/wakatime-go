#!/usr/bin/env bash

set -eu -o pipefail

work_dir=$(cd "$(dirname "$0")" && pwd)
readonly work_dir
ts=$(TZ=JST-9 date "+%Y%m%d-%H%M%S")
readonly ts

gcloud batch jobs submit "wakatime-run-saver-$ts" --location=us-west1 --config "$work_dir/.out/jobConfig.json"
