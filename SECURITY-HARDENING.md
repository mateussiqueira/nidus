# StackRun Security Hardening

## Secrets Management
- [x] No secrets in source code (verified by gitleaks)
- [x] DATABASE_URL via environment variables
- [x] JWT secret from env, not hardcoded
- [ ] HashiCorp Vault integration (Phase 4)
- [ ] Dynamic DB credentials (TTL rotation)

## Container Security
- [x] Non-root user (nidus)
- [x] Minimal base image (debian:bookworm-slim)
- [x] HEALTHCHECK in Dockerfile
- [ ] Read-only root filesystem
- [ ] seccomp profile (custom)
- [ ] AppArmor/SELinux profiles
- [ ] Image signing (cosign)

## Dependency Security
- [x] cargo audit (vulnerability scan)
- [x] CycloneDX SBOM (supply chain)
- [x] trivy image scan
- [ ] Dependabot on GitHub
- [ ] Renovate for auto-updates

## Network Security
- [x] Caddy TLS termination
- [x] Internal Docker network.*stackrun)
- [ ] mTLS between services (tonic + rustls)
- [ ] Rate limiting (implemented)
- [ ] WAF (Cloudflare or self-hosted)

## Compliance
- [ ] GDPR: data residency, right to deletion
- [ ] SOC 2: audit logging, access controls
- [ ] ISO 27001: risk assessment

## Penetration Testing
- [ ] OWASP Top 10 review
- [ ] Dependency confusion attack check
- [ ] SQL injection test suite
- [ ] XSS in dashboard

## Runtime Protection
- [x] Health check monitoring
- [x] Auto-restart on failure
- [ ] WAF rules
- [ ] DDoS protection (rate limiting)
- [ ] Audit logging (implemented)
