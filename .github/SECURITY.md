# Security policy

## Supported versions

Security fixes are applied to the latest release on the `master` branch and backported to the most recent tagged release when practical.

| Version | Supported |
| --- | --- |
| Latest `master` | ✅ |
| Latest tagged release (`v*`) | ✅ |
| Older releases | ❌ |

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security issue in VietClaw, report it privately:

1. **Preferred:** [Open a GitHub Security Advisory](https://github.com/vietclaw/vietclaw/security/advisories/new) (Private vulnerability report).
2. **Alternative:** Contact the repository maintainers through GitHub (see repository **About** → owners).

Include as much detail as possible:

- Description of the vulnerability and impact
- Steps to reproduce
- Affected versions or commits
- Proof-of-concept if available (please minimize harm to third parties)

We aim to acknowledge reports within **72 hours** and provide an initial assessment within **7 days**.

## What to expect

- We will work with you to understand and validate the issue.
- We will develop a fix and coordinate disclosure timing.
- We will credit reporters in the release notes or advisory when they wish (unless you prefer to remain anonymous).

## Scope

The following are **in scope** for this policy:

- The VietClaw daemon, CLI, and embedded web UI served by the binary
- Default tool policies (`shell_exec`, file workspace boundaries, network policy)
- Channel adapters (Discord, Telegram) as shipped in this repository
- Supply chain of dependencies declared in `go.mod` and `apps/web/package.json`

The following are generally **out of scope**:

- Misconfiguration by operators (e.g. enabling `shell_exec` on a shared host, exposing the daemon to the public internet without authentication)
- Vulnerabilities in third-party LLM providers or their APIs
- Social engineering against end users

## Secure defaults

VietClaw ships with conservative defaults:

- `tools.shell.enabled: false`
- File tools scoped to workspace when `workspace_only` is true
- Shell network policy blocks common metadata / private endpoints

Operators who enable shell execution or widen file/network access assume additional responsibility for host security.

## Safe harbor

We support good-faith security research that follows this policy. We will not pursue legal action against researchers who:

- Make a good-faith effort to avoid privacy violations, destruction of data, and service degradation
- Report findings promptly and allow reasonable time for remediation before public disclosure
