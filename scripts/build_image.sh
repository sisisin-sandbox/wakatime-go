#!/usr/bin/env bash

set -ef -o pipefail

work_dir=$(cd "$(dirname "$0")" && pwd)
readonly work_dir

cd "$work_dir/.."

TIMESTAMP=$(TZ=JST-9 date "+%Y%m%d-%H%M%S")
echo "$TIMESTAMP"
IMAGE_ID=sisisin/wakatime-go:$TIMESTAMP
echo "$IMAGE_ID"
docker build --platform linux/amd64 -t "$IMAGE_ID" .
docker login
docker push "$IMAGE_ID"

echo "Done."
echo "$IMAGE_ID"
