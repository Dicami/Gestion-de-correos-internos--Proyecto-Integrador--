# app-go · Servidor y cliente MaiZap

Código fuente en Go del prototipo. Las instrucciones completas están en el [README principal](../README.md).

## Inicio rápido

Ejecute cada comando en una terminal distinta, desde esta carpeta.

Terminal 1:

```bash
go run ./cmd/server
```

Terminal 2:

```bash
go run ./cmd/client
```

Para ejecutar las pruebas:

```bash
go test ./...
```

## Organización por capas

```text
cmd/client  ──► internal/client ──HTTP/JSON──► internal/handlers ──► internal/services ──► internal/storage ──► data/maizap.json
                     │                              │                      │
                     └──────── internal/protocol ───┘                      └──► internal/models
```

| Paquete | Responsabilidad |
|---------|-----------------|
| `models` | Entidades con campos no exportados, constructores que validan (`NewUser`, `NewEnvelope`, `NewMessage`, `NewMailboxEntry`, `NewSession`) y métodos de acceso y modificación |
| `services` | Reglas de negocio: `UserService`, `MailService` y la interfaz `Authenticator` con su implementación temporal `LocalAuthenticator` |
| `handlers` | Traducción entre HTTP y los servicios, autorización con token Bearer, códigos de estado y formato de respuesta |
| `storage` | `JSONStore`: persistencia en archivo con `sync.RWMutex`, escritura atómica y reversión del cambio en memoria si falla la escritura |
| `protocol` | Estructuras JSON compartidas por cliente y servidor |
| `client` | `APIClient` (solicitudes HTTP y errores de conexión) y `Console` (menús e ingreso de datos) |
| `config` | Lectura y validación de IP, puerto, archivos y tiempos |
| `apperr` | Error de la aplicación con categoría (`validation`, `not_found`, `forbidden`, etc.) que se convierte en un código HTTP |
