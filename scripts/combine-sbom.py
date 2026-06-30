import json, glob, os, sys

outdir = sys.argv[1] if len(sys.argv) > 1 else "/root/nidus/sbom/sbom-parts"
outpath = sys.argv[2] if len(sys.argv) > 2 else "/root/nidus/sbom/nidus-sbom.cdx.json"
parts = sorted(glob.glob(os.path.join(outdir, "*.cdx.json")))
if parts:
    combined = json.load(open(parts[0]))
    for p in parts[1:]:
        other = json.load(open(p))
        combined.setdefault("components", [])
        combined["components"].extend(other.get("components", []))
    count = len(combined.get("components", []))
    sz = os.path.getsize(outpath)
    print(f"  Combined SBOM: {sz:,} bytes ({count} components)")
else:
    print("  No SBOM parts to combine")
