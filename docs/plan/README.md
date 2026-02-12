# dotctl — Plan de Implementación

## Índice

| # | Documento | Contenido |
|---|---|---|
| 0 | [Resumen Ejecutivo](./00-executive-summary.md) | Qué hace, qué no hace, stack |
| 1 | [Arquitectura](./01-architecture.md) | Componentes, responsabilidades, diagrama ASCII, flujo de datos |
| 2 | [Autenticación](./02-auth.md) | gh CLI (recomendado), SSH, PAT, GCM. Requisitos de seguridad |
| 3 | [CLI Design](./03-cli-design.md) | Subcomandos, flags, output humano/JSON, exit codes, snippets Go |
| 4 | [Manifest](./04-manifest.md) | Schema YAML, ejemplos, condiciones, hooks, tipos Go |
| 5 | [Estructura del Repo](./05-repo-structure.md) | Directorios, Makefile, CI, linting |
| 6 | [Tray App (macOS + Linux)](./06-menubar.md) | macOS: Swift NSStatusBar, Linux: Go systray, diseño UI, comunicación, empaquetado, autostart |
| 7 | [Milestones](./07-milestones.md) | M0→M4, criterios de éxito por hito, backlog post-MVP |
| 8 | [Riesgos y Tradeoffs](./08-risks.md) | Decisiones técnicas con rationale |
| 9 | [Criterios de Aceptación](./09-acceptance-criteria.md) | Checklist completa para dar el MVP por listo |
