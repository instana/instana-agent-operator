#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Stage: validate (release-promotion pipeline)
# 1. Reads BOM from COS at the configured release-state-path.
# 2. Validates all 4 E2E status files (unless skip-e2e-check=true).
# 3. Confirms required BOM fields are present.
#
set -euo pipefail
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== validate-release.sh - start ====="

# ---------------------------------------------------------------------------
# Read pipeline properties
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
RELEASE_STATE_PATH=$(get_env release-state-path "pipeline-state/latest-release-candidate/bom.json")
SKIP_E2E_CHECK=$(get_env skip-e2e-check "false")

echo "release-state-path=${RELEASE_STATE_PATH}"
echo "skip-e2e-check=${SKIP_E2E_CHECK}"

# ---------------------------------------------------------------------------
# Read BOM
# ---------------------------------------------------------------------------
BOM_FILE=$(mktemp)
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${RELEASE_STATE_PATH}" \
  --output json "${BOM_FILE}"

echo "BOM contents:"
jq . "${BOM_FILE}"

VERSION=$(jq -r '.version'                             "${BOM_FILE}")
RELEASE_TAG=$(jq -r '."release-tag"'                   "${BOM_FILE}")
COS_STATE_PATH=$(jq -r '."cos-state-path"'             "${BOM_FILE}")
OPERATOR_SHA=$(jq -r '.sources."operator-sha"'         "${BOM_FILE}")
CHARTS_SHA=$(jq -r '.sources."charts-sha"'             "${BOM_FILE}")
OPERATOR_MANIFEST=$(jq -r '.images."operator-manifest"'         "${BOM_FILE}")
OPERATOR_MANIFEST_DIGEST=$(jq -r '.images."operator-manifest-digest"' "${BOM_FILE}")
HELM_CHART=$(jq -r '.artifacts."helm-chart"'           "${BOM_FILE}")
OLM_BUNDLE=$(jq -r '.artifacts."olm-bundle"'           "${BOM_FILE}")
OPERATOR_YAML=$(jq -r '.artifacts."operator-yaml"'     "${BOM_FILE}")

# ---------------------------------------------------------------------------
# Validate required BOM fields
# ---------------------------------------------------------------------------
echo "=== Validating BOM fields ==="
FAIL=0

check_field() {
  local field_name="$1"
  local field_value="$2"
  if [[ -z "${field_value}" || "${field_value}" == "null" ]]; then
    echo "ERROR: BOM field '${field_name}' is missing or null"
    FAIL=1
  else
    echo "  OK: ${field_name} = ${field_value}"
  fi
}

check_field "version"                  "${VERSION}"
check_field "release-tag"              "${RELEASE_TAG}"
check_field "cos-state-path"           "${COS_STATE_PATH}"
check_field "sources.operator-sha"     "${OPERATOR_SHA}"
check_field "sources.charts-sha"       "${CHARTS_SHA}"
check_field "images.operator-manifest" "${OPERATOR_MANIFEST}"
check_field "images.operator-manifest-digest" "${OPERATOR_MANIFEST_DIGEST}"
check_field "artifacts.helm-chart"     "${HELM_CHART}"
check_field "artifacts.olm-bundle"     "${OLM_BUNDLE}"
check_field "artifacts.operator-yaml"  "${OPERATOR_YAML}"

if [[ ${FAIL} -ne 0 ]]; then
  echo "ERROR: BOM validation failed — one or more required fields are missing"
  exit 1
fi
echo "BOM field validation passed."

# ---------------------------------------------------------------------------
# Validate E2E status files (unless skip-e2e-check=true)
# ---------------------------------------------------------------------------
E2E_CLUSTERS=(
  "ocp-fyre-latest"
  "gke-operator-lowest"
  "ocp-fyre-lowest"
  "gke-operator-latest"
)

if [[ "${SKIP_E2E_CHECK}" == "true" ]]; then
  echo "WARNING: skip-e2e-check=true — skipping E2E status validation (emergency use only)"
else
  echo "=== Validating E2E status files ==="
  for CLUSTER in "${E2E_CLUSTERS[@]}"; do
    STATUS_KEY="${COS_STATE_PATH}/e2e/${CLUSTER}.json"
    STATUS_FILE=$(mktemp)
    echo "  Checking ${STATUS_KEY}..."

    if ! ibmcloud cos get-object \
        --bucket "${COS_BUCKET}" \
        --key "${STATUS_KEY}" \
        --output json "${STATUS_FILE}" 2>/dev/null; then
      echo "ERROR: E2E status missing for ${CLUSTER} — stage may not have completed (key: ${STATUS_KEY})"
      rm -f "${STATUS_FILE}"
      exit 1
    fi

    CLUSTER_STATUS=$(jq -r '.status' "${STATUS_FILE}")
    rm -f "${STATUS_FILE}"

    if [[ "${CLUSTER_STATUS}" != "success" ]]; then
      echo "ERROR: E2E failed for ${CLUSTER} (status=${CLUSTER_STATUS})"
      exit 1
    fi
    echo "  OK: ${CLUSTER} → ${CLUSTER_STATUS}"
  done
  echo "All E2E status files validated successfully."
fi

# Export the COS state path so downstream promotion stages can use it
set_env cos-state-path "${COS_STATE_PATH}"
export_env cos-state-path

rm -f "${BOM_FILE}"
echo "===== validate-release.sh - end ====="
