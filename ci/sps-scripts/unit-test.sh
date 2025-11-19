#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail
# note: PIPELINE_CONFIG_REPO_PATH will point to config, not to the app folder with the current branch, use APP_REPO_FOLDER instead
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
	trap env EXIT
	env
	set -x
fi
source $WORKSPACE/$APP_REPO_FOLDER/installGolang.sh amd64
export PATH=$PATH:/usr/local/go/bin
pwd
cd $WORKSPACE/$APP_REPO_FOLDER
pwd
echo "GIT COMMIT TO TEST: $(git rev-parse --verify HEAD)"
make generate
go install
make test
