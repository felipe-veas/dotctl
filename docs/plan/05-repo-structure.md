# 5. Estructura del Repositorio

## Repo de dotctl (esta herramienta)

```
dotctl/
├── cmd/                          # Entry points (cobra commands)
│   └── dotctl/
│       └── main.go               # func main()
│
├── internal/                     # Lógica privada (no exportable)
│   ├── auth/
│   │   ├── gh.go                 # gh CLI integration
│   │   └── gh_test.go
│   ├── backup/
│   │   ├── backup.go             # Backup de archivos antes de overwrite
│   │   └── backup_test.go
│   ├── config/
│   │   ├── config.go             # ~/.config/dotctl/config.yaml
│   │   └── config_test.go
│   ├── doctor/
│   │   ├── doctor.go             # Health checks
│   │   └── doctor_test.go
│   ├── gitops/
│   │   ├── git.go                # git clone/pull/push/status
│   │   └── git_test.go
│   ├── linker/
│   │   ├── linker.go             # Symlink creation + drift detection
│   │   └── linker_test.go
│   ├── manifest/
│   │   ├── types.go              # Manifest structs
│   │   ├── parser.go             # YAML parsing + template resolution
│   │   ├── resolver.go           # Condition evaluation (when)
│   │   ├── parser_test.go
│   │   └── resolver_test.go
│   ├── output/
│   │   ├── printer.go            # Human-readable output
│   │   └── json.go               # JSON output (--json)
│   ├── platform/
│   │   ├── platform.go           # Interface + OS detection
│   │   ├── darwin.go             # macOS: open, paths, Keychain hints
│   │   ├── linux.go              # Linux: xdg-open, XDG paths
│   │   └── platform_test.go
│   └── profile/
│       ├── profile.go            # Profile resolution (OS, arch, hostname)
│       └── profile_test.go
│
├── pkg/                          # Exportable (si se necesita como library)
│   └── types/
│       └── status.go             # Tipos compartidos (StatusResponse, etc.)
│
├── mac/                          # macOS menubar app (Swift)
│   └── StatusApp/
│       ├── StatusApp.xcodeproj/
│       ├── StatusApp/
│       │   ├── AppDelegate.swift
│       │   ├── StatusBarController.swift
│       │   ├── DotctlBridge.swift    # Ejecuta binario + parsea JSON
│       │   ├── Assets.xcassets/
│       │   └── Info.plist
│       └── bin/                      # Binario Go (copiado en build)
│           └── .gitkeep
│
├── linux/                        # Linux tray app (Go)
│   └── tray/
│       ├── main.go               # Entry point tray app Linux
│       ├── tray.go               # systray setup + menu items
│       ├── bridge.go             # Ejecuta dotctl binary + parsea JSON
│       └── assets/
│           ├── icon-ok.png       # 22x22 tray icons
│           ├── icon-warn.png
│           └── icon-error.png
│
├── scripts/
│   ├── build.sh                  # Build CLI para OS/arch actual
│   ├── build-app-macos.sh        # Build universal macOS binary + Swift app
│   ├── build-tray-linux.sh       # Build Linux tray app
│   ├── install.sh                # Installer (curl | bash friendly)
│   └── release.sh                # goreleaser wrapper
│
├── testdata/                     # Fixtures para tests
│   ├── manifest_valid.yaml
│   ├── manifest_invalid.yaml
│   └── manifest_multiprofile.yaml
│
├── docs/
│   └── plan/                     # Este plan de implementación
│       ├── 00-executive-summary.md
│       ├── 01-architecture.md
│       ├── ...
│       └── 09-acceptance-criteria.md
│
├── .github/
│   └── workflows/
│       ├── ci.yaml               # Test + lint en PRs (macOS + Linux matrix)
│       └── release.yaml          # goreleaser en tags
│
├── .goreleaser.yaml              # Config de goreleaser (multi-OS/arch)
├── .golangci.yml                 # Config de golangci-lint
├── go.mod
├── go.sum
├── Makefile                      # Shortcuts: make build, test, lint
├── .gitignore
├── LICENSE
└── README.md
```

## Repo de dotfiles del usuario (ejemplo)

```
dotfiles/                         # Repo privado del usuario
├── manifest.yaml                 # Manifest declarativo
├── configs/
│   ├── zsh/
│   │   ├── .zshrc
│   │   └── .zprofile
│   ├── bash/                     # Para máquinas Linux que usen bash
│   │   └── .bashrc
│   ├── git/
│   │   ├── .gitconfig
│   │   └── .gitignore_global
│   ├── nvim/
│   │   ├── init.lua
│   │   └── lua/
│   │       └── ...
│   ├── starship.toml
│   ├── wezterm/
│   │   └── wezterm.lua
│   ├── alacritty/
│   │   └── alacritty.toml
│   ├── brew/                     # macOS only
│   │   └── Brewfile
│   ├── apt/                      # Linux only
│   │   └── packages.txt
│   ├── systemd/                  # Linux only
│   │   └── user/
│   │       └── dotctl-sync.timer
│   └── secrets/                  # Archivos cifrados
│       └── api_keys.enc.yaml
├── scripts/
│   ├── setup-macos-defaults.sh   # macOS only
│   ├── setup-linux.sh            # Linux only
│   └── install-tools.sh          # Cross-platform
└── .gitignore                    # Ignora *.token, .env, etc.
```

## Makefile

```makefile
.PHONY: build test lint clean

BINARY := dotctl
VERSION := $(shell git describe --tags --always --dirty)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.goos=$(GOOS) -X main.goarch=$(GOARCH)" \
		-o bin/$(BINARY) ./cmd/dotctl

# macOS universal binary (arm64 + amd64)
build-universal:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY)-arm64 ./cmd/dotctl
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY)-amd64 ./cmd/dotctl
	lipo -create bin/$(BINARY)-arm64 bin/$(BINARY)-amd64 -output bin/$(BINARY)-universal
	rm bin/$(BINARY)-arm64 bin/$(BINARY)-amd64

# Linux builds
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY)-linux-amd64 ./cmd/dotctl

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY)-linux-arm64 ./cmd/dotctl

# Linux tray app (requires libayatana-appindicator3-dev on build machine)
build-tray-linux:
	cd linux/tray && CGO_ENABLED=1 go build -o ../../bin/dotctl-tray ./

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY) /usr/local/bin/$(BINARY)

# macOS app bundle
app-macos: build-universal
	cp bin/$(BINARY)-universal mac/StatusApp/bin/dotctl
	cd mac/StatusApp && xcodebuild -scheme StatusApp -configuration Release build
```

## CI (.github/workflows/ci.yaml)

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    strategy:
      matrix:
        os: [macos-latest, ubuntu-latest]
        go-version: ['1.22']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
      - run: go test ./... -race -count=1
      - run: go vet ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

## .goreleaser.yaml

```yaml
version: 2

builds:
  - id: dotctl
    main: ./cmd/dotctl
    binary: dotctl
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}

  - id: dotctl-tray-linux
    main: ./linux/tray
    binary: dotctl-tray
    env:
      - CGO_ENABLED=1
    goos:
      - linux
    goarch:
      - amd64
    ldflags:
      - -s -w

universal_binaries:
  - id: dotctl-universal
    ids:
      - dotctl
    replace: false
    name_template: "dotctl-universal"

archives:
  - id: default
    builds:
      - dotctl
    format: tar.gz
    name_template: "dotctl_{{ .Os }}_{{ .Arch }}"

nfpms:
  - id: linux-packages
    builds:
      - dotctl
      - dotctl-tray-linux
    formats:
      - deb
      - rpm
    package_name: dotctl
    description: Sync dotfiles across machines
    maintainer: felipe-veas
    contents:
      - src: linux/tray/assets/dotctl.desktop
        dst: /usr/share/applications/dotctl-tray.desktop
        type: config

brews:
  - ids:
      - default
    repository:
      owner: felipe-veas
      name: homebrew-tap
    homepage: https://github.com/felipe-veas/dotctl
    description: Sync dotfiles across machines
```

## .golangci.yml (mínimo)

```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - gosimple

linters-settings:
  errcheck:
    check-type-assertions: true

issues:
  exclude-use-default: false
```
