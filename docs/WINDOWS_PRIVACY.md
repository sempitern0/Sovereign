# Privacidad Windows — alcance seguro

Estado: **auditoría de solo lectura**. Sovereign aún no cambia servicios, procesos, tareas programadas ni directivas. No presenta ninguna configuración como una garantía de eliminar toda la telemetría.

## Auditoría

```powershell
.\sovereign.ps1 privacy audit
.\sovereign.ps1 privacy audit --json
```

Consulta de forma local y sin elevación:
- Identificador publicitario en HKCU.
- Preferencias de experiencias personalizadas en HKCU.
- Directiva de datos de diagnóstico del dispositivo en HKLM, cuando exista.
- Estado del servicio `DiagTrack` (solo inventario, no modificación).
- Servicios opcionales asociados a Xbox como elementos informativos; no se asume que sean innecesarios.

Las claves pueden no existir o no aplicar según edición, versión, GPO/MDM o políticas de una organización. `unknown` significa desconocido, no activo ni desactivado.

## Herramientas externas
Sovereign no cataloga, descarga ni ejecuta herramientas externas de ajuste de Windows. WinHance se retiró del catálogo el 2026-11-21 por decisión del mantenedor: su licencia PolyForm Shield 1.0.0 no es libre y la herramienta modifica el sistema, lo que choca con la auditoría de solo lectura.

## Futuro módulo `privacy plan/apply/restore`
Cada ajuste requerirá un identificador estable, compatibilidad Windows 10/11, nivel de riesgo `balanced/advanced` por ajuste, sin perfiles de instalación, impacto, detección precisa, copia de seguridad, comando de aplicación, verificación y reversión ensayada. Nunca cambiar silenciosamente:
- Microsoft Defender, SmartScreen, firewall, UAC, Windows Update, BitLocker o Secure Boot.
- Servicios de red, DNS, VPN, logs de seguridad, EDR, acceso remoto, componentes de dominio/MDM o funciones de accesibilidad.
- Procesos del sistema basándose solo en su nombre o uso momentáneo de CPU.
- Políticas organizativas ni configuración administrada sin autorización.

Antes de implementar ajustes mutables: aprobación humana por acción, elevación explicada y separada, reversiones verificadas en VM Windows 10/11, tests de fallos parciales y aviso de reinicio. Hay configuraciones que Windows puede reponer tras una actualización; no prometer erradicación permanente.
