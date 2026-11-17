# Security Policy

## Supported Versions

Security fixes are provided for the latest release of krot. Please upgrade to
the latest version before reporting an issue. Older releases are not patched
separately.

## Reporting a Vulnerability

Please do not disclose potential vulnerabilities in public GitHub issues.

Use [GitHub's private vulnerability reporting](https://github.com/snmp161/krot/security/advisories/new)
to report a vulnerability privately.

Please include:

- the affected krot version or commit;
- operating system and architecture;
- relevant command-line options;
- steps to reproduce the issue;
- expected and actual behavior;
- potential security impact.

Do not include private keys or other sensitive configuration data. Redact IP
addresses and credentials from logs and configurations unless they are strictly
necessary to reproduce the issue.

## Scope

The UDP listener/checker protocol used by `--udp-listener` and `--udp-checker`
is intentionally unencrypted. Its lack of encryption alone is not considered a
vulnerability.
