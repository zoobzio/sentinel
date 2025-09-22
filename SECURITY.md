# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

1. **DO NOT** create a public GitHub issue
2. Email security details to the maintainers
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

## Security Best Practices

When using Sentinel:

1. **Metadata Exposure**: Be aware that Sentinel extracts and caches struct metadata. Ensure sensitive field names or tags don't leak information.

2. **Tag Values**: Struct tags may contain sensitive information. Review your tags carefully:
   - Don't include credentials in tags
   - Be cautious with `example` tags
   - Review `desc` tags for information disclosure

3. **Caching**: Sentinel uses permanent caching. Ensure your application doesn't expose the cache to untrusted sources.

4. **Relationship Discovery**: The relationship extraction feature reveals connections between types. Consider if this information should be protected in your application.

## Security Features

Sentinel is designed with security in mind:

- Zero external dependencies (reduces supply chain risks)
- No network operations
- No file system operations beyond normal Go imports
- Read-only metadata extraction
- Thread-safe caching

## Acknowledgments

We appreciate responsible disclosure of security vulnerabilities.