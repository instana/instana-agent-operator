#!/bin/bash

set -e

SCRIPT=$0
VERSION=${1:-dev}
MANIFEST_NAME=${2:-olm}

if [ $MANIFEST_NAME = "olm" ] ; then
  REDHAT=false
elif [ $MANIFEST_NAME = "redhat" ] ; then
  REGISTRY="registry.connect.redhat.com"
  REDHAT=true
fi

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
TARGET_DIR="$SCRIPTPATH/../target"
MANIFEST_DIR="$TARGET_DIR/$MANIFEST_NAME"
ZIP_PATH="$TARGET_DIR/$MANIFEST_NAME-$VERSION.zip"
PREV_PACKAGE_URL="https://raw.githubusercontent.com/operator-framework/community-operators/master/upstream-community-operators/instana-agent/instana-agent.package.yaml"
PREV_PACKAGE="$( wget -qO- "$PREV_PACKAGE_URL" | $SCRIPTPATH/yaml_to_json )"
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
CRD_DESCRIPTORS=$($SCRIPTPATH/yaml_to_json < "$SCRIPTPATH/CRD.descriptors.yaml")
EXAMPLES=$($SCRIPTPATH/yaml_to_json < "$SCRIPTPATH/../deploy/instana-agent.customresource.yaml")
RESOURCES=$($SCRIPTPATH/yaml_to_json < "$SCRIPTPATH/../deploy/instana-agent-operator.yaml")

mkdir -p $MANIFEST_DIR

jsonnet \
  --ext-str crd_descriptors="$CRD_DESCRIPTORS" \
  --ext-str-file description=$SCRIPTPATH/description.md \
  --ext-str examples="$EXAMPLES" \
  --ext-str-file image=$SCRIPTPATH/image.svg \
  --ext-str isoDate=$DATE \
  --ext-str registry=$REGISTRY \
  --ext-str prevPackage="$PREV_PACKAGE" \
  --ext-str redhat="$REDHAT" \
  --ext-str resources="$RESOURCES" \
  --ext-str version=$VERSION \
  -m $MANIFEST_DIR \
  $SCRIPTPATH/template.jsonnet

for f in $MANIFEST_DIR/*.json
do
  [ -f "$f" ] || break
  $SCRIPTPATH/json_to_yaml < $f > ${f%json}yaml
  rm $f
done

operator-courier verify $MANIFEST_DIR

pushd $MANIFEST_DIR
zip $ZIP_PATH ./*
popd