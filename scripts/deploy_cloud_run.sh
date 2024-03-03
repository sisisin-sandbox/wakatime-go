#!/usr/bin/env bash

set -eu -o pipefail

work_dir=$(cd "$(dirname "$0")" && pwd)
readonly work_dir

cd "$work_dir/.."
gcloud run jobs replace job.yaml
