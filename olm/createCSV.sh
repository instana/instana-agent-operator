#!/bin/bash

set -e

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"

VERSION=${1:-dev}
PREV_VERSION=${PREV_VERSION:-prev}
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
CRD_DESCRIPTORS=$($SCRIPTPATH/yaml_to_json < "$SCRIPTPATH/CRD.descriptors.yaml")
EXAMPLES=$($SCRIPTPATH/yaml_to_json < "$SCRIPTPATH/../deploy/instana-agent.customresource.yaml")
RESOURCES=$($SCRIPTPATH/yaml_to_json < "$SCRIPTPATH/../deploy/instana-agent-operator.yaml")

mkdir -p target/olm

jsonnet \
  --ext-str crd_descriptors="$CRD_DESCRIPTORS" \
  --ext-str-file description=$SCRIPTPATH/description.md \
  --ext-str examples="$EXAMPLES" \
  --ext-str-file image=$SCRIPTPATH/image.svg \
  --ext-str isoDate=$DATE \
  --ext-str prevVersion=$PREV_VERSION \
  --ext-str resources="$RESOURCES" \
  --ext-str version=$VERSION \
  -m $SCRIPTPATH/../target/olm \
  $SCRIPTPATH/template.jsonnet

for f in $SCRIPTPATH/../target/olm/*.json;
do
  $SCRIPTPATH/json_to_yaml < $f > ${f%json}yaml
  rm $f
done

operator-courier verify $SCRIPTPATH/../target/olm
