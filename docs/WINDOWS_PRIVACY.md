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

## WinHance
- Sitio/código oficial: https://github.com/memstechtips/Winhance
- Licencia: **PolyForm Shield 1.0.0**, código público con restricciones de redistribución competitiva. No equivale a una licencia libre OSI; es una **herramienta externa opcional**.
- WinHance incluye personalización, eliminación de aplicaciones y ajustes de Windows; no ejecutar instaladores mediante `irm ... | iex` desde Sovereign. Requiere revisión humana del instalador, procedencia y cambios propuestos.
- La publicación de WinHance en el catálogo **no autoriza ninguna modificación**.

## Futuro módulo `privacy plan/apply/restore`
Cada ajuste requerirá un identificador estable, compatibilidad Windows 10/11, perfil `balanced/advanced`, impacto, detección precisa, copia de seguridad, comando de aplicación, verificación y reversión ensayada. Nunca cambiar silenciosamente:
- Microsoft Defender, SmartScreen, firewall, UAC, Windows Update, BitLocker o Secure Boot.
- Servicios de red, DNS, VPN, logs de seguridad, EDR, acceso remoto, componentes de dominio/MDM o funciones de accesibilidad.
- Procesos del sistema basándose solo en su nombre o uso momentáneo de CPU.
- Políticas organizativas ni configuración administrada sin autorización.

Antes de implementar ajustes mutables: aprobación humana por acción, elevación explicada y separada, reversiones verificadas en VM Windows 10/11, tests de fallos parciales y aviso de reinicio. Hay configuraciones que Windows puede reponer tras una actualización; no prometer erradicación permanente.
