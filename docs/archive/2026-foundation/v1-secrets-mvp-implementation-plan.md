# Secrets MVP Implementation Plan (Historical)

Reference design:

- [Secrets design](../../secrets-design.md)

Status:

- Implemented
- Historical implementation checklist retained for traceability

## Phase 1: Dependency and base types

- [x] Add `filippo.io/age` dependency in `go.mod`.
- [x] Create `internal/secrets/secrets.go` with public types:
  - `Identity` (`PrivatePath`, `PublicKey`)
  - `InitOptions`, `EncryptOptions`, `DecryptOptions`, `RotateOptions`
  - `Status` (`Identity`, `RecipientFile`, `EncryptedFiles`, `UnprotectedFiles`)
  - `RotateResult`
  - constants for defaults and sensitive patterns

## Phase 2: Identity management

- [x] Create `internal/secrets/identity.go`:
  - `DefaultIdentityPath()`
  - `FindIdentity(configDir)`
  - `FindRecipient(repoRoot)`
  - `WriteRecipientFile(repoRoot, publicKey)`
- [x] Add `internal/secrets/identity_test.go`.

## Phase 3: age backend (encrypt/decrypt)

- [x] Create `internal/secrets/age_backend.go`:
  - `GenerateIdentity(path)`
  - `EncryptFile(inputPath, recipientPubKey)`
  - `DecryptFile(inputPath, identityPath)`
- [x] Add `internal/secrets/age_backend_test.go` roundtrip tests.

## Phase 4: Public package API

- [x] Implement `Init(repoRoot, opts)` in `init.go`.
- [x] Implement `Encrypt(repoRoot, filePath, opts)` in `encrypt.go`.
- [x] Implement `Decrypt(filePath, opts)` in `encrypt.go`.
- [x] Implement `Rotate(repoRoot, opts)` in `rotate.go`.
- [x] Implement `GetStatus(repoRoot)` in `status.go`.
- [x] Add package tests: `init_test.go`, `encrypt_test.go`, `rotate_test.go`, `status_test.go`.

## Phase 5: Cobra commands

- [x] Create `internal/cmd/secrets.go` with:
  - `dotctl secrets`
  - `secrets init`
  - `secrets encrypt`
  - `secrets decrypt`
  - `secrets status`
  - `secrets rotate`
- [x] Register `newSecretsCmd()` in `internal/cmd/root.go`.

## Phase 6: Integration with existing flows

- [x] Add push preflight check in `internal/cmd/push.go`.
- [x] Add secrets section in `internal/cmd/doctor.go`.
- [x] Enforce restrictive decrypted output permissions in `internal/linker/linker.go`.
- [x] Skip `.enc.` files in `security_checks.go` sensitive checks.

## Phase 7: File naming helpers

- [x] Add `EncryptedName(original)` and `DecryptedName(encrypted)`.
- [x] Cover edge cases with tests and `IsSensitiveName` checks.

## Phase 8: Integration tests

- [x] E2E: init -> encrypt -> decrypt roundtrip.
- [x] E2E: rotate re-encrypt flow.
- [x] E2E: status detects unprotected sensitive files.
- [x] E2E: push preflight blocks unencrypted sensitive files.
- [x] Validate `dotctl sync` with `decrypt: true` remains stable.

## Phase 9: Documentation

- [x] Update `docs/security-model.md`.
- [x] Update `docs/command-reference.md`.
- [x] Keep `docs/manifest-spec.md` unchanged (`decrypt: true` already existed).
- [x] Keep dedicated design documentation.

## Dependency order (implemented)

```text
Phase 1
  -> Phase 2
  -> Phase 7
    -> Phase 3
      -> Phase 4
        -> Phase 5
        -> Phase 6
          -> Phase 8
            -> Phase 9
```

## Files created or modified

### New (`internal/secrets/`)

- `secrets.go` (public types and constants)
- `helpers.go`
- `identity.go`
- `age_backend.go`
- `naming.go`
- `init.go`
- `encrypt.go`
- `rotate.go`
- `status.go`
- `*_test.go`

### New (`internal/cmd/`)

- `secrets.go` (cobra commands)

### Modified

- `go.mod`, `go.sum`
- `internal/cmd/root.go`
- `internal/cmd/push.go`
- `internal/cmd/doctor.go`
- `internal/cmd/security_checks.go`
- `internal/linker/linker.go`
