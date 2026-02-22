#!/usr/bin/env bash
set -euo pipefail

require_var() {
    local name="$1"
    if [ -z "${!name:-}" ]; then
        echo "ERROR: Missing required env var: ${name}" >&2
        exit 1
    fi
}

require_var BUNNY_STORAGE_KEY
require_var BUNNY_STORAGE_ZONE
require_var BUNNY_STORAGE_ENDPOINT

BUNNY_ENDPOINT="${BUNNY_STORAGE_ENDPOINT#http://}"
BUNNY_ENDPOINT="${BUNNY_ENDPOINT#https://}"
BUNNY_ENDPOINT="${BUNNY_ENDPOINT%/}"
BUNNY_BASE_PATH="${BUNNY_BASE_PATH:-analytics-backup/current}"
BUNNY_BASE_PATH="${BUNNY_BASE_PATH#/}"
BUNNY_BASE_PATH="${BUNNY_BASE_PATH%/}"

if [ -z "${BUNNY_BASE_PATH}" ]; then
    echo "ERROR: BUNNY_BASE_PATH resolves to empty after normalization" >&2
    exit 1
fi

CURL_CONNECT_TIMEOUT="${BUNNY_CURL_CONNECT_TIMEOUT_SEC:-10}"
CURL_MAX_TIME="${BUNNY_CURL_MAX_TIME_SEC:-30}"

probe_rel="${BUNNY_BASE_PATH}/_healthchecks/credential-check-$(date -u +%Y%m%dT%H%M%SZ)-$$-${RANDOM}.txt"
probe_url="https://${BUNNY_ENDPOINT}/${BUNNY_STORAGE_ZONE}/${probe_rel}"

tmp_payload="$(mktemp -t bunny-smoke-payload.XXXXXX)"
tmp_download="$(mktemp -t bunny-smoke-download.XXXXXX)"
uploaded=0

cleanup() {
    rm -f "${tmp_payload}" "${tmp_download}"
    if [ "${uploaded}" = "1" ]; then
        curl -sS \
            --connect-timeout "${CURL_CONNECT_TIMEOUT}" \
            --max-time "${CURL_MAX_TIME}" \
            -o /dev/null \
            -X DELETE \
            -H "AccessKey: ${BUNNY_STORAGE_KEY}" \
            "${probe_url}" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

cat > "${tmp_payload}" <<EOF
credential smoke test
timestamp_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
probe_rel=${probe_rel}
EOF

echo "Running Bunny credential smoke test"
echo "   Endpoint: ${BUNNY_ENDPOINT}"
echo "   Zone: ${BUNNY_STORAGE_ZONE}"
echo "   Base path: ${BUNNY_BASE_PATH}"
echo "   Probe object: ${probe_rel}"

upload_code="$(
    curl -sS \
        --connect-timeout "${CURL_CONNECT_TIMEOUT}" \
        --max-time "${CURL_MAX_TIME}" \
        -o /dev/null \
        -w "%{http_code}" \
        -X PUT \
        -H "AccessKey: ${BUNNY_STORAGE_KEY}" \
        --data-binary "@${tmp_payload}" \
        "${probe_url}"
)"

if [ "${upload_code}" != "200" ] && [ "${upload_code}" != "201" ]; then
    echo "ERROR: Bunny upload test failed (HTTP ${upload_code})" >&2
    echo "   Check BUNNY_STORAGE_KEY / BUNNY_STORAGE_ZONE / BUNNY_STORAGE_ENDPOINT." >&2
    exit 1
fi
uploaded=1

download_code="$(
    curl -sS \
        --connect-timeout "${CURL_CONNECT_TIMEOUT}" \
        --max-time "${CURL_MAX_TIME}" \
        -o "${tmp_download}" \
        -w "%{http_code}" \
        -H "AccessKey: ${BUNNY_STORAGE_KEY}" \
        "${probe_url}"
)"

if [ "${download_code}" != "200" ]; then
    echo "ERROR: Bunny download test failed (HTTP ${download_code})" >&2
    exit 1
fi

if ! cmp -s "${tmp_payload}" "${tmp_download}"; then
    echo "ERROR: Bunny readback mismatch (uploaded content != downloaded content)" >&2
    exit 1
fi

delete_code="$(
    curl -sS \
        --connect-timeout "${CURL_CONNECT_TIMEOUT}" \
        --max-time "${CURL_MAX_TIME}" \
        -o /dev/null \
        -w "%{http_code}" \
        -X DELETE \
        -H "AccessKey: ${BUNNY_STORAGE_KEY}" \
        "${probe_url}"
)"

if [ "${delete_code}" != "200" ] && [ "${delete_code}" != "202" ] && [ "${delete_code}" != "204" ] && [ "${delete_code}" != "404" ]; then
    echo "ERROR: Bunny delete cleanup failed (HTTP ${delete_code})" >&2
    exit 1
fi
uploaded=0

echo "OK: Bunny credential test passed (write/read/delete succeeded)"
