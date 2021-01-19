#!/bin/bash
#
# (c) Copyright IBM Corp. 2021
# (c) Copyright Instana Inc.
#


set -e

VERSION=${1:-dev}
ROOT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )/../.."
SCRIPT_DIR="$ROOT_DIR/olm"
MANIFEST_NAME="operator-resources"
OPERATOR_RESOURCES_DIR="$SCRIPT_DIR/$MANIFEST_NAME"
TARGET_DIR="$ROOT_DIR/target"
MANIFEST_DIR="$TARGET_DIR/$MANIFEST_NAME/$VERSION"
OPERATOR_RESOURCES=$(${SCRIPT_DIR}/yaml_to_json < "$OPERATOR_RESOURCES_DIR/instana-agent-operator.yaml")

mkdir -p ${MANIFEST_DIR}

jsonnet \
  --ext-str operatorResources="$OPERATOR_RESOURCES" \
  --ext-str version=${VERSION} \
  -m ${MANIFEST_DIR} \
  ${OPERATOR_RESOURCES_DIR}/operator-artifacts.jsonnet

for f in ${MANIFEST_DIR}/*.json
do
  [[ -f "$f" ]] || break
  ${SCRIPT_DIR}/json_to_yaml < $f > ${f%json}yaml
  rm $f
done

for f in ${MANIFEST_DIR}/*.yaml
do
  [[ -f "$f" ]] || break
  { echo "---" ; cat ${f}; } >> ${TARGET_DIR}/instana-agent-operator.v${VERSION}.yaml
  rm ${f}
done

mv ${TARGET_DIR}/instana-agent-operator.v${VERSION}.yaml ${MANIFEST_DIR}/instana-agent-operator.yaml
