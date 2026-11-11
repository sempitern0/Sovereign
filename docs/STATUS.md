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
| H3 — Perfil esencial instalable | En curso | H3-1 (VLC, PR #4) y H3-2 (LibreOffice, PR #5) **fusionadas por decisión del mantenedor antes de la aceptación real**, que queda pendiente y es la siguiente acción. CI en verde en Linux y Windows. H3-3 y H3-4 pendientes |
| H4 — Primera release distribuible | Pendiente | — |
| H5 — Datos y portabilidad | Pendiente | — |
| H6 — Android guiado | Pendiente | — |
| H7 — Kits offline | Pendiente | — |
| H8 — Conector Multisuite opcional | Pendiente; prerrequisitos externos | Multisuite no expone API ni acceso VPN todavía (su hito P03) |
| H9 — UX y guías | Pendiente | — |

## Qué funciona hoy

- `suitectl doctor | catalog list | profiles list | plan` (solo lectura) y `verify` / `apply --app <id> --yes` para `localsend`, `vlc` y `libreoffice`.
- Funciona desde cualquier carpeta: catálogo, perfiles y allowlist van incrustados. `--root <checkout>` lee una copia de desarrollo; la allowlist nunca se lee de disco.
- Instalable en `main`:
  - **LocalSend** — validado en dispositivo real (H2).
  - **VLC** — *validado con proveedores simulados; aceptación real pendiente.* Windows en ámbito máquina (aviso de UAC, solo desde terminal); Linux desde el Flathub comunitario.
  - **LibreOffice** — *validado con proveedores simulados; aceptación real pendiente.* Windows en ámbito máquina con la dependencia declarada del Visual C++ Redistributable; Linux desde Flathub (mantenido por desarrolladores de LibreOffice).
  - Android: no soportado aún.
- El plan muestra el ámbito (`scope`) y el origen (`provenance`) de cada paquete verificado, y avisa antes del consentimiento si pedirá administrador o si es un paquete comunitario (ADR-0006).
- Uso: con terminal, el gestor muestra progreso y avisos; sin terminal o con `--non-interactive`, WinGet no pregunta y falla en vez de colgarse.

## Siguiente acción

1. **Primeras instalaciones reales de VLC y LibreOffice** en equipos de prueba (Windows y Linux), desde `main`. Preparación:

   ```bash
   # Linux
   git clone https://github.com/sempitern0/Sovereign && cd Sovereign
   ./bootstrap/setup.sh                       # instala Go de usuario si falta y ejecuta doctor; no toca el PATH
   GO="$(command -v go || echo "${XDG_DATA_HOME:-$HOME/.local/share}/sovereign/toolchains/go1.27.3/bin/go")"
   GOTOOLCHAIN=local "$GO" build -o suitectl ./cmd/suitectl
   ./suitectl plan --profile essential        # avisos de paquete comunitario y siguientes pasos
   ./suitectl apply --app vlc --yes           # después: ./suitectl verify --app vlc y repetir apply
   ./suitectl apply --app libreoffice --yes
   ```

   ```powershell
   # Windows (PowerShell)
   git clone https://github.com/sempitern0/Sovereign; cd Sovereign
   .\bootstrap\setup.ps1                     # añade Go al PATH de usuario; abrir una terminal NUEVA después
   go build -o suitectl.exe .\cmd\suitectl
   .\suitectl.exe plan --profile essential   # avisos de administrador y dependencias
   .\suitectl.exe apply --app vlc --yes      # Windows mostrará UAC
   .\suitectl.exe apply --app libreoffice --yes
   ```

   Escenarios: los de [H2_LOCALSEND.md](H2_LOCALSEND.md#escenarios-de-aceptación-en-dispositivo-real) y además:
   - **Windows:** aparece UAC; cancelarlo deja error visible y sin éxito falso; tras aceptar, la app aparece en el menú Inicio y abre archivos; segunda ejecución sin cambios; `--non-interactive` se niega sin llamar a WinGet; con LibreOffice, el plan muestra solo la dependencia de la arquitectura del equipo y el VC++ Redistributable (si falta) se instala en el mismo flujo.
   - **Linux:** el plan avisa de que VLC es comunitario; la primera instalación descarga runtimes grandes (KDE para VLC, freedesktop para LibreOffice); `verify` y segunda ejecución sin cambios.

   Registrar el resultado aquí. Si algo falla: abrir un PR de corrección o retirar la entrada de `catalog/install-allowlist.yaml` (la app vuelve a quedar no instalable).
2. **H3-3:** `apply --profile` con un único consentimiento sobre el plan mostrado, resumen por app y fallos parciales visibles, más el journal local (escritura atómica y bloqueo). Ver backlog H3 en el [plan §10](../PLAN_TECNICO_SUITE_LIBRE.md#10-hoja-de-ruta-y-entregables).

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

- **2026-11-11 · main · H3-1 + H3-2 fusionadas.** PR #4 y PR #5 fusionadas (squash) a petición del mantenedor, con CI en verde en Linux y Windows y **sin aceptación real todavía** (excepción explícita al flujo de `AGENTS.md`). Siguiente: primeras instalaciones reales de VLC y LibreOffice, después H3-3.
- **2026-11-11 · feat/h3-2-libreoffice · H3-2.** LibreOffice verificado: WinGet `TheDocumentFoundation.LibreOffice` (manifiesto 26.8.2.2, publisher The Document Foundation, MSI de download.documentfoundation.org con SHA-256, solo ámbito máquina, dependencia `Microsoft.VCRedist.2015+` por arquitectura) y Flathub `org.libreoffice.LibreOffice` (manifiesto mantenido por desarrolladores de LibreOffice, compila la etiqueta oficial `libreoffice-26.8.2.2`; sin insignia de verificado ni aviso comunitario → `upstream`). Decisión del mantenedor: ámbito máquina y dependencia declarada (ADR-0006, punto 5). Nuevo campo opcional `provider_dependencies`, comparado en CI, mostrado antes del consentimiento y filtrado por arquitectura del equipo. Los tests que necesitaban una app no aprobada usan ahora una ficticia pendiente. Go test/vet (Linux; vet Windows) y 14 tests Python en verde. No ejecutado: WinGet/Flatpak reales. Siguiente: aceptación real de VLC y LibreOffice.
- **2026-11-11 · feat/h3-1-vlc · H3-1.** VLC verificado: WinGet `VideoLAN.VLC` (manifiesto 3.0.24 de `microsoft/winget-pkgs`, publisher VideoLAN, instaladores de download.videolan.org con SHA-256; solo ámbito máquina) y Flathub `org.videolan.VLC` (paquete comunitario no verificado por VideoLAN; compila desde el tarball oficial 3.0.24 con SHA-256). Decisión del mantenedor: Windows en ámbito máquina con UAC y Flathub comunitario aceptado (ADR-0006). Campos `scope`/`provenance` obligatorios y comparados en CI; ámbito máquina solo con terminal; avisos en plan y consentimiento. LocalSend registrado como `user`/`upstream` según sus manifiestos. Go test/vet (Linux; vet Windows) y 13 tests Python en verde. Corregido un test que usaba VLC con el runner real. No ejecutado: WinGet/Flatpak reales. Siguiente: aceptación real de VLC.
- **2026-11-11 · feat/h2-embedded-catalog-generic-adapter · PR #3 · H2.** Catálogo y allowlist incrustados (ADR-0004); adaptador genérico WinGet/Flatpak; instalación visible con timeouts; detección de terminal por SO y `--non-interactive`; `next_step` en el plan; tests de CLI con proveedores simulados. Documentación y hitos reorientados (ADR-0005), `CLAUDE.md` y este fichero. Go test/vet (Linux y Windows) y tests Python en verde. Pruebas reales confirmadas por el usuario; CI en verde en Linux y Windows. Fusionada en main (squash). Siguiente: H3-1.
