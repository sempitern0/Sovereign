# Estado de Sovereign

Fuente única del estado del proyecto y de la **siguiente acción**. Actualizarla al final de cada iteración (ver «Registro de iteración»). El chat o la memoria de un agente no son el registro del proyecto.

Última actualización: 2026-11-11.

## Hitos

Orden según [ADR-0005](architecture/adr/0005-roadmap-release-first.md). Estados: pendiente · en curso · en revisión · completado · bloqueado.

| Hito | Estado | Evidencia |
| --- | --- | --- |
| H0 — Fundación | Completado | Esquema v1, validador Python y CI Windows/Linux |
| H1 — Planificación | Completado | `plan` determinista y JSON v1; tests de alternativas, dependencias e incompatibilidades |
| H2 — Primera instalación (LocalSend) | Completado | PR #2 y PR #3 fusionadas. CI de PR #3 en verde en ubuntu-latest y windows-latest (Go, catálogo). **El usuario confirma pruebas reales (2026-11-11)**; no se han adjuntado registros |
| H3 — Perfil esencial instalable | Pendiente | — |
| H4 — Primera release distribuible | Pendiente | — |
| H5 — Datos y portabilidad | Pendiente | — |
| H6 — Android guiado | Pendiente | — |
| H7 — Kits offline | Pendiente | — |
| H8 — Conector Multisuite opcional | Pendiente; prerrequisitos externos | Multisuite no expone API ni acceso VPN todavía (su hito P03) |
| H9 — UX y guías | Pendiente | — |

## Qué funciona hoy

- `suitectl doctor | catalog list | profiles list | plan` (solo lectura) y `verify` / `apply --app localsend --yes`.
- Funciona desde cualquier carpeta: catálogo, perfiles y allowlist van incrustados. `--root <checkout>` lee una copia de desarrollo; la allowlist nunca se lee de disco.
- Instalable: **solo LocalSend** (WinGet en Windows, Flatpak en Linux). VLC y LibreOffice: `pending`. Android: no soportado aún.
- Uso: con terminal, el gestor muestra progreso y avisos; sin terminal o con `--non-interactive`, WinGet no pregunta y falla en vez de colgarse.

## Siguiente acción

1. **H3-1:** verificar VLC en WinGet y Flathub (IDs y fuentes oficiales con fecha), marcarlo `verified`, añadir sus entradas a `catalog/install-allowlist.yaml`, tests simulados y aceptación en dispositivo real. Ver backlog H3 en el [plan §10](../PLAN_TECNICO_SUITE_LIBRE.md#10-hoja-de-ruta-y-entregables).

## Limitaciones conocidas

- La release distribuible no existe todavía (H4): usar el bootstrap con Go de usuario o compilar.
- `bootstrap --offline` falla si el módulo `gopkg.in/yaml.v3` no está en la caché de Go; se resolverá vendorizando en H4.
- La QA real del bootstrap de Go en equipos limpios (checklist de `BOOTSTRAP_GO.md`) no consta como realizada.
- La detección «no instalado» de WinGet conserva un texto en inglés solo como respaldo; la señal principal es el código de salida.

## Registro de iteración

Añadir una entrada por PR, la más reciente arriba. Sin secretos, rutas personales ni datos de usuarios.

```text
AAAA-MM-DD · rama/PR · hito/tarea
Cambios: …
Pruebas ejecutadas y resultado: … (indicar entorno real o simulado)
No ejecutado / limitaciones: …
Siguiente acción: …
```

- **2026-11-11 · feat/h2-embedded-catalog-generic-adapter · PR #3 · H2.** Catálogo y allowlist incrustados (ADR-0004); adaptador genérico WinGet/Flatpak; instalación visible con timeouts; detección de terminal por SO y `--non-interactive`; `next_step` en el plan; tests de CLI con proveedores simulados. Documentación y hitos reorientados (ADR-0005), `CLAUDE.md` y este fichero. Go test/vet (Linux y Windows) y tests Python en verde. Pruebas reales confirmadas por el usuario; CI en verde en Linux y Windows. Fusionada en main (squash). Siguiente: H3-1.
