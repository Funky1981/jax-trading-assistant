#!/bin/bash
# compare-golden-outputs.sh
# Bash implementation of golden output comparison for CI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

GOLDEN_DIR="tests/golden"
BASELINE_DIR="${GOLDEN_DIR}/baseline"
CURRENT_DIR="${GOLDEN_DIR}"

echo "🔍 Starting golden test comparison..."
echo "Baseline: ${BASELINE_DIR}"
echo "Current:  ${CURRENT_DIR}"
echo ""

# Check if baseline directory exists
if [ ! -d "${BASELINE_DIR}" ]; then
    echo -e "${YELLOW}⚠️  Warning: Baseline directory not found at ${BASELINE_DIR}${NC}"
    echo "This may be the first run. Creating baseline from current state..."
    mkdir -p "${BASELINE_DIR}"
    
    # Copy current snapshots to baseline if they exist
    for category in signals executions orchestration; do
        if [ -d "${CURRENT_DIR}/${category}" ]; then
            mkdir -p "${BASELINE_DIR}/${category}"
            cp -r "${CURRENT_DIR}/${category}"/*.json "${BASELINE_DIR}/${category}/" 2>/dev/null || true
        fi
    done
    
    echo -e "${GREEN}✅ Baseline created${NC}"
    exit 0
fi

# Function to compare JSON files
compare_json() {
    local baseline_file=$1
    local current_file=$2
    local category=$3
    
    if [ ! -f "${baseline_file}" ]; then
        echo -e "${YELLOW}⚠️  Baseline file missing: ${baseline_file}${NC}"
        return 1
    fi
    
    if [ ! -f "${current_file}" ]; then
        echo -e "${RED}❌ Current file missing: ${current_file}${NC}"
        return 1
    fi
    
    # Use jq to compare JSON if available, otherwise use diff
    if command -v jq &> /dev/null; then
        # Normalize JSON and compare
        baseline_normalized=$(jq -S . "${baseline_file}" 2>/dev/null || cat "${baseline_file}")
        current_normalized=$(jq -S . "${current_file}" 2>/dev/null || cat "${current_file}")
        
        if [ "${baseline_normalized}" == "${current_normalized}" ]; then
            return 0
        else
            echo -e "${RED}❌ Mismatch in ${category}: $(basename ${current_file})${NC}"
            
            # Create diff file
            diff_file="${CURRENT_DIR}/${category}/diff-$(basename ${current_file})"
            diff -u <(echo "${baseline_normalized}") <(echo "${current_normalized}") > "${diff_file}" || true
            echo "   Diff saved to: ${diff_file}"
            return 1
        fi
    else
        # Fallback to simple diff
        if diff -q "${baseline_file}" "${current_file}" &> /dev/null; then
            return 0
        else
            echo -e "${RED}❌ Mismatch in ${category}: $(basename ${current_file})${NC}"
            
            diff_file="${CURRENT_DIR}/${category}/diff-$(basename ${current_file} .json).diff"
            diff -u "${baseline_file}" "${current_file}" > "${diff_file}" || true
            echo "   Diff saved to: ${diff_file}"
            return 1
        fi
    fi
}

# Compare signals
echo "📊 Comparing signals..."
signals_ok=true
if [ -d "${BASELINE_DIR}/signals" ]; then
    for baseline_file in "${BASELINE_DIR}"/signals/*.json; do
        if [ -f "${baseline_file}" ]; then
            filename=$(basename "${baseline_file}")
            current_file="${CURRENT_DIR}/signals/${filename}"
            
            if compare_json "${baseline_file}" "${current_file}" "signals"; then
                echo -e "  ${GREEN}✓${NC} ${filename}"
            else
                signals_ok=false
            fi
        fi
    done
else
    echo -e "${YELLOW}⚠️  No baseline signals found${NC}"
fi

# Compare executions
echo ""
echo "⚡ Comparing executions..."
executions_ok=true
if [ -d "${BASELINE_DIR}/executions" ]; then
    for baseline_file in "${BASELINE_DIR}"/executions/*.json; do
        if [ -f "${baseline_file}" ]; then
            filename=$(basename "${baseline_file}")
            current_file="${CURRENT_DIR}/executions/${filename}"
            
            if compare_json "${baseline_file}" "${current_file}" "executions"; then
                echo -e "  ${GREEN}✓${NC} ${filename}"
            else
                executions_ok=false
            fi
        fi
    done
else
    echo -e "${YELLOW}⚠️  No baseline executions found${NC}"
fi

# Compare orchestration
echo ""
echo "🎭 Comparing orchestration..."
orchestration_ok=true
if [ -d "${BASELINE_DIR}/orchestration" ]; then
    for baseline_file in "${BASELINE_DIR}"/orchestration/*.json; do
        if [ -f "${baseline_file}" ]; then
            filename=$(basename "${baseline_file}")
            current_file="${CURRENT_DIR}/orchestration/${filename}"
            
            if compare_json "${baseline_file}" "${current_file}" "orchestration"; then
                echo -e "  ${GREEN}✓${NC} ${filename}"
            else
                orchestration_ok=false
            fi
        fi
    done
else
    echo -e "${YELLOW}⚠️  No baseline orchestration found${NC}"
fi

# Summary
echo ""
echo "================================================"
echo "Golden Test Comparison Summary"
echo "================================================"

exit_code=0

if [ "${signals_ok}" = true ]; then
    echo -e "${GREEN}✅ Signals: PASS${NC}"
else
    echo -e "${RED}❌ Signals: FAIL${NC}"
    exit_code=1
fi

if [ "${executions_ok}" = true ]; then
    echo -e "${GREEN}✅ Executions: PASS${NC}"
else
    echo -e "${RED}❌ Executions: FAIL${NC}"
    exit_code=1
fi

if [ "${orchestration_ok}" = true ]; then
    echo -e "${GREEN}✅ Orchestration: PASS${NC}"
else
    echo -e "${RED}❌ Orchestration: FAIL${NC}"
    exit_code=1
fi

echo "================================================"

if [ ${exit_code} -eq 0 ]; then
    echo -e "${GREEN}🎉 All golden tests passed!${NC}"
else
    echo -e "${RED}💥 Golden tests failed! Review the diffs above.${NC}"
    echo ""
    echo "To update the baseline (if changes are intentional):"
    echo "  ./scripts/capture-golden-baseline.ps1"
fi

exit ${exit_code}
