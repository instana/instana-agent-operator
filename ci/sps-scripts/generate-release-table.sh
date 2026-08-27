#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2026
#
# Run generate-release-table.py from SPS.
# Required pipeline environment properties:
#   git-token   — github.com personal access token (read as GH_TOKEN)
#   ghe-token   — github.ibm.com personal access token (read as GHE_TOKEN)
#
set -eo pipefail

# RELEASE_TABLE_REPO_FOLDER must be set by the caller to point at the checkout
# that holds ci/scripts/release-table (APP_REPO_FOLDER for PR runs,
# PIPELINE_CONFIG_REPO_PATH for simple-execute runs).
SCRIPT_DIR="${WORKSPACE}/${RELEASE_TABLE_REPO_FOLDER}/ci/scripts/release-table"
PUBLISH_FLAG="${GENERATE_RELEASE_TABLE_PUBLISH:-false}"

if [ "${PUBLISH_FLAG}" = "true" ]; then
    RUN_ARGS=(--publish)
else
    RUN_ARGS=()
fi

echo "===== generate-release-table.sh - start ====="
echo "Script dir: ${SCRIPT_DIR}"
echo "Publish mode: ${PUBLISH_FLAG}"

echo "--- Running unit tests ---"
python3 -m ensurepip --upgrade
python3 -m pip install --quiet pytest
python3 -m pytest "${SCRIPT_DIR}/test_generate_release_table.py" -v

echo "--- Resolving tokens from SPS environment ---"
GH_TOKEN="$(get_secret git-token)"
export GH_TOKEN
GHE_TOKEN="$(get_secret ghe-token)"
export GHE_TOKEN

echo "--- Running generate-release-table.py ${RUN_ARGS[*]} ---"
python3 "${SCRIPT_DIR}/generate-release-table.py" "${RUN_ARGS[@]}"

echo "--- Generated release-timeline.md ---"
cat release-timeline.md

echo "===== generate-release-table.sh - end ====="
