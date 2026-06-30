#!/bin/bash
# Nidus SBOM Generator — CycloneDX + cargo audit + trivy + gitleaks
set -e

OUTPUT_DIR="${1:-/root/nidus/sbom}"
mkdir -p "$OUTPUT_DIR"

echo "═══════════════════════════════════"
echo " Nidus SBOM Generator"
echo " Output: $OUTPUT_DIR"
echo "═══════════════════════════════════"

which cargo-audit || cargo install cargo-audit 2>/dev/null
which cargo-cyclonedx || cargo install cargo-cyclonedx 2>/dev/null

echo ""
echo "[1/5] cargo audit — vulnerability scan..."
cd /root/nidus/rust
cargo audit --json > "$OUTPUT_DIR/cargo-audit.json" 2>&1 || true
VULNS=$(python3 -c "
import json
d=json.load(open('$OUTPUT_DIR/cargo-audit.json'))
v=d.get('vulnerabilities',{})
print(v.get('count',0))
" 2>/dev/null || echo "?")
echo "  Vulnerabilities found: $VULNS"

echo "[2/5] cargo cyclonedx — SBOM generation..."
cd /root/nidus/rust
cargo cyclonedx --all --format json 2>/dev/null || true
mkdir -p "$OUTPUT_DIR/sbom-parts"
find . -name "*.cdx.json" -exec cp {} "$OUTPUT_DIR/sbom-parts/" \; 2>/dev/null
python3 /root/nidus/scripts/combine-sbom.py "$OUTPUT_DIR/sbom-parts" "$OUTPUT_DIR/nidus-sbom.cdx.json"

echo "[3/5] trivy — container image scan..."
if which trivy; then
    docker images --format "{{.Repository}}:{{.Tag}}" | grep nidus | while read img; do
        echo "  Scanning: $img"
        trivy image --quiet --format json -o "$OUTPUT_DIR/trivy-${img//\//_}.json" "$img" 2>/dev/null || true
    done
else
    echo "  (trivy not installed — skipping)"
fi

echo "[4/5] gitleaks — secret leakage check..."
cd /root/nidus
if which gitleaks; then
    gitleaks detect --source . --report-path "$OUTPUT_DIR/gitleaks.json" --verbose 2>/dev/null || true
    if [ -f "$OUTPUT_DIR/gitleaks.json" ]; then
        LEAKS=$(python3 -c "
import json
d=json.load(open('$OUTPUT_DIR/gitleaks.json'))
print(len(d) if isinstance(d,list) else '?')
" 2>/dev/null || echo "?")
        echo "  Secrets found: $LEAKS"
    fi
else
    echo "  (gitleaks not installed — skipping)"
    echo "  Install: curl -sSfL https://github.com/gitleaks/gitleaks/releases/latest/download/gitleaks_linux_amd64.tar.gz | tar xz -C /usr/local/bin gitleaks"
fi

echo ""
echo "═══════════════════════════════════"
echo " SBOM generation complete!"
echo " Files:"
ls -lh "$OUTPUT_DIR/" 2>/dev/null
ls -lh "$OUTPUT_DIR/sbom-parts/" 2>/dev/null
echo "═══════════════════════════════════"
