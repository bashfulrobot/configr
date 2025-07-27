# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| develop | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in configr, please report it responsibly:

### Private Disclosure
1. **DO NOT** open a public issue for security vulnerabilities
2. Email the maintainers directly with details
3. Include steps to reproduce the vulnerability
4. Provide any relevant configuration files (sanitized)

### What to Include
- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (if available)
- Your contact information

### Response Timeline
- Initial response: Within 48 hours
- Status update: Within 7 days
- Resolution timeline: Depends on severity

## Security Features

Configr includes several security features:

### Input Validation
- Comprehensive validation of configuration files
- Path traversal protection
- Package name sanitization
- File permission validation

### Safe Operations
- Dry-run mode for testing
- Backup system for file operations
- Permission checks before modifications
- Command validation before execution

### Monitoring
- Automated security scanning with gosec
- Vulnerability checking with govulncheck
- Static analysis with staticcheck
- SARIF report generation for GitHub Security

## Security Best Practices

When using configr:

1. **Review configurations** before applying
2. **Use dry-run mode** to preview changes
3. **Validate file permissions** are appropriate
4. **Keep configr updated** to latest version
5. **Review security advisories** regularly

## Automated Security Scanning

This repository uses automated security scanning:

- **gosec**: Go security checker
- **govulncheck**: Go vulnerability database
- **staticcheck**: Static analysis
- **nancy**: Dependency vulnerability scanner
- **semgrep**: Additional security analysis

Results are available in the GitHub Security tab.

## Security Contact

For security-related questions or concerns, please contact the maintainers through the established communication channels.