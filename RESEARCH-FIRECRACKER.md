# Firecracker — The Fly.io Moat

## What makes Firecracker special?

Firecracker is AWS's open-source VMM written in Rust. It powers:
- AWS Lambda (millions of executions/second)
- AWS Fargate (serverless containers)
- Fly.io (edge microVMs)

## Architecture

Each microVM is a lightweight virtual machine with:
- Dedicated kernel (no shared kernel like containers)
- Hardware isolation via KVM
- <125ms boot (pre-built kernel + minimal rootfs)
- <5MB memory overhead (vs 50MB for QEMU)

## Why Nidus needs this

Docker containers share the host kernel. A kernel exploit in one
container compromises ALL containers. Firecracker provides true
hardware-level isolation — each tenant gets their own kernel.

## MicroVM lifecycle

```
Create → Configure (kernel, rootfs, network) → Start (<125ms)
  → Running → Stop (<10ms) → Destroy

Pre-warmed pool:
  Pool of N VMs already started, waiting for API commands
  Deploy time: instant (just configure + start app)
```

## Density comparison

| Technology | VMs per 32GB server | Memory per VM | Isolation |
|-----------|---------------------|---------------|-----------|
| Docker | 100-200 | 150-300MB | Namespace |
| Firecracker | 2000-5000 | 5-20MB | Hypervisor |
| QEMU/KVM | 50-100 | 300-500MB | Hypervisor |
