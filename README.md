# MaiZap - Gestión de Correos Internos

Prototipo funcional de un **sistema de correos internos** con arquitectura **cliente-servidor**, desarrollado en **Go** para el Proyecto Integrador de tercer semestre (Programación Orientada a Objetos, semana 10).

El servidor y el cliente son programas independientes que se comunican mediante HTTP y JSON. Los colaboradores de la Sede A y la Sede B pueden registrarse, identificarse, redactar mensajes, consultar su bandeja de entrada y sus enviados, leer mensajes y eliminarlos de su buzón, sin depender de servicios SMTP externos.

---

## Tabla de contenido

1. [Descripción de la aplicación](#descripción-de-la-aplicación)
2. [Funcionalidades implementadas](#funcionalidades-implementadas)
3. [Requisitos de software](#requisitos-de-software)
4. [Instalación](#instalación)
5. [Iniciar el servidor](#iniciar-el-servidor)
6. [Ejecutar el cliente](#ejecutar-el-cliente)
7. [Dirección IP, puerto y parámetros configurables](#dirección-ip-puerto-y-parámetros-configurables)
8. [Datos y archivos de prueba](#datos-y-archivos-de-prueba)
9. [Guion de demostración](#guion-de-demostración)
10. [API del servidor](#api-del-servidor)
11. [Estructura de carpetas](#estructura-de-carpetas)
12. [Limitaciones conocidas](#limitaciones-conocidas)
13. [Autores y contribuciones](#autores-y-contribuciones)
14. [Créditos, licencia y declaración de uso de IA](#créditos-licencia-y-declaración-de-uso-de-ia)

---

## Descripción de la aplicación

MaiZap reemplaza los canales informales de comunicación entre las dos sedes de la empresa con un canal corporativo propio y trazable:

- **Servidor (`cmd/server`)**: recibe las solicitudes, valida los datos, aplica las reglas de negocio (destinatarios válidos, permisos sobre cada mensaje, eliminación lógica) y guarda la información en un archivo JSON.
- **Cliente (`cmd/client`)**: aplicación de consola con menús. Envía solicitudes HTTP al servidor y muestra las respuestas y los errores en lenguaje claro.
- **Identificación**: en esta fase el usuario se identifica con su correo corporativo y recibe un token de sesión temporal. **No se implementa un sistema de contraseñas en Go**, porque esa responsabilidad pertenece al servicio FastAPI de la asignatura de Fundamentos de Seguridad. El servidor usa la interfaz `services.Authenticator`, y en las semanas 11 y 12 la implementación local se reemplazará por un adaptador que consuma ese servicio.

## Funcionalidades implementadas

| # | Funcionalidad | Cliente (menú) | Servidor |
|---|---------------|----------------|----------|
| 1 | Registrar usuario | `1) Registrar usuario` | `POST /api/v1/users` |
| 2 | Identificar usuario e iniciar sesión | `2) Identificarse` | `POST /api/v1/sessions` |
| 3 | Crear y enviar un mensaje interno (uno o varios destinatarios) | `1) Redactar y enviar mensaje` | `POST /api/v1/messages` |
| 4 | Consultar la bandeja de entrada (con marca de no leídos) | `2) Bandeja de entrada` | `GET /api/v1/mailbox/inbox` |
| 5 | Consultar los mensajes enviados | `3) Mensajes enviados` | `GET /api/v1/mailbox/sent` |
| 6 | Leer el contenido de un mensaje (lo marca como leído) | `4) Leer mensaje` | `GET /api/v1/messages/{id}` |
| 7 | Eliminar un mensaje del buzón propio (eliminación lógica) | `5) Eliminar mensaje` | `DELETE /api/v1/messages/{id}` |
| 8 | Consultar el directorio de usuarios | `6) Directorio de usuarios` | `GET /api/v1/users` |
| 9 | Desactivar un usuario (solo administrador) | `7) Desactivar usuario` | `POST /api/v1/users/{id}/deactivate` |
| 10 | Cerrar sesión | `8) Cerrar sesión` | `DELETE /api/v1/sessions/current` |

**Validaciones y manejo de errores**

| Situación | Respuesta |
|-----------|-----------|
| Datos vacíos o incompletos (nombre, correo, sede, área, asunto, cuerpo) | `400`, con la lista de todos los campos con problemas |
| Correo fuera del dominio `@maizap.com` | `400` |
| Destinatario inexistente o desactivado, o envío a uno mismo | `400`, con el detalle de cada destinatario rechazado |
| Asunto de más de 150 caracteres, cuerpo de más de 5000 o más de 20 destinatarios | `400` |
| Opción de menú inexistente | El cliente muestra «la opción "9" no existe» |
| Usuario, mensaje, carpeta o ruta no encontrados | `404` |
| Correo ya registrado | `409` |
| Solicitud sin token, con token inválido, revocado o expirado | `401` |
| Leer o eliminar un mensaje ajeno, desactivar usuarios sin ser administrador o iniciar sesión con una cuenta desactivada | `403` |
| Formato no permitido: `Content-Type` distinto de `application/json`, JSON mal formado, campos desconocidos o cuerpo mayor a 64 KB | `400` |
| Fallo de lectura o escritura del archivo de datos | `500` con un mensaje genérico; el detalle queda en el log del servidor |
| Servidor apagado, IP o puerto incorrectos, o tiempo de espera agotado | El cliente muestra `[Error de conexión] ...` y sigue funcionando |
| IP o puerto inválidos al iniciar | El programa termina con un mensaje claro (por ejemplo, `el puerto 99999 está fuera del rango 1-65535`) |

El proyecto no usa `panic` para manejar errores. Todas las funciones que pueden fallar devuelven un valor de tipo `error`.

## Requisitos de software

| Software | Versión |
|----------|---------|
| Go | **1.22 o superior** (se usan los patrones de rutas de `net/http` introducidos en 1.22). Probado con **Go 1.27.3**. |
| Git | Cualquier versión reciente |
| Sistema operativo | Linux, macOS o Windows (PowerShell o CMD) |

No se necesitan dependencias externas, porque el proyecto usa solo la biblioteca estándar de Go. Tampoco se necesitan base de datos ni Docker.

Para verificar la versión de Go:

```bash
go version
```

## Instalación

```bash
git clone https://github.com/Dicami/Gestion-de-correos-internos--Proyecto-Integrador--.git
cd Gestion-de-correos-internos--Proyecto-Integrador--
git checkout emilio-cuenca
cd app-go
go build ./...
go test ./...
```

Opcionalmente, se pueden compilar los ejecutables:

```bash
go build -o bin/maizap-server ./cmd/server
go build -o bin/maizap-client ./cmd/client
```

En Windows se agrega la extensión: `go build -o bin\maizap-server.exe .\cmd\server`.

## Iniciar el servidor

Ejecute los comandos desde la carpeta `app-go` en una terminal:

```bash
go run ./cmd/server
```

Salida esperada:

```text
2026/11/09 18:35:34 MaiZap servidor escuchando en http://127.0.0.1:8080
2026/11/09 18:35:34 Datos: data/maizap.json | Sesiones expiran tras 30m0s de inactividad
```

El servidor registra cada solicitud (origen, método, ruta, estado y duración) sin guardar el contenido de los mensajes. Se detiene con `Ctrl+C` y el cierre es ordenado.

Para aceptar clientes de otros equipos de la red, por ejemplo en el laboratorio:

```bash
go run ./cmd/server -host 0.0.0.0 -port 9000
```

## Ejecutar el cliente

En **otra terminal**, también desde `app-go`:

```bash
go run ./cmd/client
```

Para conectarse a un servidor en otro equipo:

```bash
go run ./cmd/client -host 192.168.1.20 -port 9000
```

Al iniciar, el cliente comprueba la conexión con `GET /health`. Si el servidor no responde, lo informa y permite reintentar desde la opción `3) Probar conexión con el servidor`.

**Uso básico**

1. `2) Identificarse` con un correo de prueba, por ejemplo `ana.torres@maizap.com`.
2. `1) Redactar y enviar mensaje`. Los destinatarios se separan con comas. El cuerpo puede ocupar varias líneas y termina con una línea que contenga solo un punto (`.`).
3. `2) Bandeja de entrada` y `3) Mensajes enviados` muestran el ID de cada mensaje (por ejemplo, `MSG-000003`), que se usa en `4) Leer` y `5) Eliminar`.

## Dirección IP, puerto y parámetros configurables

Cada parámetro se puede indicar con una **opción de línea de comandos** o con una **variable de entorno**. La opción de línea de comandos tiene prioridad.

**Servidor**

| Opción | Variable de entorno | Valor por defecto | Descripción |
|--------|---------------------|-------------------|-------------|
| `-host` | `MAIZAP_HOST` | `127.0.0.1` | IP en la que escucha (`0.0.0.0` para todas las interfaces) |
| `-port` | `MAIZAP_PORT` | `8080` | Puerto TCP (1-65535) |
| `-data` | `MAIZAP_DATA` | `data/maizap.json` | Archivo JSON donde se guardan los datos |
| `-seed` | `MAIZAP_SEED` | `data/seed.json` | Datos iniciales, copiados solo si el archivo de datos no existe |
| `-session-ttl` | `MAIZAP_SESSION_TTL` | `30m` | Inactividad máxima antes de que expire una sesión (mínimo `1m`) |

**Cliente**

| Opción | Variable de entorno | Valor por defecto | Descripción |
|--------|---------------------|-------------------|-------------|
| `-host` | `MAIZAP_SERVER_HOST` | `127.0.0.1` | IP del servidor |
| `-port` | `MAIZAP_SERVER_PORT` | `8080` | Puerto del servidor |
| `-timeout` | `MAIZAP_TIMEOUT` | `5s` | Tiempo máximo de espera por solicitud |

Para ver la ayuda:

```bash
go run ./cmd/server -h
```

```bash
go run ./cmd/client -h
```

Ejemplo con variables de entorno en Linux o macOS:

```bash
MAIZAP_PORT=9000 go run ./cmd/server
```

Ejemplo en PowerShell:

```powershell
$env:MAIZAP_PORT="9000"; go run ./cmd/server
```

## Datos y archivos de prueba

`app-go/data/seed.json` contiene datos ficticios para la demostración. En el primer arranque el servidor lo copia a `app-go/data/maizap.json`. Este segundo archivo es el que se modifica durante el uso y Git lo ignora.

| ID | Nombre | Correo | Sede | Rol | Estado |
|----|--------|--------|------|-----|--------|
| USR-0001 | Administrador TI | `admin@maizap.com` | A | administrador | activo |
| USR-0002 | Ana Torres | `ana.torres@maizap.com` | A | usuario | activo |
| USR-0003 | Luis Paredes | `luis.paredes@maizap.com` | B | usuario | activo |
| USR-0004 | Marta Ruiz | `marta.ruiz@maizap.com` | B | usuario | **desactivado** (sirve para probar el error 403) |

También incluye dos mensajes (`MSG-000001` y `MSG-000002`) entre Ana y Luis, con copia al administrador, para que las bandejas no empiecen vacías.

Para **reiniciar los datos** a su estado inicial, detenga el servidor y borre `data/maizap.json`. Puede hacerlo desde el explorador de archivos o con:

```bash
rm data/maizap.json
```

Para ejecutar las pruebas automatizadas (modelos, servicios y un recorrido HTTP completo):

```bash
go test ./...
```

## Guion de demostración

Este recorrido cubre lo que se revisa en la entrega (inicio del servidor, conexión, operaciones principales y errores):

1. Terminal 1: `go run ./cmd/server`.
2. Terminal 2: `go run ./cmd/client`. Se muestra «Conectado a 127.0.0.1:8080».
3. Opción `9`: el cliente responde que la **opción no existe**.
4. `2` con `ana.torres@maizap.com`, luego `2) Bandeja de entrada`: aparece `MSG-000002` marcado como no leído.
5. `4` con `MSG-000002`: se muestra el contenido y el mensaje queda leído.
6. `1` con destinatarios `luis.paredes@maizap.com, fantasma@maizap.com, x@gmail.com`, asunto vacío y cuerpo vacío. El servidor responde con un **error de validación** que lista los cuatro problemas.
7. `1` con un envío válido a `luis.paredes@maizap.com`, luego `3) Mensajes enviados`.
8. `7` con `USR-0003`: la operación queda **no autorizada** (403), porque Ana no es administradora.
9. `5` con `MSG-000002`: el mensaje se elimina solo del buzón de Ana.
10. `8` para cerrar sesión, luego `2` con `luis.paredes@maizap.com`: Luis ve el mensaje nuevo de Ana.
11. Detener el servidor (`Ctrl+C`) y usar cualquier opción en el cliente: aparece el **error de conexión**.

## API del servidor

Todas las respuestas son JSON con el mismo formato:

```json
{ "ok": true, "data": { } }
{ "ok": false, "error": { "code": "validation", "message": "...", "details": ["..."] } }
```

Las rutas protegidas requieren el encabezado `Authorization: Bearer <token>`, con el token obtenido en `POST /api/v1/sessions`.

| Método | Ruta | Protegida | Descripción |
|--------|------|-----------|-------------|
| GET | `/health` | No | Estado del servidor |
| POST | `/api/v1/users` | No | Registrar usuario `{name, email, site, area}` |
| POST | `/api/v1/sessions` | No | Identificarse `{email}` y recibir un token |
| DELETE | `/api/v1/sessions/current` | Sí | Cerrar la sesión |
| GET | `/api/v1/me` | Sí | Perfil del usuario identificado |
| GET | `/api/v1/users` | Sí | Directorio de usuarios |
| POST | `/api/v1/users/{id}/deactivate` | Sí, solo administrador | Desactivar un usuario |
| POST | `/api/v1/messages` | Sí | Enviar `{to: [correos], subject, body}` |
| GET | `/api/v1/mailbox/inbox` | Sí | Bandeja de entrada |
| GET | `/api/v1/mailbox/sent` | Sí | Mensajes enviados |
| GET | `/api/v1/messages/{id}` | Sí | Leer un mensaje (lo marca como leído) |
| DELETE | `/api/v1/messages/{id}` | Sí | Eliminar un mensaje del buzón propio |

Ejemplo con `curl`, con el servidor en ejecución:

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/sessions -H "Content-Type: application/json" -d '{"email":"ana.torres@maizap.com"}'
```

## Estructura de carpetas

```text
.
├── app-go/
│   ├── cmd/
│   │   ├── client/main.go         Punto de entrada del cliente de consola
│   │   └── server/main.go         Punto de entrada del servidor HTTP
│   ├── internal/
│   │   ├── models/                Entidades: User, Message, Envelope, MailboxEntry, Session
│   │   ├── services/              Lógica de negocio: usuarios, correo e identificación
│   │   ├── handlers/              Rutas HTTP, autorización por token y respuestas JSON
│   │   ├── storage/               Persistencia en archivo JSON con escritura atómica
│   │   ├── protocol/              Contrato de mensajes compartido por cliente y servidor
│   │   ├── client/                Cliente HTTP y menús de consola
│   │   ├── config/                Opciones y variables de entorno (IP, puerto, etc.)
│   │   └── apperr/                Tipo de error de la aplicación con categorías
│   ├── data/
│   │   └── seed.json              Datos de prueba
│   ├── go.mod
│   └── README.md
├── docs/
│   └── documentacion-tecnica.md   Base de la documentación técnica (exportar a PDF)
├── .gitignore
├── LICENSE
└── README.md
```

## Limitaciones conocidas

- **Identificación sin contraseña**: el prototipo identifica al usuario solo por su correo. Es una medida temporal y deliberada mientras se integra el servicio FastAPI (semanas 11 y 12), que se encargará de las contraseñas con bcrypt y de los tokens JWT.
- **Sesiones en memoria**: los tokens se pierden al reiniciar el servidor y los usuarios deben identificarse de nuevo.
- **Almacenamiento en un archivo JSON**: es adecuado para la demostración, pero no para muchos usuarios concurrentes ni grandes volúmenes. Cada escritura reescribe el archivo completo, aunque lo hace de forma atómica (archivo temporal y luego reemplazo).
- **HTTP sin cifrar**: la comunicación no usa TLS. Se usará HTTPS cuando se despliegue entre sedes.
- **Sin archivos adjuntos, sin respuestas en hilo, sin búsqueda ni paginación** de mensajes.
- **Eliminación lógica**: el mensaje desaparece solo del buzón de quien lo elimina y no existe una papelera para recuperarlo.
- **Pruebas en un solo equipo o red local**: la integración entre las dos sedes (VLAN y OSPF) corresponde a fases posteriores.

## Autores y contribuciones

| Integrante | Contribuciones |
|------------|----------------|
| Emilio Cuenca | _Completar_ |
| Danna Simaluisa | _Completar_ |
| Camila Villagran | _Completar_ |

Cada integrante debe registrar su trabajo con commits propios para que el historial muestre su contribución.

## Créditos, licencia y declaración de uso de IA

**Licencia**: [MIT](LICENSE), © 2026 Emilio Cuenca, Danna Simaluisa y Camila Villagran. Se eligió MIT porque es una licencia breve y permisiva, adecuada para un proyecto académico que se puede reutilizar citando a sus autores.

**Créditos**: el proyecto usa solo la biblioteca estándar de Go (`net/http`, `encoding/json`, `crypto/rand`, `flag`, `sync`, entre otras). No tiene dependencias de terceros.

**Declaración de uso de herramientas de inteligencia artificial**: el equipo utilizó un asistente de IA generativa como herramienta de apoyo para proponer la estructura inicial del código, redactar borradores de documentación y sugerir casos de prueba. Los integrantes revisaron, ejecutaron y validaron todo el contenido generado y lo adaptaron a los requisitos del proyecto. Los autores del trabajo son los integrantes del equipo, que son responsables de su contenido y pueden explicarlo.
