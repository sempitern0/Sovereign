# Biblioteca offline — plan de integración

Sovereign distingue **aplicación**, **paquete de datos** y **verificación del contenido**. Instalar un lector no hace que Wikipedia o los mapas estén disponibles sin Internet.

## Kiwix / Wikipedia
- Upstream: https://www.kiwix.org/ y https://github.com/kiwix/kiwix-desktop (GPL-3.0).
- Formato de datos: `.zim`, elegido por idioma/tema (Wikipedia, Wikivoyage, Wikibooks, etc.). Catálogo oficial de Kiwix: https://library.kiwix.org/.
- Linux: existe el manifiesto de Flathub `org.kiwix.desktop`. No se ha añadido a la allowlist de instalación hasta validar su mantenimiento, procedencia y versión real. Windows queda como instalación manual pendiente.
- Flujo futuro: seleccionar idioma y contenido, ver tamaño/espacio disponible, descargar en partes con reanudación, checksum de origen verificado, guardar en `BibliotecaPersonal/Conocimiento/ZIM`, indexar localmente y permitir exportar sin nube.

## CoMaps / mapas
- Upstream: https://www.comaps.app/; código Apache-2.0 en https://codeberg.org/comaps/comaps.
- Hay manifiesto Flatpak público `app.comaps.comaps`. Su publicación estable y soporte requieren QA; no se habilita instalación en H2 sin pruebas.
- La disponibilidad offline **requiere descargar regiones** dentro de la aplicación. Los paquetes de mapas y sus versiones son independientes del instalador.
- Flujo futuro: seleccionar regiones (con estimación de espacio y caducidad), verificar archivos descargados mediante fuentes oficiales, exportar/importar mapas locales, no asumir que los archivos `.mwm` son portables entre versiones.

## Estado actual
- Kiwix y CoMaps figuran en `catalog/apps` y como opciones futuras del asistente y del catálogo.
- No se han añadido a `catalog/install-allowlist.yaml`; por tanto **Sovereign no los instala aún**.
- Tampoco se descargan ZIM, mapas ni bibliotecas de forma silenciosa.
- La opción de menú explica cómo obtener contenidos y qué queda pendiente, sin prometer acceso offline antes de tener los datos.

## Criterios antes de habilitar cada descarga
1. Fuente oficial, versión, licencia de la app y del contenido.
2. Integridad SHA-256 o firma de origen y control de descarga incompleta.
3. Espacio, red y cuotas, tamaño visible y consentimiento.
4. Instalación y lectura probadas sin conexión en Windows/Linux donde corresponda.
5. Preservar archivos del usuario, exportación y backup, sin depender de Multisuite.
