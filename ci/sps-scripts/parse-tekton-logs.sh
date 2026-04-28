#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2026
#
# Parse Tekton pipeline logs to extract only relevant failure information
# This reduces token usage when processing logs with AI coding agents
#
# Usage:
#   ./parse-tekton-logs.sh <log-file>
#   ibmcloud dev tekton-logs <pipeline-id> --run-id <run-id> | ./parse-tekton-logs.sh
#

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Read from file or stdin
INPUT="${1:-/dev/stdin}"

# Temporary file for processing
TEMP_FILE=$(mktemp)
trap "rm -f ${TEMP_FILE}" EXIT

cat "${INPUT}" > "${TEMP_FILE}"

echo "========================================"
echo "TEKTON PIPELINE LOG SUMMARY"
echo "========================================"
echo ""

# Extract pipeline run information
echo -e "${BLUE}=== Pipeline Information ===${NC}"
grep -E "^Logs for task.*step" "${TEMP_FILE}" | head -5 || echo "No task information found"
echo ""

# Check for overall success/failure
echo -e "${BLUE}=== Overall Status ===${NC}"
if grep -q "Exit Code: '1'" "${TEMP_FILE}"; then
    echo -e "${RED}❌ FAILED${NC}"
    FAILED=1
else
    echo -e "${GREEN}✅ SUCCESS${NC}"
    FAILED=0
fi
echo ""

# Extract test failures
echo -e "${BLUE}=== Test Failures ===${NC}"
if grep -E "(--- FAIL:|FAIL\s|panic:)" "${TEMP_FILE}" > /dev/null 2>&1; then
    # Extract test failure context (10 lines before and after)
    grep -B 10 -A 5 -E "(--- FAIL:|FAIL\s.*github.com|panic:)" "${TEMP_FILE}" | \
        grep -v "^--$" | \
        grep -v "time=\".*level=" | \
        sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' || true
else
    echo "No test failures found"
fi
echo ""

# Extract panic stack traces
echo -e "${BLUE}=== Panic Stack Traces ===${NC}"
if grep -q "^.*panic:" "${TEMP_FILE}"; then
    # Extract panic and following stack trace (up to 20 lines)
    awk '/panic:/{found=1; count=0} found{print; count++; if(count>=20) found=0}' "${TEMP_FILE}" | \
        sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' || true
else
    echo "No panics found"
fi
echo ""

# Extract error messages
echo -e "${BLUE}=== Error Messages ===${NC}"
grep -E "(Error:|error:|ERROR|FAILED|failed to)" "${TEMP_FILE}" | \
    grep -v "skip loading plugin" | \
    grep -v "failed to load plugin" | \
    grep -v "failed to initialize a tracing processor" | \
    grep -v "aufs is not supported" | \
    grep -v "NotFound" | \
    grep -v "::error::" | \
    sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' | \
    head -20 || echo "No significant errors found"
echo ""

# Extract exit codes
echo -e "${BLUE}=== Exit Codes ===${NC}"
grep -E "Exit Code:|exit status|make:.*Error" "${TEMP_FILE}" | \
    grep -v "exit status 1.*aufs" | \
    sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' || echo "No exit codes found"
echo ""

# Extract test summary if present
echo -e "${BLUE}=== Test Summary ===${NC}"
if grep -q "^.*ok.*coverage:" "${TEMP_FILE}"; then
    grep -E "^.*(ok|FAIL)\s+github.com/instana" "${TEMP_FILE}" | \
        sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' || true
else
    echo "No test summary found"
fi
echo ""

# Extract commit status updates
echo -e "${BLUE}=== Commit Status Updates ===${NC}"
grep -E "set-commit-status.*--state=" "${TEMP_FILE}" | \
    sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' | \
    grep -oE "(--state=[^ ]+|--context=[^ ]+|--description=[^-]+)" | \
    paste -d ' ' - - - || echo "No commit status updates found"
echo ""

# Extract cleanup messages
echo -e "${BLUE}=== Cleanup ===${NC}"
grep -E "(Cleaning up|Deleted \[https://)" "${TEMP_FILE}" | \
    sed 's/^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}T[0-9:\.]*Z //' | \
    tail -5 || echo "No cleanup messages found"
echo ""

# Summary statistics
echo -e "${BLUE}=== Statistics ===${NC}"
TOTAL_LINES=$(wc -l < "${TEMP_FILE}")
FAIL_COUNT=$(grep -c "FAIL" "${TEMP_FILE}" || echo "0")
ERROR_COUNT=$(grep -c -i "error" "${TEMP_FILE}" || echo "0")
echo "Total log lines: ${TOTAL_LINES}"
echo "FAIL occurrences: ${FAIL_COUNT}"
echo "Error occurrences: ${ERROR_COUNT}"
echo ""

echo "========================================"
echo "END OF SUMMARY"
echo "========================================"

# Exit with same code as pipeline
exit ${FAILED}

# Made with Bob
