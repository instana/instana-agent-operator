#!/usr/bin/env bash
set -euo pipefail
# note: PIPELINE_CONFIG_REPO_PATH will point to config, not to the app folder with the current branch, use APP_REPO_FOLDER instead
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
	trap env EXIT
	env
	set -x
fi
./installGolang.sh 1.24.4 amd64
export PATH=$PATH:/usr/local/go/bin
make generate
go install
make build
