# Changelog

<!-- markdownlint-disable MD024 -->

## [1.9.0](https://github.com/felipe-veas/dotctl/compare/v1.8.0...v1.9.0) (2026-02-13)

### Features

* **manifest:** automatically copy suggested config files to repo ([a1ecdb6](https://github.com/felipe-veas/dotctl/commit/a1ecdb6d2d992bc848180f59e5ec60be03f7dd9e))
* **manifest:** automatically copy suggested config files to repo ([209efe8](https://github.com/felipe-veas/dotctl/commit/209efe8001b30157aba7d80584a29516cd5161d9))

## [1.8.0](https://github.com/felipe-veas/dotctl/compare/v1.7.1...v1.8.0) (2026-02-13)

### Features

* add secrets management with age encryption ([7ae1dd1](https://github.com/felipe-veas/dotctl/commit/7ae1dd1b69ca9978d172bb0efe0396b3c6afb9ad))
* add secrets management with age encryption ([b81d3ce](https://github.com/felipe-veas/dotctl/commit/b81d3ceee04ce11ad43ee02c539a8f9784fd12f5))

## [1.7.1](https://github.com/felipe-veas/dotctl/compare/v1.7.0...v1.7.1) (2026-02-13)

### Bug Fixes

* **aur:** address shellcheck warnings and improve error handling ([c4fcc62](https://github.com/felipe-veas/dotctl/commit/c4fcc62c7fa5a43a2b929b0fa924b603bcd88f4f))

## [1.7.0](https://github.com/felipe-veas/dotctl/compare/v1.6.0...v1.7.0) (2026-02-13)

### Features

* add manifest suggest command and update documentation ([7337348](https://github.com/felipe-veas/dotctl/commit/7337348f4d8beb46871b111f2895b37d5634ff4d))
* **cmd:** add manifest suggest command ([b05a6ad](https://github.com/felipe-veas/dotctl/commit/b05a6ad6f6cf5a0993269fd296b03dd17b9d14ee))
* **gitops:** use local git identity for push ([46c4f87](https://github.com/felipe-veas/dotctl/commit/46c4f873657c1a886be36b60519a52ecbfba3c50))

## [1.6.0](https://github.com/felipe-veas/dotctl/compare/v1.5.0...v1.6.0) (2026-02-13)

### Features

* **cmd:** warn when pushing from a different dirty repo ([704ffe9](https://github.com/felipe-veas/dotctl/commit/704ffe993d08b816435457be89fc873c5d6d6990))
* **cmd:** warn when pushing from a different dirty repo ([27c86be](https://github.com/felipe-veas/dotctl/commit/27c86bebcc86fb8bcd0d7805ae310e949c477609))

## [1.5.0](https://github.com/felipe-veas/dotctl/compare/v1.4.0...v1.5.0) (2026-02-12)

### Features

* **backup:** implement backup rotation to limit stored snapshots ([8d557e7](https://github.com/felipe-veas/dotctl/commit/8d557e712923f931f907e58125629ef11a0b139e))
* **cli:** enhance multi-repo support in doctor and tests ([1c30e86](https://github.com/felipe-veas/dotctl/commit/1c30e86eafc48f815c916141c85365c3ff78f990))
* **cmd:** add diff and watch commands ([8c78d1b](https://github.com/felipe-veas/dotctl/commit/8c78d1b556d2c6ac009092c56eca36253d8cce0c))
* Multi-repo enhancements, watch command, and packaging ([bb37843](https://github.com/felipe-veas/dotctl/commit/bb3784399821613e05dc6b9db6c56d2b4b6471a9))

## [1.4.0](https://github.com/felipe-veas/dotctl/compare/v1.3.0...v1.4.0) (2026-02-12)

### Features

* **core:** implement file decryption with age/sops support ([58713fd](https://github.com/felipe-veas/dotctl/commit/58713fd29bad5d8ddbd6fd8b661426ee2f49abe4))
* post-MVP features (decryption, notifications, packaging) ([080f09e](https://github.com/felipe-veas/dotctl/commit/080f09e6675f5454c75746372f5087ca339ea926))
* **tray:** implement native notifications for linux ([b32955c](https://github.com/felipe-veas/dotctl/commit/b32955c86eaf0e9d5f76f0573dd9f006c411e121))
* **tray:** implement native notifications for macOS ([8735458](https://github.com/felipe-veas/dotctl/commit/8735458b05e029761e7929bc381eecd8e7e9c921))

## [1.3.0](https://github.com/felipe-veas/dotctl/compare/v1.2.0...v1.3.0) (2026-02-12)

### Features

* **cmd:** enhance doctor with gitignore checks ([91b5d17](https://github.com/felipe-veas/dotctl/commit/91b5d179ac05f3030dcfcbc2d91fd57c3d65e3d5))
* **core:** implement file locking and sync rollback ([da340c7](https://github.com/felipe-veas/dotctl/commit/da340c7172e50ba8e1d7a8b221acf260a6de6878))
* **core:** implement logging, verbose mode, and gitops tracing ([68d1290](https://github.com/felipe-veas/dotctl/commit/68d129007a9d377f11b121584f40897bd3540d39))
* **core:** improve auth error hints ([1ed3721](https://github.com/felipe-veas/dotctl/commit/1ed3721195a6ad0517abb9082d0dba226c882de5))
* **linker:** add human-friendly filesystem error messages ([ca12deb](https://github.com/felipe-veas/dotctl/commit/ca12debcca75a1339af28ad26cf62fc4aa760a65))
* M4 hardening (logging, locking, rollback, security) ([9384f01](https://github.com/felipe-veas/dotctl/commit/9384f01fd192c30023dddfa36ba33c02201a6a7d))
* **manifest:** support ignore patterns filtering ([c91e559](https://github.com/felipe-veas/dotctl/commit/c91e55913a44e12335b78b5ca2e88d01db8de373))

### Bug Fixes

* **cmd:** add logging to hooks and integration tests ([59ab425](https://github.com/felipe-veas/dotctl/commit/59ab425ef1cd552c42168973d3ec30ba473b58c3))

## [1.2.0](https://github.com/felipe-veas/dotctl/compare/v1.1.0...v1.2.0) (2026-02-12)

### Features

* **cmd:** implement hooks and bootstrap command ([971e863](https://github.com/felipe-veas/dotctl/commit/971e863e4f82774556083d5ec105bc32c8929286))
* implement tray apps and hooks ([dc44d97](https://github.com/felipe-veas/dotctl/commit/dc44d979148e99b161f65b82e9fcefd1d57ad7cd))
* **tray:** implement linux tray app ([e3b6a78](https://github.com/felipe-veas/dotctl/commit/e3b6a78185e7d0f157e40e9737cfa2e96907a238))
* **tray:** scaffold macos status app ([c268844](https://github.com/felipe-veas/dotctl/commit/c26884478b5a24120b232a248618bdca45186bd1))

## [1.1.0](https://github.com/felipe-veas/dotctl/compare/v1.0.0...v1.1.0) (2026-02-12)

### Features

* **cmd:** implement git integration, doctor, and status commands ([8c3ec38](https://github.com/felipe-veas/dotctl/commit/8c3ec3896d1b0abac9584e22a9b6fe223bdd35a6))
* **core:** implement auth and gitops packages ([d1abfbc](https://github.com/felipe-veas/dotctl/commit/d1abfbc5a686579f4305b42e7e06519244999b7e))
* implement git operations, auth, and doctor commands ([7906b7d](https://github.com/felipe-veas/dotctl/commit/7906b7d428a968017f8de000c406586998eee4f2))

## 1.0.0 (2026-02-12)

### Features

* implement MVP core functionality ([5ee2d6f](https://github.com/felipe-veas/dotctl/commit/5ee2d6f7d4c340418d388aa996d038508b2af376))
* MVP implementation of dotctl ([ccceb07](https://github.com/felipe-veas/dotctl/commit/ccceb07fca1450e34ab1497aafd4c0bb51c36c69))
