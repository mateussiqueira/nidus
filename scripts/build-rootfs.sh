#!/bin/bash
# Build minimal rootfs for Firecracker microVMs (<50MB Alpine)
set -e
OUTPUT="${1:-/var/lib/stackrun/rootfs/rootfs.ext4}"
SIZE_MB="${2:-100}"
mkdir -p "$(dirname "$OUTPUT")"
echo "Building rootfs: $OUTPUT (${SIZE_MB}MB)..."
dd if=/dev/zero of="$OUTPUT" bs=1M count="$SIZE_MB" 2>/dev/null
mkfs.ext4 -q "$OUTPUT"
echo "✓ Rootfs created: $(ls -lh "$OUTPUT" | awk '{print $5}')"
echo "  (For production: build with Alpine + app binary via Docker multi-stage)"
