#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Run generate-release-table.py --publish from the SPS simple-execute pipeline.
# Required pipeline environment properties:
#   git-token   — github.com personal access token (read as GH_TOKEN)
#   ghe-token   — github.ibm.com personal access token (read as GHE_TOKEN)
#
set -eo pipefail

SCRIPT_DIR="${WORKSPACE}/${PIPELINE_CONFIG_REPO_PATH}/ci/scripts/release-table"

echo "===== generate-release-table.sh - start ====="
echo "Script dir: ${SCRIPT_DIR}"

echo "--- Running unit tests ---"
python3 -m ensurepip --upgrade
python3 -m pip install --quiet pytest
python3 -m pytest "${SCRIPT_DIR}/test_generate_release_table.py" -v

echo "--- Resolving tokens from SPS environment ---"
GH_TOKEN="$(get_env git-token)"
export GH_TOKEN
GHE_TOKEN="$(get_env ghe-token)"
export GHE_TOKEN

echo "--- Running generate-release-table.py --publish ---"
python3 "${SCRIPT_DIR}/generate-release-table.py" --publish

echo "===== generate-release-table.sh - end ====="
