#!/usr/bin/env bash

set -eu -o pipefail

gcloud run jobs execute wakatime-downloader --region=us-west1 --args='--target-date=2024-02-29'
