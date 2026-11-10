# H2 — LocalSend, primer flujo de instalación segura

Estado: **experimental, pendiente de QA real en Windows y Linux**. El código de CI simula los gestores y no instala software.

## Proveniencia verificada el 2026-11-10

- Proyecto upstream: https://localsend.org/ y https://github.com/localsend/localsend
- **Windows / WinGet:** ID exacto `LocalSend.LocalSend`, manifiesto del repositorio comunitario oficial de Microsoft, versión revisada 1.18.2: https://github.com/microsoft/winget-pkgs/blob/master/manifests/l/LocalSend/LocalSend/1.18.2/LocalSend.LocalSend.yaml
- **Linux / Flathub:** ID `org.localsend.localsend_app`, nombre confirmado por https://localsend.org/download y https://flathub.org/apps/org.localsend.localsend_app ; fuente del manifiesto https://github.com/flathub/org.localsend.localsend_app
- WinGet y Flatpak son distribuidores de terceros; la aprobación del ID y del origen no equivale a una auditoría completa de binarios futuros.
- Android no se habilita en H2. VLC y LibreOffice permanecen pendientes.

## Primer uso

En el equipo **objetivo**, clonar el repositorio y ejecutar, desde su raíz:

```sh
go run ./cmd/suitectl doctor
go run ./cmd/suitectl plan --profile essential --json
go run ./cmd/suitectl verify --app localsend
go run ./cmd/suitectl apply --app localsend --yes
go run ./cmd/suitectl verify --app localsend
go run ./cmd/suitectl apply --app localsend --yes
```

**Importante:** `apply --app localsend` sin `--yes` no invoca gestores ni instala nada. `--yes` es una autorización explícita para **este** paquete, no para todo el perfil. El gestor puede pedir acuerdos adicionales. No se aceptan automáticamente acuerdos de WinGet, ni se eleva el proceso de forma silenciosa.

### Windows

- Requiere `winget` instalado y el repositorio de Microsoft `winget` configurado.
- Consulta `winget list --id LocalSend.LocalSend --exact --source winget --disable-interactivity`.
- Trata únicamente el HRESULT documentado `0x8A150014` como ausencia. Errores imprevistos detienen el flujo.
- Instala mediante `winget install --id LocalSend.LocalSend --exact --source winget --no-upgrade --scope user`; no instala actualizaciones deliberadamente. Solicitudes UAC y acuerdos pertenecen al gestor.
- Evitar ejecutar pruebas reales con derechos elevados; comprobar que el alcance final es de usuario.

### Linux

- Requiere Flatpak y un remoto **de usuario** `flathub` configurado con la URL oficial `https://dl.flathub.org/repo/`.
- Sovereign comprueba el remoto; no ejecuta `remote-add` ni solicita permisos para añadirlo.
- Detecta instalación existente mediante `flatpak list --app --columns=application` (alcances user/system).
- Instala mediante `flatpak install --user --noninteractive flathub org.localsend.localsend_app`, y verifica mediante la misma consulta local.
- Instalar el runtime requerido puede descargar dependencias de Flathub. No se garantiza reinstalación offline en H2.

## Escenarios de aceptación pendientes en dispositivo real

1. Equipo limpio: `verify` falla de forma reconocible sin modificar el equipo.
2. Sin consentimiento: `apply` no llama al gestor.
3. Instalación autorizada: comprobar app LocalSend y permisos, luego `verify`.
4. Segunda ejecución: `already installed`, sin instalación adicional.
5. Remoto erróneo o ausente, descarga interrumpida o administrador que cancela la instalación: error visible, sin declarar éxito.
6. Windows: revisar salida localizada de `winget list` en castellano e inglés y resultado exacto de instalación.
7. Linux: probar instalación de sistema preexistente y que no se duplica en alcance de usuario.
8. Windows y Linux: confirmar que el proceso no lee Multisuite ni guarda credenciales.

Los tests de CI cubren el contrato con un proveedor falso; no sustituye esta prueba en máquinas reales. H2 **no** autoriza desplegar masivamente la suite.
