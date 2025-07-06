# Security Policy

## Project Status

finspect is currently in **alpha stage** and under heavy development. There are no stable releases yet, and the API/features may change significantly between commits.

**Note**: As this is alpha software, use at your own risk in production environments.

## Reporting a Vulnerability

I take security seriously. If you discover a security vulnerability, please report it through one of the following channels:

1. **GitHub Issues**: Create an issue with the security label
2. **Direct Contact**: Reach out through GitHub

Please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes

### Responsible Disclosure

I kindly ask that you:

- Allow me reasonable time to address the issue before public disclosure
- Avoid exploiting the vulnerability beyond what's necessary for verification
- Not access or modify other users' data

## Security Best Practices

When using finspect:

1. **File Permissions**: Be cautious when mounting directories with sensitive data
2. **Access Control**: Currently, finspect operates with the permissions of the user running it
3. **Network Operations**: Future cloud adaptors will use secure connections (HTTPS/TLS)
4. **Data Storage**: Local data is stored with filesystem permissions

## Future Security Enhancements

Planned security features:

- Encrypted blob storage
- OAuth2 for cloud adaptors
- Access control lists (ACLs) for shared deployments
- Audit logging

Thanks for helping keep finspect secure.
