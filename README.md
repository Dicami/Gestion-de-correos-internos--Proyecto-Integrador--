# CorpLink - Gestion de Correos Internos 

Sistema de comunicación interna y correo empresarial cliente-servidor desarrollado en **Go** e integrado con un servicio de autenticación en FastAPI. Esta solución permite la interacción segura, autenticada y trazable mediante bandejas de entrada, envío de mensajes y gestión de correspondencia local entre los colaboradores de la Sede A y Sede B.

---

## Descripción del Sistema

La solución responde a la problemática de la empresa tecnológica tras la apertura de su segunda sede operativa (Sede B). Anteriormente, la coordinación entre sedes se realizaba a través de canales informales y no autenticados, generando riesgos de fuga de información, falta de auditoría y problemas de trazabilidad.

Connecta centraliza la comunicación a través de:

- **Servidor de Correo (Go):** Gestiona la lógica cliente-servidor, procesa rutas REST para el almacenamiento, consulta de bandejas (entrada y enviados), lectura y eliminación lógica de correos institucionales de manera completamente independiente de un servicio SMTP externo.
- **Servicio de Autenticación (FastAPI):** Gestiona la verificación de credenciales corporativas (contraseñas con hash bcrypt) y la emisión de tokens JWT.
- **Cliente de Consola (Go):** Interfaz CLI liviana que permite a los colaboradores redactar correos, consultar su bandeja de entrada, revisar elementos enviados y gestionar sus mensajes de forma local.

---

## Integrantes del Equipo

- Emilio Cuenca
- Danna Simaluisa
- Camila Villagran

---

## Alcance Preliminar

### Incluido en el Proyecto
- **Autenticación Segura:** Inicio de sesión con correo corporativo y contraseñas protegidas mediante hash bcrypt. Emisión de tokens JWT con roles (Usuario y Administrador).   
- **Gestión de Correos Internos:** Bandeja de entrada, mensajes enviados, redacción, lectura de correspondencia y eliminación de correos de forma local y privada.   
- **Control de Sesiones:** Expiración automática de sesión, cierre manual y registro de timestamps de inicio/fin.   
- **Infraestructura Distribuida:** Despliegue de los servicios en la VLAN de servidores (VLAN 20) en ambas sedes, comunicándose mediante enrutamiento dinámico OSPF sobre OpenWrt.   
- **Componente Estadístico:** Recolección de eventos de uso y métricas técnicas para su análisis descriptivo.

### Fuera del Alcance
- **Servidores externos:** Integración con servidores SMTP públicos o externos.
- **Archivos adjuntos:** Envío de archivos adjuntos pesados, imágenes o documentos multimedia complejos.
- **Comunicación en tiempo real:** Llamadas de voz, video o mensajería instantánea en tiempo real vía WebSockets.
- **Interfaz gráfica:** Interfaz gráfica (GUI) o aplicaciones móviles (el cliente opera exclusivamente por consola).
- **Infraestructura en la nube:** Alta disponibilidad avanzada y balanceo de carga en la nube.

---

## Estructura del Repositorio

```text
Gestion de correos /
├── app-go/                  # Cliente y Servidor en Go
├── auth-service/            # Servicio de autenticación en FastAPI (Python)
├── network/                 # Configuración de red y reglas de comunicación inter-sedes
├── research/                # Investigaciones y análisis previos
├── docs/                    # Documentación del proyecto
│   └── especificacion-requisitos.pdf  # Análisis, especificación de requisitos, casos de uso, matriz y guion
├── tests/                   # Pruebas de integración y validación
├── docker-compose.yml       # Orquestación de contenedores (Servidor Go, FastAPI, BD)
├── .gitignore
├── LICENSE
└── README.md
