#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: mirror-instana-images.sh --registry <host[:port]> --namespace <namespace> [options]

Mirrors the current Instana images into an OpenShift internal registry and tags
both :latest and the resolved version in the destination repository. Multi-arch
images are copied as a single platform image by default for better compatibility
with the OpenShift internal registry route.

Required:
  --registry <host[:port]>          Destination registry hostname
  --namespace <namespace>           Destination namespace / repository prefix

Optional:
  --dest-creds <user:pass>          Destination registry credentials (stored in a temp auth file, not passed as a process argument)
  --dest-tls-verify <true|false>    Destination TLS verification (default: false)
  --agent-download-key <key>        Download key for containers.instana.io
  --multi-arch <mode>               Skopeo multi-arch mode (default: system)
  --output-file <path>              Output properties file (default: ./mirror-instana-images.properties)
  --operator-image <image>          Source operator image
  --agent-image <image>             Source agent image
  --k8sensor-image <image>          Source k8sensor image
  --help                            Show this help

Defaults:
  operator-image = icr.io/instana/instana-agent-operator
  agent-image    = containers.instana.io/instana/release/agent/static
  k8sensor-image = icr.io/instana/k8sensor

Examples:
  # Generic registry
  ./mirror-instana-images.sh \
    --registry registry.example.com \
    --namespace instana-mirror \
    --dest-creds "user:password" \
    --agent-download-key "$INSTANA_AGENT_DOWNLOAD_KEY"

  # OpenShift internal registry (via default route)
  TOKEN=$(oc whoami -t)
  INSTANA_AGENT_DOWNLOAD_KEY=""
  REGISTRY=$(oc get route default-route \
    -n openshift-image-registry \
    -o jsonpath='{.spec.host}')
  ./mirror-instana-images.sh \
    --registry "$REGISTRY" \
    --namespace instana-mirror \
    --dest-creds "kubeadmin:$TOKEN" \
    --dest-tls-verify false \
    --agent-download-key "$INSTANA_AGENT_DOWNLOAD_KEY"
EOF
}

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2
}

timed_run() {
  local description="$1"
  shift
  local start end elapsed status

  start="$(date +%s)"
  log "START: $description"
  set +e
  "$@"
  status=$?
  set -e
  end="$(date +%s)"
  elapsed="$((end - start))"
  log "DONE: $description (${elapsed}s, exit=$status)"
  return "$status"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Required command not found: $1" >&2
    exit 1
  }
}

# Write credentials to a temp auth file and clean it up on exit.
# Sets DEST_AUTHFILE (empty string if no credentials were provided).
DEST_AUTHFILE=""
setup_dest_auth() {
  if [ -z "$DEST_CREDS" ]; then
    return
  fi
  local user pass registry_host
  user="${DEST_CREDS%%:*}"
  pass="${DEST_CREDS#*:}"
  registry_host="${REGISTRY%%/*}"
  DEST_AUTHFILE="$(mktemp -u --suffix=.json)"
  # Register cleanup so the file is always removed, even on error or SIGINT.
  trap 'rm -f "$DEST_AUTHFILE"' EXIT
  # skopeo login creates and writes the credentials JSON to the given authfile path.
  # We use mktemp -u (no pre-creation) because skopeo rejects a pre-existing empty file.
  skopeo login \
    --authfile "$DEST_AUTHFILE" \
    --tls-verify="$DEST_TLS_VERIFY" \
    --username "$user" \
    --password "$pass" \
    "$registry_host" >/dev/null
  log "Destination credentials stored in temp auth file (not in process args)"
}

login_agent_registry() {
  if [ -n "$AGENT_DOWNLOAD_KEY" ]; then
    timed_run "skopeo login containers.instana.io" \
      skopeo login -u _ -p "$AGENT_DOWNLOAD_KEY" containers.instana.io >/dev/null
  fi
}

semver_tags() {
  local image="$1"
  local inspect_out
  # Capture stderr separately so a network/auth error is not silently eaten
  # by the grep pipe's exit-1-on-no-match.
  inspect_out="$(skopeo inspect "docker://$image:latest" 2>&1)" || {
    printf 'skopeo inspect failed for %s:latest: %s\n' "$image" "$inspect_out" >&2
    return 1
  }
  printf '%s\n' "$inspect_out" \
    | jq -r '.RepoTags[]' \
    | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -t. -k1,1nr -k2,2nr -k3,3nr \
    || true   # grep exits 1 on no match; that is not an error here
}

resolve_latest_semver_tag_by_digest() {
  local image="$1"
  local latest_digest tag digest result

  log "Inspecting digest for $image:latest"
  latest_digest="$(skopeo inspect "docker://$image:latest" | jq -r '.Digest')"

  if [ -z "$latest_digest" ] || [ "$latest_digest" = "null" ]; then
    log "ERROR: could not resolve digest for $image:latest"
    return 1
  fi

  result=""
  while read -r tag; do
    [ -n "$tag" ] || continue
    log "Checking $image:$tag against latest digest"
    digest="$(skopeo inspect "docker://$image:$tag" | jq -r '.Digest')"
    if [ "$digest" = "$latest_digest" ]; then
      log "Matched latest digest with $image:$tag"
      result="$tag"
      break
    fi
  done < <(semver_tags "$image")

  if [ -z "$result" ]; then
    log "ERROR: no semver tag matched the digest of $image:latest"
    return 1
  fi
  echo "$result"
}

resolve_agent_version() {
  local image="$1"
  local version

  log "Inspecting label version for $image:latest"
  version="$(skopeo inspect "docker://$image:latest" | jq -r '.Labels.version // empty')"

  if [ -z "$version" ]; then
    log "ERROR: .Labels.version is missing or null in $image:latest"
    return 1
  fi
  echo "$version"
}

# Detect the host architecture and map it to a Docker arch name.
host_arch() {
  local machine
  machine="$(uname -m)"
  case "$machine" in
    x86_64)          echo "amd64" ;;
    aarch64|arm64)   echo "arm64" ;;
    s390x)           echo "s390x" ;;
    ppc64le)         echo "ppc64le" ;;
    *)               echo "$machine" ;;  # pass through and let skopeo decide
  esac
}

copy_tag() {
  local src_image="$1"
  local src_tag="$2"
  local dest_repo="$3"
  local dest_tag="$4"

  local cmd=(skopeo copy)

  if [ -n "$DEST_AUTHFILE" ]; then
    cmd+=(--dest-authfile "$DEST_AUTHFILE")
  fi

  if [ "$MULTI_ARCH" = "system" ]; then
    cmd+=(--override-arch "$(host_arch)")
  else
    cmd+=(--multi-arch="$MULTI_ARCH")
  fi

  cmd+=(--dest-tls-verify="$DEST_TLS_VERIFY")
  cmd+=("docker://$src_image:$src_tag")
  cmd+=("docker://$REGISTRY/$dest_repo:$dest_tag")

  if ! timed_run "Copying $src_image:$src_tag -> $REGISTRY/$dest_repo:$dest_tag" \
    "${cmd[@]}"; then
    if [[ "$REGISTRY" == *.apps.* ]] || [[ "$REGISTRY" == image-registry.openshift-image-registry.svc:* ]]; then
      cat >&2 <<EOF
Copy to OpenShift internal registry failed.
Make sure the target project/namespace exists before mirroring, for example:
  oc new-project $NAMESPACE
EOF
    fi
    exit 1
  fi
}

write_output_property() {
  local key="$1"
  local value="$2"

  printf '%s=%s\n' "$key" "$value" | tee -a "$OUTPUT_FILE"
}

resolve_dest_digest() {
  local dest_repo="$1"
  local dest_tag="$2"
  local inspect_args=(skopeo inspect --tls-verify="$DEST_TLS_VERIFY")

  if [ -n "$DEST_AUTHFILE" ]; then
    inspect_args+=(--authfile "$DEST_AUTHFILE")
  fi

  inspect_args+=("docker://$REGISTRY/$dest_repo:$dest_tag")
  "${inspect_args[@]}" | jq -r '.Digest'
}

mirror_image() {
  local component="$1"
  local src_image="$2"
  local resolved_version="$3"
  local dest_name="$4"
  local dest_repo="$NAMESPACE/$dest_name"

  copy_tag "$src_image" latest "$dest_repo" latest
  copy_tag "$src_image" latest "$dest_repo" "$resolved_version"

  local digest
  digest="$(resolve_dest_digest "$dest_repo" "$resolved_version")"

  write_output_property "${component}_version"     "$resolved_version"
  write_output_property "${component}_digest"      "$digest"
  write_output_property "${component}_source"      "$src_image:latest"
  write_output_property "${component}_destination" "$REGISTRY/$dest_repo"
}

REGISTRY=""
NAMESPACE=""
DEST_CREDS=""
DEST_TLS_VERIFY="false"
AGENT_DOWNLOAD_KEY=""
MULTI_ARCH="system"
OUTPUT_FILE="./mirror-instana-images.properties"
OPERATOR_IMAGE="icr.io/instana/instana-agent-operator"
AGENT_IMAGE="containers.instana.io/instana/release/agent/static"
K8SENSOR_IMAGE="icr.io/instana/k8sensor"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --registry) REGISTRY="$2"; shift 2 ;;
    --namespace) NAMESPACE="$2"; shift 2 ;;
    --dest-creds) DEST_CREDS="$2"; shift 2 ;;
    --dest-tls-verify) DEST_TLS_VERIFY="$2"; shift 2 ;;
    --agent-download-key) AGENT_DOWNLOAD_KEY="$2"; shift 2 ;;
    --multi-arch) MULTI_ARCH="$2"; shift 2 ;;
    --output-file) OUTPUT_FILE="$2"; shift 2 ;;
    --operator-image) OPERATOR_IMAGE="$2"; shift 2 ;;
    --agent-image) AGENT_IMAGE="$2"; shift 2 ;;
    --k8sensor-image) K8SENSOR_IMAGE="$2"; shift 2 ;;
    --help) usage; exit 0 ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

[ -n "$REGISTRY" ] || { echo "Missing required --registry" >&2; usage >&2; exit 1; }
[ -n "$NAMESPACE" ] || { echo "Missing required --namespace" >&2; usage >&2; exit 1; }

case "$DEST_TLS_VERIFY" in
  true|false) ;;
  *) echo "Invalid --dest-tls-verify '$DEST_TLS_VERIFY': must be true or false" >&2; exit 1 ;;
esac

case "$MULTI_ARCH" in
  system|all|index-only) ;;
  *) echo "Invalid --multi-arch '$MULTI_ARCH': must be system, all, or index-only" >&2; exit 1 ;;
esac

require_command skopeo
require_command jq

login_agent_registry
setup_dest_auth

# Resolve all versions before touching the output file; this way a failed
# version lookup never leaves behind a truncated or partial properties file.
log "Resolving operator version from $OPERATOR_IMAGE:latest"
operator_version="$(resolve_latest_semver_tag_by_digest "$OPERATOR_IMAGE")"

log "Resolving agent version from $AGENT_IMAGE:latest"
agent_version="$(resolve_agent_version "$AGENT_IMAGE")"

log "Resolving k8sensor version from $K8SENSOR_IMAGE:latest"
k8sensor_version="$(resolve_latest_semver_tag_by_digest "$K8SENSOR_IMAGE")"

# All versions resolved successfully — safe to (re)create the output file.
: > "$OUTPUT_FILE"

mirror_image operator "$OPERATOR_IMAGE" "$operator_version" instana-agent-operator
mirror_image agent "$AGENT_IMAGE" "$agent_version" agent
mirror_image k8sensor "$K8SENSOR_IMAGE" "$k8sensor_version" k8sensor

log "Wrote mirror properties to $OUTPUT_FILE"
