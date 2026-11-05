#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Authenticate exactly as promotion does, but never upload or modify GCS objects.
set -euo pipefail
umask 077
CHECK_DIR=$(mktemp -d "${WORKSPACE}/gcp-read-XXXXXX")
trap 'rm -rf "${CHECK_DIR}"' EXIT
export CLOUDSDK_CONFIG="${CHECK_DIR}/gcloud"
get_secret gcp-bucket-write-agents-io-service-key > "${CHECK_DIR}/key.json"
gcloud auth activate-service-account --key-file "${CHECK_DIR}/key.json"
INDEX=gs://agents.instana.io/helm/index.yaml
gsutil stat "${INDEX}" >/dev/null
gsutil cp "${INDEX}" "${CHECK_DIR}/index.yaml"
python3 - "${CHECK_DIR}/index.yaml" <<'PY'
from pathlib import Path
import sys
import yaml

index = yaml.safe_load(Path(sys.argv[1]).read_text())
if not isinstance(index, dict) or index.get("apiVersion") != "v1" or not index.get("entries", {}).get("instana-agent"):
    raise ValueError("GCP Helm index is missing valid instana-agent entries")
PY
echo "GCP read check passed: ${INDEX} (upload permissions not tested)"
