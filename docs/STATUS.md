# Estado de Sovereign

Fuente única del estado del proyecto y de la **siguiente acción**. Actualizarla al final de cada iteración (ver «Registro de iteración»). El chat o la memoria de un agente no son el registro del proyecto.

Última actualización: 2026-11-22.

## Hitos

Orden según [ADR-0005](architecture/adr/0005-roadmap-release-first.md). Estados: pendiente · en curso · en revisión · completado · bloqueado.

| Hito | Estado | Evidencia |
| --- | --- | --- |
| H0 — Fundación | Completado | Esquema v1, validador Python y CI Windows/Linux |
| H1 — Planificación | Completado | `plan` determinista y JSON v1; tests de alternativas, dependencias e incompatibilidades |
| H2 — Primera instalación (LocalSend) | Completado | PR #2 y PR #3 fusionadas. CI de PR #3 en verde en ubuntu-latest y windows-latest (Go, catálogo). **El usuario confirma pruebas reales (2026-11-11)**; no se han adjuntado registros |
| H3 — Perfil esencial instalable | Completo en código; aceptación real parcial | H3-1 a H3-4 fusionadas (PR #4–#7). En Windows real solo consta la **detección** de VLC y LibreOffice ya instalados; falta la instalación real del perfil en un equipo limpio (Windows y Linux) |
| H4 — Primera release distribuible | En curso | H4-1 fusionado (PR #8; 2026-11-19). Generación reproducible de binarios/SBOM verificada por CI; sin etiqueta, publicación ni comprobación de attestation reales. H4-2 (bootstrap con release verificada) pendiente |
| H5 — Datos y portabilidad | Pendiente | — |
| H6 — Android guiado | Pendiente | — |
| H7 — Kits offline | Pendiente | — |
| H8 — Conector Multisuite opcional | Pendiente; prerrequisitos externos | Multisuite no expone API ni acceso VPN todavía (su hito P03) |
| H9 — UX y guías | Pendiente | — |

## Qué funciona hoy

- `suitectl doctor | catalog list | profiles list | plan` (solo lectura) y `verify` / `apply --app <id> --yes` para `localsend`, `vlc` y `libreoffice`.
- `status`: estado deseado frente a observado y registrado, con avisos de deriva; solo lectura, texto y JSON.
- `apply --profile essential --yes` instala con un solo consentimiento todo lo que falta del perfil, sigue tras un fallo, bloquea dependientes y resume por app; journal local sin datos personales y bloqueo contra dos `apply` simultáneos (ADR-0007).
- Funciona desde cualquier carpeta: catálogo, perfiles y allowlist van incrustados. `--root <checkout>` lee una copia de desarrollo; la allowlist nunca se lee de disco.
- Instalable en `main`:
  - **LocalSend** — validado en dispositivo real (H2).
  - **VLC** — *simulado; en Windows real validada la detección de una instalación existente (máquina) y la idempotencia; instalación, UAC y cancelación pendientes.* Windows en ámbito máquina (aviso de UAC, solo desde terminal); Linux desde el Flathub comunitario.
  - **LibreOffice** — *simulado; en Windows real validada la detección de una instalación existente y la idempotencia; instalación, UAC, cancelación y VC++ Redistributable pendientes.* Windows en ámbito máquina con la dependencia declarada del Visual C++ Redistributable; Linux desde Flathub (mantenido por desarrolladores de LibreOffice).
  - **CoMaps** — *simulado; pendiente de aceptación real.* Linux desde Flathub (`app.comaps.comaps`, comunitario); Windows con WinGet `CoMaps.CoMaps` (`scope: unspecified`: sin `--scope`, aviso de posible UAC, dependencia VC++ x64).
  - **Kiwix** — *simulado; pendiente de aceptación real.* Windows con Kiwix JS Electron (`Kiwix.KiwixJS.Electron`, ámbito usuario); Linux con Kiwix Desktop de Flathub (`org.kiwix.desktop`, comunitario, riesgo aceptado: sin actualizaciones desde 2024-12 y runtime sin soporte).
  - Android: no soportado aún.
- El plan muestra el ámbito (`scope`) y el origen (`provenance`) de cada paquete verificado, y avisa antes del consentimiento si pedirá administrador o si es un paquete comunitario (ADR-0006).
- Uso: con terminal, el gestor muestra progreso y avisos; sin terminal o con `--non-interactive`, WinGet no pregunta y falla en vez de colgarse.

## Siguiente acción

0. **Fusionar #14 → #15 → #16** tras revisión y hacer la aceptación real del setup completo después (norma flexibilizada el 2026-11-22). Si una app falla en real, revertirla o volver su plataforma a `pending` en la siguiente PR. Validar también, como hipótesis abiertas: que `winget install` sin `--scope` instala CoMaps (UAC o no) y que 3 `winget list` simultáneos no fallan.
1. **Revisar la integración en `main`** de H4-1 (PR #8 ya fusionada): comprobar la CI de `main`, repetibilidad de las compilaciones y salud del catálogo. La publicación de una versión etiquetada todavía no se ha probado.
2. **Publicar la primera release** cuando el mantenedor lo decida: `git tag -a v0.4.0 -m "suitectl 0.4.0" && git push origin v0.4.0` (o `v0.4.0-rc.1` para una pre-release). El workflow firma la procedencia y crea la release. Verificarla con las instrucciones del README («Download a release») y registrar aquí el resultado.
3. **Aceptación real del perfil** en un equipo sin VLC ni LibreOffice, idealmente ya con el binario de la release (sin Git ni Go):

   ```bash
   # Linux
   git clone https://github.com/sempitern0/Sovereign && cd Sovereign
   ./bootstrap/setup.sh                       # instala Go de usuario si falta y ejecuta doctor; no toca el PATH
   GO="$(command -v go || echo "${XDG_DATA_HOME:-$HOME/.local/share}/sovereign/toolchains/go1.27.3/bin/go")"
   GOTOOLCHAIN=local "$GO" build -o suitectl ./cmd/suitectl
   ./suitectl apply --profile essential       # sin --yes: lista lo que falta, no cambia nada (código 3)
   ./suitectl apply --profile essential --yes # instala lo que falte; repetirlo debe dar "already installed"
   ./suitectl status                          # todo "installed"; "Profile complete"
   ```

   ```powershell
   # Windows (PowerShell)
   git clone https://github.com/sempitern0/Sovereign; cd Sovereign
   .\bootstrap\setup.ps1                     # añade Go al PATH de usuario; abrir una terminal NUEVA después
   go build -o suitectl.exe .\cmd\suitectl
   .\suitectl.exe apply --profile essential        # sin --yes: avisos y lista de lo que falta
   .\suitectl.exe apply --profile essential --yes  # Windows mostrará UAC para VLC y LibreOffice
   .\suitectl.exe status                           # desinstalar una app a mano y repetir: debe avisar de la deriva
   ```

   Escenarios: los de [H2_LOCALSEND.md](H2_LOCALSEND.md#escenarios-de-aceptación-en-dispositivo-real) y además:
   - **Windows:** aparece UAC; cancelarlo deja error visible y sin éxito falso; tras aceptar, la app aparece en el menú Inicio y abre archivos; segunda ejecución sin cambios; `--non-interactive` se niega sin llamar a WinGet; con LibreOffice, el plan muestra solo la dependencia de la arquitectura del equipo y el VC++ Redistributable (si falta) se instala en el mismo flujo.
   - **Linux:** el plan avisa de que VLC es comunitario; la primera instalación descarga runtimes grandes (KDE para VLC, freedesktop para LibreOffice); `verify` y segunda ejecución sin cambios.
   - **Perfil:** cancelar el UAC de una app la marca `failed` y el resto continúa; el resumen termina con código 1 y repetir el comando completa lo que faltaba; dos `apply` a la vez → el segundo se niega; el journal (`%LOCALAPPDATA%\Sovereign\state\journal.json` o `~/.local/state/sovereign/journal.json`) refleja cada resultado sin rutas ni nombres de usuario.

   Registrar el resultado aquí. Si algo falla: abrir un PR de corrección o retirar la entrada de `catalog/install-allowlist.yaml` (la app vuelve a quedar no instalable).
4. **H4-2:** el bootstrap descarga la release de la versión fijada, comprueba `SHA256SUMS` y, si `gh` está disponible, la attestation; ya no instala Go para usar Sovereign. Ver [plan §10](../PLAN_TECNICO_SUITE_LIBRE.md#10-hoja-de-ruta-y-entregables) y la fila «Release / bootstrap (H4)» de `AGENTS.md`.

## Limitaciones conocidas

- Aún no se ha publicado ninguna release: la primera sale al crear una etiqueta `v*` (decisión del mantenedor). La PR #8 está fusionada y la CI verificó la compilación reproducible, pero todavía no la firma/publicación desde un tag real. Hasta entonces, bootstrap con Go de usuario o compilar (ya funciona sin red gracias a `vendor/`).
- Los binarios no llevan firma Authenticode; SmartScreen puede avisar en Windows. La autenticidad se verifica con `gh attestation verify`.
- La QA real del bootstrap de Go en equipos limpios (checklist de `BOOTSTRAP_GO.md`) no consta como realizada.
- La detección «no instalado» de WinGet conserva un texto en inglés solo como respaldo; la señal principal es el código de salida.

## Registro de iteración

- **2026-11-22 · feat/kiwix-windows (#16) · Mejoras de la revisión de calidad.** (1) `install.Cache` guarda el estado de WinGet por ID: `setup --yes` ya no repite `winget list` antes de instalar; se sigue verificando después (alineado con `AGENTS.md`). (2) La semántica del ámbito tiene un único dueño en `internal/catalog` (`Scope*`, `ScopeMayElevate`, `ScopeNotice`); instalador, planner, validación y allowlist la consultan. (3) Kiwix Linux pasa a `provenance: community` (no consta verificación de Flathub por Kiwix); el plan muestra el aviso comunitario. (4) Comentarios de ámbito corregidos. Tests: `profilePackages(t)` sustituye la lista fija de tres apps; nuevos tests de caché WinGet (reutilización e intentos fallidos no cacheados). Pruebas (Linux, simulado): gofmt, `go vet` (también `GOOS=windows`), `go test -race ./...`, validador y tests Python en verde.

- **2026-11-22 · feat/kiwix-windows (#16) · Normas flexibles, ámbito `unspecified`, rendimiento y refactor.** Decisiones del mantenedor: PRs por lotes, aceptación real tras fusionar (con reversión), paquetes comunitarios y ámbito máquina/no declarado admitidos con aviso; sigue solo software libre y la decisión explícita para paquetes sin mantenimiento. Cambios: `AGENTS.md`, `CLAUDE.md` y enmienda de ADR-0006; `scope: unspecified` (Go, JSON Schema, validador Python, planner y WinGet sin `--scope`, trata como máquina); CoMaps Windows y Kiwix Linux habilitados; `install.Cache` + inspección paralela (máx. 3); CLI con tabla de comandos y `output.go`. **Medido con proveedor simulado:** `setup --yes` con 5 apps Flatpak que faltan pasa de 25 llamadas (5 install, 15 list, 5 remotes) a 12 (5 install, 6 list, 1 remotes). Pruebas (Linux, simulado): gofmt, `go vet` (también `GOOS=windows`), `go test -race ./...`, validador, tests Python, `sh -n` y release en seco, en verde. **Hipótesis no validadas:** WinGet sin `--scope` con el instalador NSIS de CoMaps; concurrencia de `winget list` en un equipo real; ganancia de tiempo real en Windows (no medida).

- **2026-11-21 · feat/kiwix-windows · Kiwix en el setup único (Windows).** Decisión del mantenedor: Kiwix JS en Windows y Linux pendiente. WinGet `Kiwix.KiwixJS.Electron` 3.9.3-E (manifiesto en microsoft/winget-pkgs; publisher Kiwix; GPL-3.0; NSIS x86 desde github.com/kiwix/kiwix-js-pwa con SHA-256; instalador de ámbito usuario `/CURRENTUSER`; sin `PackageDependencies`) → `scope: user`, `provenance: upstream`. Kiwix Desktop no existe en WinGet. Linux `pending`: Flathub `org.kiwix.desktop` 2.4.1 sin cambios desde 2024-12 y runtime KDE 5.15-23.08 sin soporte. Añadido a `profiles/essential.yaml`. Pruebas (Linux, simulado; compilación de tests con GOOS=windows): gofmt, go vet, go test, validador y tests Python en verde. No ejecutado: WinGet real, CI Windows.
  - Aceptación pendiente (Windows real): `.\sovereign.ps1 setup` lista Kiwix sin aviso de UAC; `setup --yes` lo instala sin pedir administrador y aparece en el menú Inicio; `verify --app kiwix`; segunda ejecución sin cambios; abrir Kiwix JS, descargar un ZIM pequeño y leerlo sin red.

- **2026-11-21 · feat/comaps · CoMaps en el setup único.** Linux verificado: Flathub `app.comaps.comaps` 2026.11.07 (manifiesto en github.com/flathub/app.comaps.comaps, etiqueta oficial `v2026.11.07-16` con commit fijado, última actualización 2026-11-15, runtime KDE 6.11), `scope: user`, `provenance: community` (no se pudo confirmar verificación de Flathub por el proyecto; Codeberg no es accesible desde el entorno del agente). Windows sigue `pending`: WinGet `CoMaps.CoMaps` 2026.06.13 (CoMaps Community, NSIS x64, dependencia VC++ x64) **no declara `Scope`**; no se inventa. Añadido a `profiles/essential.yaml`. `plan`/`setup` muestran las apps no verificadas en la plataforma como «not yet available (skipped)» y no las cuentan como pendientes. Tests nuevos: app pendiente del setup se omite sin tocar el gestor ni dejar el perfil incompleto; resumen del plan. Tests de perfil/estado calculan los paquetes desde la allowlist. Pruebas (Linux, simulado): gofmt, go vet, go test, validador y tests Python en verde. No ejecutado: Flatpak real, CI Windows.
  - Aceptación pendiente (Linux real): `sh sovereign.sh setup` muestra CoMaps con aviso comunitario; `setup --yes` lo instala (descarga runtime KDE 6.11); `verify --app comaps`; segunda ejecución sin cambios; abrir CoMaps y descargar una región sin cuenta.
  - Para habilitar Windows: en un equipo real, `winget show CoMaps.CoMaps` e instalar manualmente observando si aparece UAC y si la ruta es `Program Files` (máquina) o `%LOCALAPPDATA%` (usuario); con ese dato, PR aparte.

- **2026-11-21 · chore/remove-winhance · Retirar WinHance.** Decisión del mantenedor: solo software libre en el catálogo. Eliminado `catalog/apps/winhance.yaml` y el enlace en `privacy audit`; test que impide reintroducirlo; README corregido (seguía citando `profiles/offline.yaml`, eliminado en #13). Pruebas (entorno del agente, Linux, simulado): gofmt, go vet, go test, validador de catálogo y 22 tests Python en verde. No ejecutado: CI Windows (la ejecuta GitHub).

- **2026-11-21 · feat/single-setup-experience · Instalación personal unificada.** Eliminada la selección de perfiles y los YAML vacíos de Offline, Multimedia y Avanzado. Un único `setup` inspecciona y prepara el equipo; `setup --yes` usa la allowlist, consentimiento, journal e idempotencia existentes. `essential` sigue como identificador interno para compatibilidad. Mapas/Wikipedia y privacidad pasan a opciones específicas, sin prometer automatización aún no implementada. Pruebas Windows/Linux y CI pendientes.


- **2026-11-21 · feat/offline-knowledge-windows-privacy · Biblioteca offline y privacidad Windows.** Catálogo opcional inicial para Kiwix, CoMaps y WinHance con licencias y procedencia diferenciadas. Auditoría Windows de solo lectura, sin modificación de telemetría, servicios ni procesos; asistente ampliado. ZIM y mapas todavía requieren descarga independiente y no hay nuevas apps en la allowlist. CI y QA pendientes.


Añadir una entrada por PR, la más reciente arriba. Sin secretos, rutas personales ni datos de usuarios.

```text
AAAA-MM-DD · rama/PR · hito/tarea
Cambios: …
Pruebas ejecutadas y resultado: … (indicar entorno real o simulado)
No ejecutado / limitaciones: …
Siguiente acción: …
```

- **2026-11-19 · feat/interactive-wizard · Asistente de terminal.** Menú interactivo con perfiles, previsualización, instalación confirmada escribiendo `SI`, estado, catálogo, diagnóstico y aplicaciones individuales revisadas. Reutiliza la allowlist; no requiere librerías TUI externas, mantiene la CLI para automatización y no instala Git. Go solo se prepara automáticamente al ejecutar desde fuente. Validación CI y QA en terminales reales pendientes de aceptación.
- **2026-11-19 · feat/terminal-ux-one-command · UX terminal y arranque (PR #10).** `plan`, `status` y `apply --profile` muestran por defecto resúmenes compactos y solo avisos accionables; `--verbose` ofrece detalle y `--json` mantiene su contrato. Nuevos `sovereign.ps1`/`sovereign.sh` ejecutan cualquier comando de la CLI desde un clon y utilizan el bootstrap Go existente sin instalación global. CI: tests/compilación Windows y Linux, contratos y doble build reproducible superados (runs 44579125378, 44579125398); smoke tests con Go preinstalado. Pendiente: QA de descarga de Go desde cero y aceptación de primeras instalaciones en Windows/Linux. Siguiente acción: integrar PR y validar en terminales reales.
- **2026-11-19 · main · H4-1 fusionada (PR #8).** Binarios reproducibles y SBOM para Windows/Linux x86-64/ARM64 integrados. CI de la rama de PR verde; publicación y attestation reales no comprobadas. Limpieza posterior: diagnóstico limitado a gestores soportados por plataforma, retirada de función muerta `IsOffline`, simplificación de `.gitattributes` y acciones de CI fijadas por SHA. Se conserva documentación de arquitectura, licencias y fuentes vendorizadas. Próximo paso: QA desde máquinas limpias y primera release cuando proceda.
- **2026-11-13 · feat/h4-1-release · H4-1.** `vendor/` con `gopkg.in/yaml.v3` v3.0.1 (hash idéntico al de `go.sum`; obtenido del repositorio oficial vía proxy local porque el proxy de Go no es accesible desde el entorno del agente; la CI lo regenera desde el proxy oficial y exige cero diferencias). Compilación y tests sin red con caché vacía. `suitectl version [--json]` con versión y commit inyectados. `tools/release.py`: 4 objetivos, archivos deterministas, SBOM SPDX 2.3 y SHA256SUMS; dos compilaciones locales independientes dieron artefactos idénticos; binario extraído ejecutado fuera del repositorio. Workflow `release.yml`: acciones fijadas por SHA (entradas comprobadas en cada `action.yml`), jobs con mínimo privilegio, publicación solo con etiqueta `v*` tras firmar la procedencia (ADR-0008). Tests: 8 de Python para la herramienta de release y 1 de Go para `version`. No ejecutado: el workflow en GitHub con etiqueta (no se ha publicado ninguna release) ni la verificación con `gh attestation verify`.
- **2026-11-13 · main · H3-4 fusionada (PR #7).**
- **2026-11-11 · feat/h3-4-status · H3-4.** `status [--profile] [--json]` de solo lectura: por app, estado observado, último resultado registrado y nota de deriva (registrado como instalado pero ya no presente, último intento fallido, instalado fuera de Sovereign) con siguiente paso; `complete` en JSON (`schema_version` 1); entradas del journal fuera del perfil en `other_recorded`. `state.Store.Peek` lee sin tocar el disco; `Load` se apoya en él. Tests con proveedores simulados, incluido que `status` no modifica el directorio de estado ni ejecuta comandos de instalación (mutación `Peek`→`Load` detectada). Go test/vet en verde. No ejecutado: proveedores reales.
- **2026-11-11 · main · H3-3 fusionada (PR #6).**
- **2026-11-11 · feat/h3-3-apply-profile · H3-3.** `apply --profile` con un consentimiento, orden topológico, continuación tras fallos, `blocked` para dependientes y resumen con código 1 si el perfil queda incompleto. Nuevo `internal/state`: journal atómico (0600, sin salida de proveedores: solo clases de error), journal corrupto apartado y bloqueo con detección de proceso muerto (ADR-0007). `apply --app` también usa bloqueo y journal. Tests con proveedores simulados, incluido que el journal no filtra rutas; mutaciones comprobadas (sin bloqueo por dependencia o sin lock, los tests fallan). Go test/vet (Linux; vet Windows) en verde. La primera CI falló porque la regla `state/` de `.gitignore` ocultaba `internal/state/`; corregida (`/state/`) y verificado un checkout limpio. No ejecutado: proveedores reales.
- **2026-11-11 · prueba real del mantenedor · Windows amd64.** `bootstrap\setup.ps1` correcto (doctor: winget disponible). `go build` correcto. `apply --app vlc --yes` y `apply --app libreoffice --yes` → `installed (Already installed; no changes)`: **valida en real la detección de paquetes de ámbito máquina y la idempotencia**. No valida instalación, UAC, cancelación ni la dependencia VC++ (las apps ya estaban instaladas). Linux sin probar.
- **2026-11-11 · main · H3-1 + H3-2 fusionadas.** PR #4 y PR #5 fusionadas (squash) a petición del mantenedor, con CI en verde en Linux y Windows y **sin aceptación real todavía** (excepción explícita al flujo de `AGENTS.md`). Siguiente: primeras instalaciones reales de VLC y LibreOffice, después H3-3.
- **2026-11-11 · feat/h3-2-libreoffice · H3-2.** LibreOffice verificado: WinGet `TheDocumentFoundation.LibreOffice` (manifiesto 26.8.2.2, publisher The Document Foundation, MSI de download.documentfoundation.org con SHA-256, solo ámbito máquina, dependencia `Microsoft.VCRedist.2015+` por arquitectura) y Flathub `org.libreoffice.LibreOffice` (manifiesto mantenido por desarrolladores de LibreOffice, compila la etiqueta oficial `libreoffice-26.8.2.2`; sin insignia de verificado ni aviso comunitario → `upstream`). Decisión del mantenedor: ámbito máquina y dependencia declarada (ADR-0006, punto 5). Nuevo campo opcional `provider_dependencies`, comparado en CI, mostrado antes del consentimiento y filtrado por arquitectura del equipo. Los tests que necesitaban una app no aprobada usan ahora una ficticia pendiente. Go test/vet (Linux; vet Windows) y 14 tests Python en verde. No ejecutado: WinGet/Flatpak reales. Siguiente: aceptación real de VLC y LibreOffice.
- **2026-11-11 · feat/h3-1-vlc · H3-1.** VLC verificado: WinGet `VideoLAN.VLC` (manifiesto 3.0.24 de `microsoft/winget-pkgs`, publisher VideoLAN, instaladores de download.videolan.org con SHA-256; solo ámbito máquina) y Flathub `org.videolan.VLC` (paquete comunitario no verificado por VideoLAN; compila desde el tarball oficial 3.0.24 con SHA-256). Decisión del mantenedor: Windows en ámbito máquina con UAC y Flathub comunitario aceptado (ADR-0006). Campos `scope`/`provenance` obligatorios y comparados en CI; ámbito máquina solo con terminal; avisos en plan y consentimiento. LocalSend registrado como `user`/`upstream` según sus manifiestos. Go test/vet (Linux; vet Windows) y 13 tests Python en verde. Corregido un test que usaba VLC con el runner real. No ejecutado: WinGet/Flatpak reales. Siguiente: aceptación real de VLC.
- **2026-11-11 · feat/h2-embedded-catalog-generic-adapter · PR #3 · H2.** Catálogo y allowlist incrustados (ADR-0004); adaptador genérico WinGet/Flatpak; instalación visible con timeouts; detección de terminal por SO y `--non-interactive`; `next_step` en el plan; tests de CLI con proveedores simulados. Documentación y hitos reorientados (ADR-0005), `CLAUDE.md` y este fichero. Go test/vet (Linux y Windows) y tests Python en verde. Pruebas reales confirmadas por el usuario; CI en verde en Linux y Windows. Fusionada en main (squash). Siguiente: H3-1.
