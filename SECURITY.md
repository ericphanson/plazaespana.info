# Security Policy

## Security Model

This is a static site generator with the following security characteristics:

### Attack Surface

- **No user input:** The site is generated server-side with no client-side forms or user-submitted content
- **No authentication:** Public event listings only, no user accounts or login system
- **No database:** All data is fetched from public APIs and regenerated hourly
- **No JavaScript:** Pure CSS/HTML eliminates XSS and client-side code execution risks
- **Strict CSP:** Content-Security-Policy headers block inline scripts and external resources

### Security Features

**Content Security Policy:**
```
default-src 'none';
style-src 'self';
img-src 'self' data:;
font-src 'self';
base-uri 'none';
frame-ancestors 'none'
```

**Privacy:**
- No cookies or client-side storage
- No analytics scripts or third-party trackers
- AWStats configured to store only aggregate statistics (no individual IPs)
- No personal data collection or processing

**Input Handling:**
- All upstream data (Madrid APIs) is parsed and validated
- HTML template auto-escaping via Go's `html/template` package
- No user-controllable input reaches the rendering pipeline

**Deployment:**
- Static files only (no server-side execution at request time)
- Atomic file writes prevent serving partial updates
- Snapshot-based fallback if upstream APIs fail

## Known Limitations

### No Application-Level Rate Limiting
This project relies on the hosting provider for rate limiting and DDoS protection. If deploying to your own infrastructure, implement appropriate rate limiting at the web server or CDN level.

### Public Deployment Details
Infrastructure and deployment patterns are documented for educational purposes. When adapting for production use:
- Change all default paths and configurations
- Implement additional security hardening for your environment
- Review and update security headers for your threat model

### Upstream Data Sources
The application trusts data from Madrid's open data portals (datos.madrid.es, esmadrid.com). These are assumed to be reliable, but:
- No cryptographic verification of upstream data
- Malicious or corrupted upstream data could affect site content
- Application implements basic validation but not deep content inspection

## Reporting Vulnerabilities

If you discover a security vulnerability in this project, please report it responsibly:

### For Security Issues

**DO:**
1. Report via [GitHub Security Advisories](https://github.com/ericphanson/plaza-espana-calendar/security/advisories/new)
2. Include detailed steps to reproduce the issue
3. Allow reasonable time for a fix before public disclosure (suggest 90 days)

**DO NOT:**
- Post security vulnerabilities in public issues
- Exploit vulnerabilities against the live production site
- Attempt to access systems or data beyond what's necessary to demonstrate the issue

### For Non-Security Issues

For bugs, feature requests, or general questions:
- Open a regular [GitHub issue](https://github.com/ericphanson/plaza-espana-calendar/issues)
- No need for private disclosure

## Supported Versions

This is a personal project with no formal support SLA. Security updates will be provided on a best-effort basis.

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| other   | :x:                |

## Security Best Practices for Deployment

If you're deploying this project to your own infrastructure:

### Recommended

1. **Use HTTPS:** Always serve over TLS/SSL
2. **Security headers:** Configure CSP, HSTS, X-Frame-Options, etc.
3. **Regular updates:** Keep Go toolchain and hosting environment updated
4. **Monitor logs:** Watch for unusual access patterns
5. **Backup strategy:** Maintain backups of configuration and historical data

### Optional

6. **Web Application Firewall:** Consider WAF for additional protection
7. **CDN:** Use CDN for DDoS protection and global performance
8. **Monitoring:** Set up uptime and security monitoring

## Dependencies

This project has **zero external Go dependencies** (standard library only), which:
- Reduces supply chain attack surface
- Simplifies security auditing
- Eliminates dependency vulnerability concerns

The only runtime dependencies are:
- Go standard library (keep Go version updated)
- Operating system and web server (keep patched)

## License and Warranty

This software is provided under the MIT License **without warranty**. See LICENSE file for details. You are responsible for security when deploying to production.

---

**Last Updated:** 2025-10-23
**Security Contact:** Via GitHub Security Advisories
