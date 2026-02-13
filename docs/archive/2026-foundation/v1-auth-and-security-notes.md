# V1 Auth and Security Notes

## Auth strategy

- Preferred: HTTPS + `gh` authentication.
- Fully supported: SSH repository URLs.
- Goal: avoid custom OAuth token storage logic in dotctl.

## Security principles

- Do not persist plaintext GitHub tokens in dotctl config.
- Support sensitive file guardrails (`ignore` patterns, doctor checks).
- Keep encrypted file support explicit (`decrypt: true` + `copy` mode).

## Operational posture

- Clear user-facing remediation for auth failures.
- Secure defaults with explicit opt-in where risk is higher.
