# Biblioteca offline — plan de integración

Sovereign distingue **aplicación**, **paquete de datos** y **verificación del contenido**. Instalar un lector no hace que Wikipedia o los mapas estén disponibles sin Internet.

## Kiwix / Wikipedia
- Upstream: https://www.kiwix.org/ y https://github.com/kiwix/kiwix-desktop (GPL-3.0).
- Formato de datos: `.zim`, elegido por idioma/tema (Wikipedia, Wikivoyage, Wikibooks, etc.). Catálogo oficial de Kiwix: https://library.kiwix.org/.
- **Windows:** WinGet `Kiwix.KiwixJS.Electron` verificado el 2026-11-21 (3.9.3-E, publisher Kiwix, ámbito usuario, sin UAC ni dependencias). Kiwix Desktop no está publicado en WinGet; Kiwix JS es el lector oficial de Kiwix y descarga ZIM desde la propia app. Forma parte del setup único; falta aceptación real.
- **Linux:** Flathub `org.kiwix.desktop` 2.4.1 (Kiwix Desktop) habilitado el 2026-11-22 por decisión del mantenedor, con riesgo aceptado: sin actualizaciones desde dic-2024 y runtime KDE 5.15-23.08 sin soporte. Revisar cuando Kiwix publique un Flatpak nuevo.
- Flujo futuro: seleccionar idioma y contenido, ver tamaño/espacio disponible, descargar en partes con reanudación, checksum de origen verificado, guardar en `BibliotecaPersonal/Conocimiento/ZIM`, indexar localmente y permitir exportar sin nube.

## CoMaps / mapas
- Upstream: https://www.comaps.app/; código Apache-2.0 en https://codeberg.org/comaps/comaps.
- **Linux:** Flathub `app.comaps.comaps` verificado el 2026-11-21 (estable 2026.11.07, compilado desde la etiqueta oficial; registrado como `community` al no constar verificación de Flathub). Forma parte del setup único; falta aceptación real.
- **Windows:** WinGet `CoMaps.CoMaps` 2026.06.13 (x64, dependencia VC++ x64) habilitado el 2026-11-22 con `scope: unspecified`: el manifiesto no declara ámbito, Sovereign no pasa `--scope` y avisa de que Windows puede pedir administrador.
- La disponibilidad offline **requiere descargar regiones** dentro de la aplicación. Los paquetes de mapas y sus versiones son independientes del instalador.
- Flujo futuro: seleccionar regiones (con estimación de espacio y caducidad), verificar archivos descargados mediante fuentes oficiales, exportar/importar mapas locales, no asumir que los archivos `.mwm` son portables entre versiones.

## Estado actual
- CoMaps y Kiwix están en la allowlist y en el setup único en Windows y Linux.
- Instalar la app no descarga mapas: el contenido offline requiere un paso aparte.
- Tampoco se descargan ZIM, mapas ni bibliotecas de forma silenciosa.
- La opción de menú explica cómo obtener contenidos y qué queda pendiente, sin prometer acceso offline antes de tener los datos.

## Criterios antes de habilitar cada descarga
1. Fuente oficial, versión, licencia de la app y del contenido.
2. Integridad SHA-256 o firma de origen y control de descarga incompleta.
3. Espacio, red y cuotas, tamaño visible y consentimiento.
4. Instalación y lectura probadas sin conexión en Windows/Linux donde corresponda.
5. Preservar archivos del usuario, exportación y backup, sin depender de Multisuite.
