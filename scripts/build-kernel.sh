#!/bin/bash
# Download pre-built Firecracker kernel
set -e
OUTPUT="${1:-/var/lib/nidus/kernel/vmlinux-5.10}"
mkdir -p "$(dirname "$OUTPUT")"
KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.8/x86_64/vmlinux-5.10.225"
echo "Downloading kernel..."
curl -fsSL "$KERNEL_URL" -o "$OUTPUT"
echo "✓ Kernel downloaded: $(ls -lh "$OUTPUT" 2>/dev/null | awk '{print $5}')"
