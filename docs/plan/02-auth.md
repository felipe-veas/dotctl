# 2. Flujo de Autenticación GitHub

## Recomendación: `gh` CLI como auth provider

```
dotctl init
  │
  ▼
¿gh instalado?  ──NO──→ Error: "Instala gh: brew install gh (macOS) / apt install gh (Linux)"
  │
  YES
  ▼
¿gh auth status OK?  ──NO──→ Ejecuta: gh auth login --web
  │
  YES
  ▼
token = $(gh auth token)
  │
  ▼
git clone https://github.com/<user>/<repo>.git
  (gh configura git credential helper automáticamente)
```

### Por qué `gh` CLI

1. **Ya resuelve OAuth device flow** — el usuario hace login una vez, `gh` almacena el token en el credential store nativo del OS.
2. **Cross-platform credential storage**:
   - macOS → Keychain
   - Linux → libsecret (GNOME Keyring / KDE Wallet) o archivo cifrado
3. **No tocamos tokens** — nunca leemos, imprimimos ni almacenamos tokens. Solo verificamos que `gh auth status` sea exitoso.
4. **Git credential helper** — `gh` se registra como credential helper de Git, así `git push/pull` funciona transparentemente.
5. **Scope mínimo** — `gh auth login` pide solo los scopes necesarios (repo access).

### Implementación en Go

```go
// internal/auth/gh.go
package auth

import (
    "fmt"
    "os/exec"
    "runtime"
    "strings"
)

func EnsureGHAuth() error {
    // Verificar que gh está instalado
    if _, err := exec.LookPath("gh"); err != nil {
        hint := "brew install gh"
        if runtime.GOOS == "linux" {
            hint = "https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
        }
        return fmt.Errorf("gh CLI not found — install: %s", hint)
    }

    // Verificar auth status
    cmd := exec.Command("gh", "auth", "status")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("gh not authenticated — run: gh auth login --web")
    }

    return nil
}

func GetGitProtocol() (string, error) {
    out, err := exec.Command("gh", "config", "get", "git_protocol").Output()
    if err != nil {
        return "https", nil // default
    }
    return strings.TrimSpace(string(out)), nil
}
```

### Flujo completo de `dotctl init`

```
$ dotctl init --repo github.com/felipe-veas/dotfiles --profile macstudio

✓ gh CLI found
✓ gh authenticated as felipe-veas
✓ Cloned repo to ~/.config/dotctl/repo
✓ Profile set to: macstudio
✓ Config written to ~/.config/dotctl/config.yaml

Run 'dotctl sync' to apply your dotfiles.
```

## Alternativas Evaluadas

### SSH Keys

| Aspecto | Evaluación |
|---|---|
| Setup | Requiere generar key, agregar a GitHub, configurar `~/.ssh/config` |
| UX | Más pasos manuales para el usuario |
| Seguridad | Buena (key en disco protegida por passphrase + ssh-agent) |
| Cross-platform | Funciona idéntico en macOS y Linux |
| Veredicto | **Viable como fallback**, pero más fricción que `gh` |

Si el usuario prefiere SSH, `dotctl init` acepta URL SSH:
```
$ dotctl init --repo git@github.com:felipe-veas/dotfiles.git
```
No requiere `gh` en este caso — Git usa SSH agent directamente.

### Personal Access Token (PAT)

| Aspecto | Evaluación |
|---|---|
| Setup | Generar en GitHub Settings → Developer → Tokens |
| UX | Token es un string largo que el usuario debe pegar |
| Seguridad | **Riesgo**: el usuario podría guardarlo en texto plano |
| Veredicto | **No recomendado para MVP**. Si se soporta, almacenar en credential store nativo |

### Git Credential Manager (GCM)

| Aspecto | Evaluación |
|---|---|
| Setup | Viene con Git for macOS. Disponible como paquete para Linux |
| UX | Transparente si ya está configurado |
| Seguridad | Buena (Keychain en macOS, libsecret en Linux) |
| Veredicto | **Funciona automáticamente** si `gh` no está. No requiere código extra |

## Decisión Final

```
Primario:  gh CLI (OAuth device flow) → 90% de los casos. Funciona en macOS y Linux
Fallback:  SSH keys → para usuarios que prefieren SSH. Cross-platform nativo
Implícito: GCM → si Git ya tiene credential helper, funciona sin código adicional
Rechazado: PAT manual → riesgo de exposición
```

## Credential Storage por OS

| OS | `gh` credential backend | Ubicación |
|---|---|---|
| macOS | macOS Keychain | Gestionado por Security framework |
| Linux (GNOME) | libsecret / GNOME Keyring | `~/.local/share/keyrings/` |
| Linux (KDE) | libsecret / KDE Wallet | KWallet |
| Linux (headless) | Archivo cifrado | `~/.config/gh/hosts.yml` (cifrado por gh) |

`gh` detecta automáticamente el backend disponible. No requiere configuración por parte de `dotctl`.

## Requisitos de Seguridad

| Requisito | Implementación |
|---|---|
| Nunca imprimir tokens en stdout/stderr | No ejecutamos `gh auth token` — solo `gh auth status` para validar |
| Storage seguro | Delegado a `gh` → Keychain (macOS) / libsecret (Linux) |
| Tokens no en el repo | `.gitignore` incluye `*.token`, `.env`, config local |
| Logs sin tokens | El logger redacta cualquier string que parezca token (regex: `gh[ps]_[A-Za-z0-9]+`) |
| SSH keys no en repo | Validación en `doctor`: si `~/.ssh/id_*` aparece en manifest → error |
