# Suite Libre — Plan técnico de alto nivel

> **Estado:** plan vigente; H0–H2 completados. Estado operativo y siguiente acción: [`docs/STATUS.md`](docs/STATUS.md).  
> **Versión del plan:** 1.1 · 2026-11-11 (orden de hitos revisado en [ADR-0005](docs/architecture/adr/0005-roadmap-release-first.md); v1.0 · 2026-11-09).  
> **Plataformas objetivo:** Windows, Linux y Android.  
> **Servidor opcional:** Multisuite (interoperabilidad por capacidades; nunca requisito de funcionamiento).  
> **Documento de producto de origen:** `PLAN_SUITE_LIBRE_OFFLINE.md` (externo; no está versionado en este repositorio). Este plan recoge lo necesario de él.

**Cómo leer este documento.** Es el *qué* y el *porqué* del producto. No hace falta leerlo entero en cada iteración: [`AGENTS.md`](AGENTS.md) indica qué secciones leer según la tarea, y `docs/STATUS.md` dice qué está hecho y qué toca ahora. Si este plan contradice una ADR aceptada o el código, prevalecen la ADR y el código; registra la discrepancia en STATUS.

## 0. Instrucciones de ejecución para cualquier agente

Este documento es la fuente de verdad inicial para iniciar el desarrollo. Antes de codificar:

1. Inspeccionar el repositorio existente y **reutilizar** sus convenciones, herramientas y CI. Si está vacío, aplicar el diseño propuesto aquí. No asumir que Multisuite está instalado ni accesible.
2. Leer `docs/STATUS.md`, `AGENTS.md` y las secciones de este plan y ADR que indique AGENTS.md para la tarea. Identificar decisiones ya implementadas antes de cambiarlas.
3. Proponer una implementación incremental **por pull request**, con alcance acotado, pruebas, documentación y criterio de aceptación. Evitar refactorizaciones ajenas al hito.
4. Priorizar un **camino completo y verificable** (manifest → plan → instalación → verificación) frente a implementar muchas aplicaciones parcialmente.
5. No inventar identificadores de paquetes, APIs, URLs de publicación ni permisos: contrastarlos con fuentes oficiales, anotando fuente y fecha de verificación.
6. Mantener la seguridad y la reversibilidad. Un agente no debe instalar paquetes, borrar datos, alterar credenciales ni cambiar la configuración del sistema sin acción o autorización explícita del usuario.
7. Si falta una decisión no bloqueante, adoptar el valor por defecto documentado y registrar un ADR. Si compromete seguridad o datos, marcar la operación como no soportada y continuar con lo seguro.
8. Al terminar cada PR: explicar cambios, comandos de prueba, resultados reales, limitaciones y siguiente paso; no declarar superadas pruebas que no se ejecutaron.

**Tarea actual:** la indicada en `docs/STATUS.md` (H3 al cerrar H2). No empezar por la UI gráfica ni por conectores NAS.

---

## 1. Objetivo y resultado esperado

Construir un **orquestador local-first de software libre** que permita preparar, mantener, migrar y recuperar entornos personales de uso diario mediante un catálogo declarativo, perfiles y adaptadores por plataforma. Debe poder obtenerse mediante clonación Git o descarga HTTPS de una release y **seguir siendo útil sin conexión, cuenta de terceros ni servidor propio**.

### Capacidades principales

- Detectar Windows/Linux/Android, arquitectura, dependencias, permisos, recursos y conectividad.
- Resolver perfiles de necesidades en aplicaciones concretas compatibles con cada dispositivo.
- Ofrecer un plan previo (*dry-run*) con cambios, fuentes, privilegios, tamaño estimado cuando sea comprobable y riesgos.
- Instalar y comprobar aplicaciones mediante gestores de paquetes existentes, sin reinventarlos.
- Preparar estructura local de datos, exportaciones y contenido offline seleccionado.
- Registrar estado, errores y versiones de manera legible, sin secretos.
- Integrarse voluntariamente con Multisuite y otros proveedores a través de contratos de interoperabilidad.
- Diagnosticar y probar escenarios sin Internet y restauración de copias.

### No objetivos de la primera versión

- Crear una tienda de apps, un NAS, una nube de sincronización propia o un gestor de contraseñas.
- Reimplementar Syncthing, LocalSend, calibre, Navidrome, Joplin o Paperless-ngx.
- Instalar APK silenciosamente en Android convencional, saltar permisos o modificar el sistema con técnicas privilegiadas.
- Garantizar sincronización bidireccional de todas las apps o una copia offline completa de todas las bibliotecas.
- Distribuir contenido de terceros sin derechos o redistribuir instaladores sin verificar licencias.
- Hacer de Multisuite, GitHub o cualquier otra plataforma un requisito en tiempo de ejecución.

### Principios e invariantes

1. **Independencia:** la instalación, configuración básica, datos y funciones esenciales no dependen de Multisuite.
2. **Local-first:** lo que está realmente descargado y validado puede utilizarse sin red.
3. **Portabilidad:** los documentos del usuario son archivos/formatos exportables; no se bloquean en un catálogo propietario.
4. **Idempotencia:** repetir `apply` sobre el mismo estado no produce cambios innecesarios ni pérdida de datos.
5. **Mínimos privilegios y consentimiento:** privilegios solo por operación; nunca cambios destructivos por defecto.
6. **Declarativo y trazable:** el catálogo define el estado deseado; el registro local refleja el observado.
7. **Interoperabilidad por capacidad:** servicios y clientes pueden sustituirse mediante adaptadores.
8. **Mantenibilidad:** núcleo pequeño, pruebas automatizadas, contratos estables y documentación ejecutable.
9. **Seguridad en la cadena de suministro:** orígenes confiables, verificación de artefactos y mínimo código remoto ejecutado.
10. **Sin sorpresas:** no transmitir inventarios, rutas, secretos ni telemetría por defecto.

## 2. Arquitectura de referencia

```text
               Git clone / Release ZIP por HTTPS / USB verificado
                                    |
                             Bootstrap local
                        (PowerShell / shell mínimo)
                                    |
                            suite CLI (suitectl)
             +----------------------+---------------------+
             |                      |                     |
      Catálogo y perfiles     Motor de operaciones    Diagnósticos
      YAML + JSON Schema    detect → resolve → plan   doctor/verify
                            → apply → reconcile
             |                      |
             +----------+-----------+
                        |
           Adaptadores por plataforma (interfaces)
             |                 |                 |
        Windows/WinGet    Linux/Flatpak,      Android/F-Droid,
                          gestor nativo       Obtainium, guiado
             |                 |                 |
             +-----------------+-----------------+
                               |
                     Datos y apps locales
              BibliotecaPersonal / perfiles / backups
                               |
                  Conectores opcionales (opt-in)
                               |
            Multisuite / servicios de terceros compatibles
        Navidrome · Paperless-ngx · HomeBox · Joplin/WebDAV

       Si Multisuite está apagado: el motor y los datos
                   previamente preparados funcionan.
```

### Componentes

| Componente | Responsabilidad | Restricción |
| --- | --- | --- |
| `bootstrap` | Descargar/ejecutar una release verificada o invocar binario local; nunca contener lógica de negocio | No `curl \| sh` o equivalente como flujo recomendado |
| `catalog` | Apps, perfiles, requisitos, orígenes, capacidades, licencias y pruebas | Datos versionados y validados antes de publicar |
| `resolver` | Resolver perfil + OS + arquitectura + alternativas + dependencias | Determinista, sin efectos secundarios |
| `planner` | Comparar estado deseado y real; informar acciones | Dry-run obligatorio y reproducible |
| `executor` | Aplicar operaciones mediante proveedores y verificar resultados | Idempotente; no borrar datos ni desinstalar por defecto |
| `providers` | WinGet, Flatpak, APT/DNF cuando proceda, Android guiado | Adaptadores desacoplados del núcleo |
| `data` | Crear carpetas, exportaciones, recursos offline y metadatos locales | Nunca almacenar secretos en manifiestos Git |
| `doctor` | Comprobar compatibilidad, estado, operaciones offline y fallos | Resultados legibles + salida estructurada |
| `connectors` | Integraciones voluntarias con servicios autohospedados | Deben degradar de forma segura si no están disponibles |

### Decisión de stack inicial (ADR-001 propuesto)

- **Núcleo de escritorio: Go**, distribuido como un binario por OS/arquitectura, para no exigir runtimes previos. CLI primero; sin daemon ni API pública en el MVP.
- **Bootstrap:** scripts pequeños de PowerShell y POSIX shell, que detectan plataforma, validan el artefacto y ejecutan el binario. El ZIP de release incluye el binario y no requiere Git.
- **Contratos y configuración:** YAML para catálogo/perfiles; JSON Schema o validación semántica equivalente; JSON para salida máquina y eventos. El esquema debe versionarse explícitamente.
- **Android:** primer hito móvil mediante catálogo exportable y guía interactiva para F-Droid/Obtainium y permisos de usuario. No se asume ejecución nativa de `suitectl` ni instalación desatendida. Una app acompañante se evalúa después.
- **Persistencia local:** estado estructurado en el directorio de datos de la aplicación, separado de los documentos personales; escrituras atómicas, migración de esquema y bloqueo para evitar dos `apply` simultáneos.
- **UI:** CLI accesible y mensajes en lenguaje claro primero; TUI/GUI solo tras estabilizar APIs y contratos.

Estas decisiones son un punto de partida para acelerar el desarrollo. Cualquier sustitución debe registrar una ADR con impactos en portabilidad, empaquetado, seguridad y testabilidad.

## 3. Modelo declarativo y perfiles

### Entidades mínimas

- `Application`: ID interno estable, nombre, licencia, propósito, fuente oficial, soportes por plataforma, paquetes, dependencias, permisos, capacidades, verificación, política offline y fecha de validación.
- `Profile`: ID, descripción, aplicaciones requeridas/opcionales, alternativas mutuamente excluyentes y políticas de datos.
- `Device`: plataforma, arquitectura, disponibilidad de gestores, recursos y capacidades; sin identificadores personales enviados a servidores.
- `Operation`: ID determinista, tipo, precondiciones, cambios previstos, requisito de privilegios, reversibilidad y verificación.
- `Provider`: detección, consulta, instalación/actualización y comprobación; operaciones no soportadas deben ser explícitas.
- `Connector`: capacidades y estado, autenticación autorizada, importación/exportación/sincronización si hay soporte real.

### Perfilado inicial

| Perfil | Apps/funciones representativas | Política |
| --- | --- | --- |
| `essential` | VLC, LibreOffice, LocalSend, gestor KDBX de escritorio, Ente Auth, Joplin | Base pequeña, alternativas por OS; **no** instalar apps incompatibles |
| `offline` | Kiwix, CoMaps en Android, calibre/lector, documentación y recursos esenciales | Descargar explícitamente datos elegidos, no colecciones gigantes |
| `multimedia` | Picard, FreeTube, NewPipe Android, yt-dlp/FFmpeg, Audacious | Separar reproducción, organización y descargas |
| `advanced` | Krokiet, Buzz, Upscayl, restic, Syncthing donde esté validado | Exigir requisitos y consentimiento, p. ej. GPU o almacenamiento |

**Política de alternativas:** elegir una app principal por función y dispositivo; no instalar Joplin y QOwnNotes, ni varios lectores, automáticamente sin preferencia explícita. Apps y disponibilidad deben validarse por plataforma y versión real antes de incluirse en una release.

### Ejemplo esquemático (no utilizar sin completar/verificar IDs)

```yaml
schema_version: 1
id: localsend
name: LocalSend
category: transfer
profiles: [essential]
capabilities: [lan_transfer]
platforms:
  windows:
    provider: winget
    package_id: VERIFY_AGAINST_OFFICIAL_SOURCE
  linux:
    provider: flatpak
    package_id: VERIFY_AGAINST_OFFICIAL_SOURCE
  android:
    provider: fdroid
    package_id: VERIFY_AGAINST_OFFICIAL_SOURCE
verification:
  type: provider_installed_state
offline:
  usage: lan_without_internet
  installation: requires_verified_local_package
```

**Regla CI:** cualquier entrada marcada `VERIFY_*`, URL vacía, licencia sin decidir, ID ambiguo o proveedor no soportado **impide habilitar** instalación automática en una release.

## 4. CLI pública v0 y flujo de instalación

Nombres propuestos, sujetos a pruebas de usabilidad:

```text
suitectl doctor [--json]
suitectl catalog list [--platform windows|linux|android]
suitectl profiles list
suitectl plan --profile essential [--json]
suitectl apply --profile essential [--yes]
suitectl verify --profile essential [--offline] [--json]
suitectl status [--json]
suitectl export-profile --output profile.yaml
suitectl integrations list
```

**Secuencia de `apply`:** validar manifest → detectar entorno → consultar proveedores → resolver/mostrar plan → solicitar confirmación y permisos → ejecutar pasos independientes con registro → verificar estado → generar resumen. Sin `--yes`, cualquier operación que cambie el sistema debe pedir consentimiento. `--yes` no salta elevaciones, permisos Android ni validación de fuentes.

Requisitos funcionales concretos:

- Repetir `plan` con igual catálogo y estado observado produce el mismo plan, salvo datos externos explícitamente señalados.
- Repetir `apply` sin cambios deja todas las operaciones en `already_satisfied` o equivalente.
- No reinstalar aplicaciones porque su versión no coincide exactamente salvo política de actualización solicitada.
- El fallo de una aplicación no debe corromper el estado del resto; preservar errores por paso y resultado global no exitoso si hay requisitos fallidos.
- Nunca desinstalar, limpiar bibliotecas, sobrescribir documentos ni sincronizar carpetas de modo bidireccional como efecto de `apply` por defecto.
- La reversión solo es automática para operaciones locales explícitamente reversibles; no prometer rollback genérico de gestores externos.
- Cada proveedor expone errores normalizados: `unsupported`, `not_found`, `permission_required`, `network_unavailable`, `verification_failed`, `execution_failed`.
- Salidas CLI legibles y JSON versionado para CI, agentes y automatizaciones; códigos de salida documentados.

## 5. Datos locales, sincronización y recuperación

### Estructura portable del usuario

```text
BibliotecaPersonal/
├── Musica/
├── Videos/
├── Lecturas/
├── Documentos/
├── NotasExportadas/
├── OfflineEsencial/
├── Intercambio/
└── Instaladores/
```

- Ruta configurable, sin imponer cambios sobre carpetas existentes. Todas las operaciones de creación son no destructivas.
- Distinguir `biblioteca fuente`, `copia de dispositivo`, `bandeja de intercambio` y `backup`. Una carpeta sincronizada **no** equivale a backup.
- Mantener originales y formatos abiertos cuando resulte viable: FLAC/MP3/Opus, EPUB/PDF/CBZ, ODT/ODS, Markdown/JEX, ICS/VCF, M3U8 con rutas relativas, KDBX.
- **No** sincronizar archivos SQLite activos de Joplin ni abrir simultáneamente una biblioteca calibre en varios escritores. Usar sincronización/exportación soportada por cada aplicación.
- LocalSend: intercambio puntual en LAN. Syncthing: sincronización de carpetas solo por elección explícita, con versionado y ensayo de conflictos; cliente Android sujeto a validación independiente.
- Restic: backup cifrado a otro destino, con prueba periódica de restauración; no almacenar la única contraseña de recuperación en el equipo respaldado.
- El registro local de `suitectl` jamás es la fuente única de los datos del usuario: debe poder reconstruirse mediante `doctor`.

### Tres niveles de disponibilidad

1. **Uso offline:** aplicaciones instaladas y datos seleccionados accesibles sin Internet (objetivo de MVP para ficheros de muestra).
2. **LAN sin Internet:** transferencias locales y acceso a servicios en red disponibles, sin depender de DNS externo.
3. **Instalación/restauración offline:** kit específico por OS/arquitectura con paquetes y dependencias verificadas, manifiesto y derechos de redistribución revisados (objetivo posterior).

`verify --offline` debe comprobar recursos locales **sin depender de peticiones externas**. Una prueba real de desconexión debe complementar el diagnóstico lógico. Los contenidos Kiwix ZIM, mapas, modelos de Buzz y adjuntos Joplin deben descargarse por separado: instalar la aplicación no garantiza la disponibilidad de contenido.

## 6. Multisuite: integración opcional, no dependencia

### Contrato de interoperabilidad

Multisuite es **un proveedor posible** de servicios, no el plano de control obligatorio ni la fuente exclusiva de datos. Los conectores operan bajo un contrato genérico:

```text
Connector
  discover(config) -> supported|unavailable|untrusted
  capabilities()  -> lista de capacidades reales
  health()        -> available|degraded|offline
  authorize()     -> mecanismo soportado y autorizado
  export/import() -> solo si la API lo permite
  sync()          -> solo con políticas de conflictos explícitas
  disconnect()    -> eliminar vínculo sin eliminar archivos locales
```

No todas las operaciones estarán implementadas en todos los conectores. Un conector debe declarar capacidades soportadas y no simularlas.

### Casos de uso candidatos

| Dominio | Multisuite cuando está disponible | Comportamiento independiente/offline |
| --- | --- | --- |
| Música | Navidrome para catálogo y streaming; descarga/exportación mediante mecanismos validados | Archivos locales reproducibles en VLC/Audacious; playlists portables |
| Documentos | Paperless-ngx para OCR y archivo, si el servicio está desplegado y su API habilitada | PDFs y documentos locales; edición en LibreOffice |
| Inventario | HomeBox para inventario centralizado, cuando esté instalado y permita integración | Exportación local consultable si el formato y API lo permiten; sin prometer edición offline |
| Notas | Backend Joplin compatible (p. ej. WebDAV/servidor validado) | Notas y adjuntos ya descargados por el cliente; sincronización diferida |
| Archivos | Comparticiones/servicios explícitamente autorizados | `BibliotecaPersonal`, LocalSend y copias independientes |

**Seguridad del vínculo:** configuración explícita u autodetección restringida a servicios autorizados, TLS con verificación de identidad, sin aceptar certificados no confiables silenciosamente; credenciales solo en almacenamiento seguro disponible en el OS, nunca en YAML, logs, Git o URLs persistidas. Un fallo del NAS nunca bloqueará `doctor`, `plan`, `apply` local ni acceso a documentos previamente descargados.

**Compatibilidad:** validar versiones y endpoints reales del repositorio Multisuite antes de escribir adaptadores. No asumir que Paperless-ngx/HomeBox están desplegados por aparecer en el catálogo.

### Estado real de Multisuite (revisado 2026-11-11)

- Multisuite es un **portal privado Laravel/Livewire** que enlaza servicios existentes, cada uno con su interfaz y sus cuentas. Hoy integra **Navidrome** (música) y aprovisiona cuentas de Navidrome por invitación. No hay SSO.
- **No expone ninguna API para clientes como Sovereign.** El acceso remoto privado (VPN/TLS) sigue pendiente en su hito P03.
- Su ampliación prevista (P04) evalúa Komga (lectura), Jellyfin (vídeo) y un servicio de notas. Paperless-ngx y HomeBox **no** están en su hoja de ruta: la tabla anterior son candidatos de Sovereign, no servicios disponibles.
- Por tanto, H8 empieza con un **contrato escrito y acordado en ambos repositorios** (descubrimiento de servicios autorizados para el usuario, capacidades, salud y versión). El primer caso de uso es música: Sovereign propondría instalar un cliente compatible con la API Subsonic de Navidrome y mostrar la URL del servicio, **sin almacenar credenciales** de Multisuite. Detalle y prerrequisitos: [`integrations/multisuite/README.md`](integrations/multisuite/README.md).

## 7. Seguridad, privacidad y cadena de suministro

- Adherirse proporcionalmente a **NIST SSDF (SP 800-218)**, prácticas **OpenSSF** y principio de mínimo privilegio. Registrar amenazas y mitigaciones breves en `docs/security/threat-model.md`.
- Preferir repositorios/gestores oficiales. Validar identificador, publicador, licencia, canal de actualización y estado de mantenimiento; fijar versiones en bundles reproducibles.
- Descargar artefactos por HTTPS de orígenes confiables; verificar hashes contra manifiestos autenticados y firmas cuando el upstream las facilite. **SHA-256 por sí solo no autentica** si el checksum se obtiene de la misma fuente no confiable.
- Distribuir releases con checksums, procedencia y firma de proyecto; incluir SBOM (SPDX o CycloneDX) cuando se publique el binario.
- No ejecutar scripts remotos arbitrarios ni admitir URLs editables como comandos de shell. Invocar gestores con argumentos estructurados, sin interpolación insegura.
- Secretos fuera de repositorio/configuración exportable; redactar logs; no telemetría ni descubrimiento de red por defecto sin opt-in.
- `apply` tiene efecto en el sistema: plan visible y consentimiento. Para cambios de privilegio o incorporación de certificados usar confirmación específica.
- No mezclar gestión de paquetes con tratamiento automático de datos personales. Borrado/limpieza con Krokiet siempre manual y con previsualización.
- Para Android, respetar instalación y permisos asistidos por usuario; sin prometer instalación silenciosa ni excepciones al sandbox del sistema.
- Definir política de actualización de dependencias y respuesta a vulnerabilidades; publicar `SECURITY.md` antes de la primera release pública.

## 8. Calidad de ingeniería y criterios de entrega

### Convenciones de desarrollo

- Commits pequeños, PRs revisables, protección de rama principal, revisión obligatoria y ADR para cambios transversales.
- `README.md` de entrada rápida, `AGENTS.md` con comandos/arquitectura, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, licencia explícita y ejemplos de configuración sin secretos.
- Versionado semántico de CLI/releases y **versionado independiente de esquemas**; compatibilidad hacia atrás o migraciones verificadas.
- Núcleo organizado por interfaces para poder simular proveedores sin instalar software durante pruebas.
- Formateo automático, análisis estático, lint de shell/YAML, tests y build en CI.
- No sobreingeniería: no microservicios, no base de datos remota, no Kubernetes, no sistema de plugins descargables en el MVP.

### Pirámide de pruebas

| Nivel | Qué probar | Obligación mínima |
| --- | --- | --- |
| Unitarias | Resolución, elegibilidad OS, alternativas, idempotencia, errores, redacción de secretos | Ejecutar en cada PR; objetivo orientativo ≥80 % de cobertura de lógica de dominio, sin usarlo como sustituto de revisión |
| Contratos | Esquemas, manifiestos y adaptadores con respuestas simuladas | Cada app habilitada tiene paquete verificable y regla de detección |
| Integración | Plan + proveedor simulado o entorno efímero; estado persistente | Reejecución sin cambios; errores parciales controlados |
| E2E | Instalación real de perfil reducido Windows/Linux | Matriz en CI si el runner lo permite; si no, prueba reproducible y bloqueo de release |
| Android | Selección de apps, enlaces verificados, permisos y compatibilidad | Validación en emulador y muestra de dispositivo real antes de declarar soporte |
| Resiliencia | Sin Internet, NAS apagado, caída de descarga, actualización interrumpida | Comportamiento documentado y sin corrupción de datos |
| Recuperación | Exportación e importación de perfil; restauración de archivo de muestra desde backup | Restauración comprobada, no solo creación de copia |

### Gates de CI recomendados

1. `format` / `lint` / análisis estático y detección de secretos.
2. Validación de YAML y semántica del catálogo; IDs y fuentes verificados.
3. Test unitarios y de contratos, incluidos errores y reintentos.
4. Compilación Windows/Linux para arquitecturas soportadas.
5. Escaneo de vulnerabilidades y dependencias; revisión de licencias.
6. Pruebas de smoke de `doctor`, `plan` y `apply` con proveedor simulado.
7. En release: integración E2E, SBOM, hashes, firma y artefactos de procedencia.

**Definición de terminado (DoD) de toda PR:** código y pruebas pasando; documentación actualizada; cambios de contrato versionados; seguridad y privacidad revisadas; rutas de error implementadas; logs sin secretos; comandos de comprobación reproducibles; riesgos pendientes explícitos. No se fusiona por apariencia de funcionamiento.

## 9. Estructura del repositorio

Estructura **real** a 2026-11-11. Las carpetas marcadas *(futuro)* se crean solo cuando un hito las necesite; no anticiparlas.

```text
Sovereign/
├── README.md · AGENTS.md · CLAUDE.md · LICENSE · SECURITY.md · CONTRIBUTING.md · CHANGELOG.md
├── PLAN_TECNICO_SUITE_LIBRE.md      # este plan
├── embed.go                         # catálogo, perfiles y allowlist incrustados (ADR-0004)
├── go.mod / go.sum
├── cmd/suitectl/                    # CLI: argumentos, salida y mapeo de errores
├── internal/
│   ├── catalog/                     # carga y validación estricta de YAML
│   ├── planner/                     # resolución determinista de perfiles (sin efectos)
│   ├── install/                     # allowlist, WinGet/Flatpak, runner de procesos
│   ├── diagnostics/                 # doctor
│   ├── state/                       # (futuro, H3) journal local con escritura atómica y bloqueo
│   └── connectors/                  # (futuro, H8) conectores opcionales
├── catalog/apps/ · catalog/install-allowlist.yaml · profiles/ · schemas/
├── tools/validate_catalog.py · tests/            # contrato del catálogo (Python, CI)
├── bootstrap/                       # setup.sh / setup.ps1 (hoy: Go de usuario; H4: release verificada)
├── integrations/multisuite/ · android/           # notas de diseño; sin código hasta H8 / H6
├── docs/STATUS.md · docs/architecture/adr/ · docs/security/ · docs/recovery/
└── .github/workflows/
```

`AGENTS.md` contiene solo instrucciones operativas (lectura, comandos, invariantes, normas de PR) y enlaza aquí, sin duplicar el plan.

## 10. Hoja de ruta y entregables

> Orden revisado en [ADR-0005](docs/architecture/adr/0005-roadmap-release-first.md): primero un perfil instalable y una release que no exija Git ni Go; después datos, Android, kits offline y, al final, la integración opcional con Multisuite. **Una PR por incremento revisable**; un hito empieza cuando el anterior tiene evidencia registrada en `docs/STATUS.md`. El estado de cada hito vive en STATUS, no aquí.

| Hito | Alcance implementable | Evidencia de aceptación |
| --- | --- | --- |
| **H0 — Fundación** | Repositorio, licencia, AGENTS, ADR-0001, esquema v1, catálogo mínimo, CI de validación | Catálogo erróneo falla CI |
| **H1 — Planificación** | `doctor`, `catalog`, `profiles`, `plan`; detección de plataforma y proveedores | Plan determinista, JSON versionado, tests de alternativas e incompatibilidades |
| **H2 — Primera instalación** | LocalSend en WinGet y Flatpak: `apply` con consentimiento, `verify`, idempotencia; catálogo incrustado y allowlist (ADR-0003/0004) | Instalar, verificar y repetir sin duplicar en Windows y Linux reales |
| **H3 — Perfil esencial instalable** | Verificar e incorporar apps del perfil `essential` **una por PR** (VLC y LibreOffice primero); `apply --profile` con un único consentimiento sobre el plan mostrado, resumen por paso y fallos parciales visibles; journal local en `internal/state` (escritura atómica, bloqueo contra dos `apply` simultáneos, sin secretos); `status` | Perfil completo en un equipo limpio de cada SO; segunda ejecución sin cambios; fallo de una app no corrompe el resto; journal reconstruible con `doctor` |
| **H4 — Primera release distribuible** | Binarios Windows/Linux x86-64 y ARM64 generados en CI; ZIP con checksums, SBOM y procedencia firmada; bootstrap que descarga y verifica la release en lugar de Go; dependencias vendorizadas para compilar sin red | Usuario sin Git ni Go descarga, verifica y ejecuta `plan`/`apply`; build reproducible documentado |
| **H5 — Datos y portabilidad** | `BibliotecaPersonal` configurable y no destructiva; `export-profile`/import; guía de backup y restauración de muestra | No se sobrescriben archivos; export/import probado en otro equipo; restauración comprobada |
| **H6 — Android guiado** | Selección de perfil y fuentes F-Droid/Obtainium verificadas; guía de permisos; comprobaciones documentadas | Incorporación guiada en emulador y dispositivo real, sin instalación silenciosa |
| **H7 — Kits offline** | Kits por SO/arquitectura con paquetes verificados y derechos de redistribución revisados; recursos esenciales (p. ej. Kiwix) por elección explícita; `verify --offline` | Instalación de muestra sin red donde haya kit; uso offline comprobado sin NAS |
| **H8 — Conector Multisuite opcional** | Contrato acordado con el repositorio Multisuite (descubrimiento de servicios autorizados, capacidades, salud); primer caso: música vía Navidrome; credenciales en el almacén seguro del SO | Con Multisuite accesible: caso completo; apagado o sin VPN: flujo local intacto |
| **H9 — UX y guías** | TUI/GUI solo si los contratos están estables y compensa; guías de usuario Windows/Linux/Android | Checklist de aceptación de usuario final |

### Siguiente backlog ejecutable (H3)

**PR H3-1 — VLC verificado.** Contrastar IDs y fuentes oficiales de WinGet y Flathub con fecha; marcar `verified` en el catálogo; añadir entradas a la allowlist; tests con proveedor simulado; aceptación en dispositivo real antes de fusionar.

**PR H3-2 — LibreOffice verificado.** Igual que H3-1. Revisar tamaño y runtimes de Flatpak y avisarlo en el plan.

**PR H3-3 — `apply --profile` y journal.** Ejecutar solo entradas instalables del plan mostrado, con un consentimiento; continuar ante fallos de una app y devolver resultado global no exitoso; journal atómico con bloqueo. Probar interrupción y segunda ejecución.

**PR H3-4 — `status`.** Mostrar estado observado frente al deseado a partir del journal y los gestores, sin efectos.

No ampliar Android, kits offline ni Multisuite hasta cerrar H3 y H4.

## 11. Criterios de aceptación de la primera versión útil

- [ ] Un usuario puede descargar un ZIP verificado, ejecutar el instalador apropiado y ver el plan sin tener Git. *(H4)*
- [x] `suitectl plan --profile essential` no modifica sistema ni datos y explica por qué omite aplicaciones incompatibles. *(H1–H2: estados, motivo y `next_step`)*
- [ ] Windows y una distribución Linux objetivo instalan y verifican un subconjunto publicado del perfil esencial. *(H3; LocalSend ya lo cumple desde H2)*
- [ ] Segunda ejecución sin cambios no reinstala software ni pisa configuraciones. *(H3 para el perfil; cumplido para LocalSend en H2)*
- [ ] Los errores de instalación y permisos son claros; no se ocultan fallos parciales. *(H3)*
- [ ] Se crea o reutiliza `BibliotecaPersonal` sin pérdida de archivos preexistentes. *(H5)*
- [ ] Los archivos de muestra se abren sin Internet y sin Multisuite. *(H5)*
- [ ] Es posible exportar el perfil y reconstruir el plan en otro equipo compatible. *(H5)*
- [ ] El proyecto no solicita ni registra secretos, no envía telemetría y no necesita iniciar sesión. *(invariante desde H0; revisión formal en H4)*
- [ ] CI, revisión de seguridad y pruebas exigidas por la release están documentadas y ejecutadas. *(H4)*

**El soporte Android del MVP es guiado, no plenamente automático.** La automatización móvil completa, la sincronización entre aplicaciones y el kit de reinstalación totalmente offline pertenecen a hitos posteriores y no deben presentarse como ya resueltos.

## 12. Riesgos y decisiones pendientes, sin bloquear el inicio

| Riesgo/decisión | Mitigación o valor por defecto |
| --- | --- |
| Múltiples distros Linux | Soportar explícitamente 1 familia al principio y Flatpak cuando encaje; ampliar con pruebas |
| Disponibilidad real de paquetes | Catálogo con estado `verified/unsupported/pending`; impedir instalación si está pendiente |
| Instalación sin permisos elevados | Preferir ámbito usuario cuando sea posible; comunicar elevación inevitable |
| Android heterogéneo | Flujo guiado y pruebas reales; no prometer instalación silenciosa |
| Variación de APIs de Multisuite | Primer conector tras inspeccionar servicio concreto y versión desplegada |
| Aplicaciones sin exportación | Declarar soporte de solo instalación y recuperación limitada; no inventar interoperabilidad |
| Fallos de red o proveedor | Operaciones reintentables y registro consistente; funcionamiento local independiente |
| Dependencias y licencias | Auditoría antes de empaquetar; terceros fuera del bundle cuando no pueda redistribuirse |
| Exceso de trabajo inicial | CLI + un perfil + una app real primero, UI y servicios después |
| Cambio de identificadores upstream | Comprobaciones periódicas del catálogo y PR de actualización revisada |

## 13. Referencias oficiales y procedencia

**Origen funcional:** `PLAN_SUITE_LIBRE_OFFLINE.md`, documento externo no versionado aquí (objetivos, catálogo inicial, estructura `BibliotecaPersonal`, límites de offline y pruebas de aceptación). Lo necesario está incorporado en este plan. Lo implementado y validado se registra en `docs/STATUS.md`; el resto de este documento sigue siendo propuesta.

Referencias de implementación y seguridad:

- [Microsoft WinGet — import](https://learn.microsoft.com/en-us/windows/package-manager/winget/import)
- [Flatpak — uso](https://docs.flatpak.org/en/latest/using-flatpak.html)
- [Flathub — instalación por usuario o por sistema](https://docs.flathub.org/docs/for-users/user-vs-system-install)
- [Android Developers — permisos e instalación de paquetes](https://developer.android.com/reference/android/Manifest.permission)
- [NIST Secure Software Development Framework, SP 800-218](https://csrc.nist.gov/pubs/sp/800/218/final)
- [SLSA — niveles y procedencia](https://slsa.dev/)
- [OpenSSF Scorecard](https://securityscorecards.dev/)
- [SPDX](https://spdx.dev/) / [CycloneDX](https://cyclonedx.org/)

---

## Orden de ejecución autorizado por este plan

**H0 → H1 → H2 completados. Seguir H3 → H4 → H5 → H6 → H7 → H8 → H9** ([ADR-0005](docs/architecture/adr/0005-roadmap-release-first.md)). Cada incremento conserva seguridad, reproducibilidad, datos locales y ausencia de dependencia obligatoria de Multisuite o de cualquier servicio de red.
