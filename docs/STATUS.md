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
| H3 — Perfil esencial instalable | En curso | H3-1 (VLC) y H3-2 (LibreOffice) fusionadas; en Windows real solo se ha validado la **detección** (ya estaban instaladas), no la instalación. H3-3 (`apply --profile`, journal, bloqueo) en revisión. H3-4 pendiente |
| H4 — Primera release distribuible | Pendiente | — |
| H5 — Datos y portabilidad | Pendiente | — |
| H6 — Android guiado | Pendiente | — |
| H7 — Kits offline | Pendiente | — |
| H8 — Conector Multisuite opcional | Pendiente; prerrequisitos externos | Multisuite no expone API ni acceso VPN todavía (su hito P03) |
| H9 — UX y guías | Pendiente | — |

## Qué funciona hoy

- `suitectl doctor | catalog list | profiles list | plan` (solo lectura) y `verify` / `apply --app <id> --yes` para `localsend`, `vlc` y `libreoffice`.
- Rama H3-3: `apply --profile essential --yes` instala con un solo consentimiento todo lo que falta del perfil, sigue tras un fallo, bloquea dependientes y resume por app; journal local sin datos personales y bloqueo contra dos `apply` simultáneos (ADR-0007).
- Funciona desde cualquier carpeta: catálogo, perfiles y allowlist van incrustados. `--root <checkout>` lee una copia de desarrollo; la allowlist nunca se lee de disco.
- Instalable en `main`:
  - **LocalSend** — validado en dispositivo real (H2).
  - **VLC** — *simulado; en Windows real validada la detección de una instalación existente (máquina) y la idempotencia; instalación, UAC y cancelación pendientes.* Windows en ámbito máquina (aviso de UAC, solo desde terminal); Linux desde el Flathub comunitario.
  - **LibreOffice** — *simulado; en Windows real validada la detección de una instalación existente y la idempotencia; instalación, UAC, cancelación y VC++ Redistributable pendientes.* Windows en ámbito máquina con la dependencia declarada del Visual C++ Redistributable; Linux desde Flathub (mantenido por desarrolladores de LibreOffice).
  - Android: no soportado aún.
- El plan muestra el ámbito (`scope`) y el origen (`provenance`) de cada paquete verificado, y avisa antes del consentimiento si pedirá administrador o si es un paquete comunitario (ADR-0006).
- Uso: con terminal, el gestor muestra progreso y avisos; sin terminal o con `--non-interactive`, WinGet no pregunta y falla en vez de colgarse.

## Siguiente acción

1. **Revisar y fusionar H3-3** (rama `feat/h3-3-apply-profile`).
2. **Completar las pruebas reales** en un equipo donde VLC y LibreOffice **no** estén instalados (por ejemplo, una VM de Windows recién creada y una distribución Linux con Flatpak), usando ya el perfil: `suitectl apply --profile essential` (sin `--yes`, debe listar lo que falta y no cambiar nada) y después con `--yes`. Preparación del equipo:

   ```bash
   # Linux
   git clone https://github.com/sempitern0/Sovereign && cd Sovereign
   ./bootstrap/setup.sh                       # instala Go de usuario si falta y ejecuta doctor; no toca el PATH
   GO="$(command -v go || echo "${XDG_DATA_HOME:-$HOME/.local/share}/sovereign/toolchains/go1.27.3/bin/go")"
   GOTOOLCHAIN=local "$GO" build -o suitectl ./cmd/suitectl
   ./suitectl apply --profile essential       # sin --yes: lista lo que falta, no cambia nada (código 3)
   ./suitectl apply --profile essential --yes # instala lo que falte; repetirlo debe dar "already installed"
   ./suitectl verify --app vlc
   ```

   ```powershell
   # Windows (PowerShell)
   git clone https://github.com/sempitern0/Sovereign; cd Sovereign
   .\bootstrap\setup.ps1                     # añade Go al PATH de usuario; abrir una terminal NUEVA después
   go build -o suitectl.exe .\cmd\suitectl
   .\suitectl.exe apply --profile essential        # sin --yes: avisos y lista de lo que falta
   .\suitectl.exe apply --profile essential --yes  # Windows mostrará UAC para VLC y LibreOffice
   ```

   Escenarios: los de [H2_LOCALSEND.md](H2_LOCALSEND.md#escenarios-de-aceptación-en-dispositivo-real) y además:
   - **Windows:** aparece UAC; cancelarlo deja error visible y sin éxito falso; tras aceptar, la app aparece en el menú Inicio y abre archivos; segunda ejecución sin cambios; `--non-interactive` se niega sin llamar a WinGet; con LibreOffice, el plan muestra solo la dependencia de la arquitectura del equipo y el VC++ Redistributable (si falta) se instala en el mismo flujo.
   - **Linux:** el plan avisa de que VLC es comunitario; la primera instalación descarga runtimes grandes (KDE para VLC, freedesktop para LibreOffice); `verify` y segunda ejecución sin cambios.
   - **Perfil:** cancelar el UAC de una app la marca `failed` y el resto continúa; el resumen termina con código 1 y repetir el comando completa lo que faltaba; dos `apply` a la vez → el segundo se niega; el journal (`%LOCALAPPDATA%\Sovereign\state\journal.json` o `~/.local/state/sovereign/journal.json`) refleja cada resultado sin rutas ni nombres de usuario.

   Registrar el resultado aquí. Si algo falla: abrir un PR de corrección o retirar la entrada de `catalog/install-allowlist.yaml` (la app vuelve a quedar no instalable).
3. **H3-4:** `status`: estado observado frente al deseado a partir del journal y de los gestores, sin efectos (texto y JSON). Con H3-4 y las pruebas reales, H3 queda completo y se pasa a H4 (release distribuible). Ver [plan §10](../PLAN_TECNICO_SUITE_LIBRE.md#10-hoja-de-ruta-y-entregables).

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

- **2026-11-11 · feat/h3-3-apply-profile · H3-3.** `apply --profile` con un consentimiento, orden topológico, continuación tras fallos, `blocked` para dependientes y resumen con código 1 si el perfil queda incompleto. Nuevo `internal/state`: journal atómico (0600, sin salida de proveedores: solo clases de error), journal corrupto apartado y bloqueo con detección de proceso muerto (ADR-0007). `apply --app` también usa bloqueo y journal. Tests con proveedores simulados, incluido que el journal no filtra rutas; mutaciones comprobadas (sin bloqueo por dependencia o sin lock, los tests fallan). Go test/vet (Linux; vet Windows) en verde. No ejecutado: proveedores reales.
- **2026-11-11 · prueba real del mantenedor · Windows amd64.** `bootstrap\setup.ps1` correcto (doctor: winget disponible). `go build` correcto. `apply --app vlc --yes` y `apply --app libreoffice --yes` → `installed (Already installed; no changes)`: **valida en real la detección de paquetes de ámbito máquina y la idempotencia**. No valida instalación, UAC, cancelación ni la dependencia VC++ (las apps ya estaban instaladas). Linux sin probar.
- **2026-11-11 · main · H3-1 + H3-2 fusionadas.** PR #4 y PR #5 fusionadas (squash) a petición del mantenedor, con CI en verde en Linux y Windows y **sin aceptación real todavía** (excepción explícita al flujo de `AGENTS.md`). Siguiente: primeras instalaciones reales de VLC y LibreOffice, después H3-3.
- **2026-11-11 · feat/h3-2-libreoffice · H3-2.** LibreOffice verificado: WinGet `TheDocumentFoundation.LibreOffice` (manifiesto 26.8.2.2, publisher The Document Foundation, MSI de download.documentfoundation.org con SHA-256, solo ámbito máquina, dependencia `Microsoft.VCRedist.2015+` por arquitectura) y Flathub `org.libreoffice.LibreOffice` (manifiesto mantenido por desarrolladores de LibreOffice, compila la etiqueta oficial `libreoffice-26.8.2.2`; sin insignia de verificado ni aviso comunitario → `upstream`). Decisión del mantenedor: ámbito máquina y dependencia declarada (ADR-0006, punto 5). Nuevo campo opcional `provider_dependencies`, comparado en CI, mostrado antes del consentimiento y filtrado por arquitectura del equipo. Los tests que necesitaban una app no aprobada usan ahora una ficticia pendiente. Go test/vet (Linux; vet Windows) y 14 tests Python en verde. No ejecutado: WinGet/Flatpak reales. Siguiente: aceptación real de VLC y LibreOffice.
- **2026-11-11 · feat/h3-1-vlc · H3-1.** VLC verificado: WinGet `VideoLAN.VLC` (manifiesto 3.0.24 de `microsoft/winget-pkgs`, publisher VideoLAN, instaladores de download.videolan.org con SHA-256; solo ámbito máquina) y Flathub `org.videolan.VLC` (paquete comunitario no verificado por VideoLAN; compila desde el tarball oficial 3.0.24 con SHA-256). Decisión del mantenedor: Windows en ámbito máquina con UAC y Flathub comunitario aceptado (ADR-0006). Campos `scope`/`provenance` obligatorios y comparados en CI; ámbito máquina solo con terminal; avisos en plan y consentimiento. LocalSend registrado como `user`/`upstream` según sus manifiestos. Go test/vet (Linux; vet Windows) y 13 tests Python en verde. Corregido un test que usaba VLC con el runner real. No ejecutado: WinGet/Flatpak reales. Siguiente: aceptación real de VLC.
- **2026-11-11 · feat/h2-embedded-catalog-generic-adapter · PR #3 · H2.** Catálogo y allowlist incrustados (ADR-0004); adaptador genérico WinGet/Flatpak; instalación visible con timeouts; detección de terminal por SO y `--non-interactive`; `next_step` en el plan; tests de CLI con proveedores simulados. Documentación y hitos reorientados (ADR-0005), `CLAUDE.md` y este fichero. Go test/vet (Linux y Windows) y tests Python en verde. Pruebas reales confirmadas por el usuario; CI en verde en Linux y Windows. Fusionada en main (squash). Siguiente: H3-1.
