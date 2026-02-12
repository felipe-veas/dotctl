# 4. Diseño del Manifest Declarativo

## Ubicación

```
<repo-root>/manifest.yaml
```

El manifest vive en el root del repo de dotfiles (no en el repo de `dotctl`). Es la fuente declarativa de qué se sincroniza y cómo.

## Schema

```yaml
# manifest.yaml — dotctl manifest
version: 1

# Variables globales reutilizables
vars:
  config_home: "~/.config"
  brew_bundle: "configs/brew/Brewfile"

# Reglas de sincronización
files:
  - source: configs/zsh/.zshrc
    target: ~/.zshrc

  - source: configs/zsh/.zprofile
    target: ~/.zprofile

  - source: configs/git/.gitconfig
    target: ~/.gitconfig

  - source: configs/git/.gitignore_global
    target: ~/.gitignore_global

  - source: configs/nvim
    target: "{{ .config_home }}/nvim"
    mode: symlink          # symlink (default) | copy

  - source: configs/starship.toml
    target: "{{ .config_home }}/starship.toml"

  # Reglas condicionales por OS
  - source: configs/brew/Brewfile
    target: "{{ .config_home }}/brew/Brewfile"
    when:
      os: darwin

  - source: configs/apt/packages.txt
    target: "{{ .config_home }}/apt/packages.txt"
    when:
      os: linux

  # Reglas condicionales por perfil
  - source: configs/wezterm/wezterm.lua
    target: "{{ .config_home }}/wezterm/wezterm.lua"
    when:
      profile: [macstudio, laptop]

  - source: configs/alacritty/alacritty.toml
    target: "{{ .config_home }}/alacritty/alacritty.toml"
    when:
      profile: laptop       # solo en el laptop

  # Archivo con secrets — cifrado con age/sops
  - source: configs/secrets/api_keys.enc.yaml
    target: "{{ .config_home }}/secrets/api_keys.yaml"
    mode: copy             # copy, no symlink (se descifra en destino)
    decrypt: true          # dotctl ejecutará sops -d antes de copiar
    when:
      os: darwin

# Archivos/patterns a NUNCA sincronizar (safety net)
ignore:
  - "*.token"
  - "*.pem"
  - "*.key"
  - ".env"
  - "id_rsa*"
  - "id_ed25519*"

# Hooks
hooks:
  post_sync:
    - command: brew bundle --file=configs/brew/Brewfile --no-lock
      when:
        os: darwin
    - command: configs/scripts/set-macos-defaults.sh
      when:
        os: darwin
        profile: macstudio

  pre_sync:
    - command: echo "Starting sync..."

  bootstrap:
    - command: /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
      description: "Install Homebrew"
      when:
        os: darwin
    - command: brew bundle --file=configs/brew/Brewfile
      description: "Install packages"
      when:
        os: darwin
    - command: configs/scripts/setup-defaults.sh
      description: "Set macOS defaults"
      when:
        os: darwin
        profile: macstudio
```

## Campos por Entrada

| Campo | Tipo | Default | Descripción |
|---|---|---|---|
| `source` | string | **requerido** | Path relativo al root del repo |
| `target` | string | **requerido** | Path absoluto o con `~` en la máquina destino |
| `mode` | `symlink` \| `copy` | `symlink` | Cómo aplicar el archivo |
| `when.os` | string \| []string | (todos) | `darwin`, `linux`, `windows` |
| `when.profile` | string \| []string | (todos) | Nombre(s) de perfil |
| `decrypt` | bool | `false` | Si es `true`, descifra con `sops`/`age` antes de aplicar |
| `backup` | bool | `true` | Hacer backup del archivo existente antes de reemplazar |

## Resolución de Variables

Las variables de `vars:` se resuelven con Go templates:
```
"{{ .config_home }}/nvim" → "~/.config/nvim"
```

Variables built-in (siempre disponibles):
- `{{ .home }}` → `$HOME`
- `{{ .os }}` → `runtime.GOOS`
- `{{ .profile }}` → perfil activo
- `{{ .hostname }}` → `os.Hostname()`

## Evaluación de `when`

```go
// Pseudocódigo
func shouldApply(entry Entry, ctx Context) bool {
    if entry.When.OS != "" && !matches(entry.When.OS, ctx.OS) {
        return false
    }
    if entry.When.Profile != "" && !matches(entry.When.Profile, ctx.Profile) {
        return false
    }
    return true
}
```

Lógica: todas las condiciones en `when` son AND. Si `when` no está presente, la regla aplica siempre.

## Estructura de Go

```go
// internal/manifest/types.go
package manifest

type Manifest struct {
    Version int               `yaml:"version"`
    Vars    map[string]string `yaml:"vars"`
    Files   []FileEntry       `yaml:"files"`
    Ignore  []string          `yaml:"ignore"`
    Hooks   HookSet           `yaml:"hooks"`
}

type FileEntry struct {
    Source  string    `yaml:"source"`
    Target string    `yaml:"target"`
    Mode   string    `yaml:"mode"`    // "symlink" | "copy"
    When   Condition `yaml:"when"`
    Decrypt bool     `yaml:"decrypt"`
    Backup  *bool    `yaml:"backup"`  // nil = default true
}

type Condition struct {
    OS      StringOrSlice `yaml:"os"`
    Profile StringOrSlice `yaml:"profile"`
}

type HookSet struct {
    PreSync   []Hook `yaml:"pre_sync"`
    PostSync  []Hook `yaml:"post_sync"`
    Bootstrap []Hook `yaml:"bootstrap"`
}

type Hook struct {
    Command     string    `yaml:"command"`
    Description string    `yaml:"description"`
    When        Condition `yaml:"when"`
}

// StringOrSlice permite "darwin" o ["darwin", "linux"]
type StringOrSlice []string
```

## Validaciones (en `dotctl doctor` y antes de sync)

1. Todos los `source` existen en el repo
2. No hay targets duplicados
3. Ningún `source` matchea patterns de `ignore`
4. Variables referenciadas en templates existen en `vars`
5. `mode` es "symlink" o "copy"
6. Si `decrypt: true`, verificar que `sops` o `age` está en PATH
