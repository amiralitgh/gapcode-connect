# Security

GapCode Connect connects internet-facing chat accounts to a local coding
agent. A compromised bot token or overly broad allowlist can therefore provide
access to the configured project and, when enabled, the host shell.

## Deployment guidance

- Store Telegram and Bale tokens in environment variables or a secret manager.
- Never commit `config.toml`, `.env`, logs, session data, or private keys.
- Set each platform's `allow_from` to explicit trusted user IDs.
- Set project-level `admin_from` to explicit trusted user IDs.
- Leave `admin_from` unset unless privileged commands are required.
- Do not use `admin_from = "*"` on shared or public deployments.
- Keep the management web server bound to localhost unless it is protected by
  authentication and a trusted network boundary.
- Treat the configured project directory and agent mode as sensitive.
- Rotate a bot token immediately if it appears in logs, shell history, an issue,
  or a chat.

## Reporting a vulnerability

Please do not publish credentials or an exploitable proof of concept in a
public issue. Use GitHub's private vulnerability reporting for this repository
when available. Otherwise, contact the repository owner privately through
GitHub and include:

1. affected version or commit;
2. reproduction steps that do not expose secrets;
3. impact and required permissions;
4. any suggested mitigation.

Security fixes should include a regression test where practical.
