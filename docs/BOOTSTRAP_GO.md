# Bootstrap de Go — Windows y Linux

Estado: soporte inicial de desarrolladores, incluido en la PR H2. **No es necesario instalar Go para ejecutar futuros binarios compilados de Sovereign**; los ZIPs de release del hito H4 contendrán un ejecutable autónomo y el bootstrap pasará a descargar y verificar esa release ([ADR-0005](architecture/adr/0005-roadmap-release-first.md)).

## Comportamiento

- Si `go version` informa una versión compatible con `go.mod` (>= 1.23) en la plataforma actual, reutiliza el Go existente y **no descarga nada**.
- De lo contrario, instala **Go 1.27.3** exclusivamente en el espacio del usuario, sin MSI ni `sudo`. En Windows, después de validar Go, el instalador añade su `bin` al **PATH de usuario** de forma idempotente. No modifica el PATH del sistema; con `-NoPath` se evita ese cambio. En Linux permanece sin modificar el PATH.
- Linux: `${XDG_DATA_HOME:-$HOME/.local/share}/sovereign/toolchains/go1.27.3`. Windows: `%LOCALAPPDATA%\Sovereign\toolchains\go1.27.3`.
- Arquitecturas soportadas: Windows x86-64/ARM64 y Linux x86-64/ARM64. Otras arquitecturas fallan explícitamente, sin descargar contenido desconocido.
- Fuente: exclusivamente HTTPS `https://go.dev/dl/go1.27.3.<os>-<arch>.<zip|tar.gz>`. Los hashes SHA-256 de cada archivo están fijados en scripts y fueron contrastados con [la distribución oficial de Go](https://go.dev/dl/) (2026-11-10).
- **Windows:** el directorio local `bin` de Go se publica en HKCU/PATH únicamente si es un ejecutable validado y se evita duplicar la entrada. Para poder ejecutar `go` desde una ventana PowerShell previamente abierta, hay que abrir una **nueva terminal** o actualizar el PATH de esa sesión manualmente. Los procesos hijo (por ejemplo `pwsh -File`) no pueden alterar el entorno de su consola padre.
- No ejecuta scripts remotos. El binario de Go solo se ejecuta tras verificar hash y extraer el archivo oficial.
- Archivos temporales se eliminan; la versión se instala en una ruta versionada y no pisa otras instalaciones. Una caché parcial o incompatible provoca un error explícito.
- Al finalizar ejecuta `suitectl doctor`; el administrador puede usar después `suitectl plan`, `verify` o `apply` con sus propios requisitos de consentimiento.
- `--offline` (Linux) / `-Offline` (Windows) impide descargar Go; también deshabilita la resolución online de módulos de Go. Si `gopkg.in/yaml.v3` aún no está en caché, `go run` fallará con un error claro. Para una experiencia offline sin módulos, se necesita un futuro binario de release o bundle local.
- No se instala el gestor de paquetes WinGet o Flatpak: su configuración permanece fuera del alcance y requiere autorización distinta.

## Ejemplos

Linux:

```sh
git clone https://github.com/sempitern0/Sovereign.git
cd Sovereign
sh bootstrap/setup.sh --json
sh bootstrap/setup.sh --offline
```

Windows PowerShell:

```powershell
git clone https://github.com/sempitern0/Sovereign.git
cd Sovereign
.\bootstrap\setup.ps1 -Json
.\bootstrap\setup.ps1 -Offline
.\bootstrap\setup.ps1 -NoPath   # evita la persistencia en PATH de usuario
```

## Checklist de aceptación en máquinas limpias

1. Con Go >= 1.23 ya en PATH: no descargar ni alterar PATH, ejecutar doctor. Si es un Go local ya instalado fuera de PATH, publicarlo en el PATH de usuario.
2. Sin Go: descargar Go oficial, verificar SHA-256, instalar sin elevación y ejecutar doctor.
3. Repetir: reutilizar instalación local sin volver a descargar.
4. Sin conexión ni Go: falla con código no cero y no intenta acceder a Internet.
5. Archivo manipulado/hash incorrecto: no ejecutar ni instalar su contenido.
6. Caché parcial preexistente: no sobrescribirla, informar del problema.
7. Windows x86-64 y ARM64; Linux x86-64 y ARM64: validar arquitectura y rutas reales.
8. Confirmar que ningún script pide `sudo`, UAC ni escribe en directorios del sistema.
9. Windows: comprobar idempotencia del PATH de usuario, nueva terminal y `-NoPath`; restaurar el PATH original tras las pruebas.

En CI se verifican la sintaxis y la ejecución con Go preinstalado. **La descarga en equipos sin Go aún exige QA real**; no se ha realizado una instalación real de un toolchain en ambos sistemas dentro de esta PR.
