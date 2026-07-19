![GripMock](https://github.com/bavix/gripmock/assets/5111255/d1fc10ef-2149-4302-8e24-aef4fdfe043c)

[![Coverage Status](https://coveralls.io/repos/github/bavix/gripmock/badge.svg?branch=master)](https://coveralls.io/github/bavix/gripmock?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/bavix/gripmock/v3)](https://goreportcard.com/report/github.com/bavix/gripmock/v3)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# GripMock 🚀

**Idiomas:** [English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja-JP.md) | [Deutsch](README.de.md) | Español

> Aviso: Esta página ha sido traducida automáticamente. Puede contener imprecisiones o estar incompleta. Consulte el original en inglés [`README.md`](README.md) como referencia.

**El servidor mock gRPC más rápido y confiable para pruebas y desarrollo.**

GripMock crea un servidor mock a partir de sus archivos `.proto` o descriptores `.pb` compilados, haciendo que las pruebas gRPC sean simples y eficientes. Perfecto para pruebas de extremo a extremo, entornos de desarrollo y pipelines CI/CD.

![greeter](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-greeter.gif)

## ✨ Características

- **Runtime nativo** - Motor de un solo proceso sin generación de código gRPC en tiempo de ejecución
- **Fuentes de descriptores** - Cargue API desde `.proto`, `.pb` compilados, módulos BSR o reflection gRPC
- **Carga dinámica de servicios `.pb`** - Cargue descriptores protobuf compilados en tiempo de ejecución vía API sin reinicios
- **Gestión de stubs en caliente** - Cree, actualice y elimine stubs vía API/UI sin reiniciar el servidor
- **Matching flexible** - `equals`, `contains`, `matches`, `glob`, cabeceras, prioridad y límites de coincidencia
- **Flexibilidad en orden de arrays** - Ignore opcionalmente el orden de arrays para reducir aserciones frágiles
- **Plantillas dinámicas** - Construya respuestas desde el payload de la solicitud, cabeceras y contexto del stream
- **Cobertura gRPC completa** - Unary, server streaming, client streaming y streaming bidireccional
- **Simulación de errores, detalles y retardos** - Devuelva códigos de estado gRPC realistas, detalles (`Any`) y temporización de respuestas
- **Soporte TLS y mTLS** - Ejecute entornos de prueba gRPC/HTTP seguros con opciones TLS nativas
- **Soporte avanzado de tipos Protobuf** - Maneje tipos well-known y extended protobuf (`google.protobuf.*`, `google.type.*`)
- **YAML/JSON + Schema** - Cree stubs en ambos formatos con validación JSON Schema en IDE
- **Ecosistema de plugins** - Extienda funciones con plugins Go y etiquetas de imágenes builder
- **Plantillas Faker incorporadas** - Genere datos ficticios realistas de persona/contacto/geo/red directamente en plantillas (`faker.*`)
- **Tracing OpenTelemetry** - Tracing OTLP para rutas gRPC y HTTP (`otelgrpc` + `otelhttp`)
- **Métricas Prometheus (`/metrics`)** - Métricas de runtime/proceso (`go_*`, `process_*`) más métricas de GripMock
- **APIs operativas** - Endpoints de salud, API de descriptores, API de stubs y panel web
- **Embedded SDK (Experimental)** - Ejecute GripMock dentro de pruebas Go con ayudantes de verificación
- **API MCP (Experimental)** - Endpoint MCP transmisible para integración con agentes y herramientas
- **Modos Upstream (Experimental)** - Modos `proxy`, `replay`, `capture` para migración gradual desde servicios upstream reales a mocks locales

## 📚 Documentación

**[Documentación completa](https://bavix.github.io/gripmock)** - Guía completa con ejemplos

- **API de descriptores (`/api/descriptors`)**: Carga en tiempo de ejecución de descriptores proto compilados (`.pb`) con flujo de trabajo curl validado: [docs](https://bavix.github.io/gripmock/guide/api/descriptors)
- **Modos Upstream (Experimental)**: `proxy`, `replay`, `capture` con guía práctica de implementación: [docs](https://bavix.github.io/gripmock/guide/modes)
- **Embedded SDK (Experimental)**: Pruebas en proceso con `sdk.NewServer`, `Match()`, `Return()` y verificación: [docs](https://bavix.github.io/gripmock/guide/embedded-sdk)
- **Referencia Faker**: Catálogo clave por clave del faker incorporado con ejemplos: [docs](https://bavix.github.io/gripmock/guide/stubs/faker)
- **OpenTelemetry + Métricas**: Variables de entorno de tracing y comportamiento de `/metrics`: [docs](https://bavix.github.io/gripmock/guide/introduction/advanced-usage)
- **GitHub Actions (CI/CD)**: Acción oficial de workflow para descargar, iniciar, esperar disponibilidad y detener GripMock automáticamente: [docs](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 🧬 Evolución del proyecto

GripMock comenzó como un fork de [tokopedia/gripmock](https://github.com/tokopedia/gripmock) y luego evolucionó hasta convertirse en un proyecto independiente completamente reescrito.

Hoy GripMock es un runtime independiente centrado en flujos de trabajo prácticos de prueba:

- Arquitectura nativa en proceso (sin generación de código en tiempo de ejecución)
- Fuentes de descriptores flexibles y operaciones en tiempo de ejecución (stubs en caliente + API de descriptores)
- Funciones de prueba de estilo productivo (streaming, plantillas, modos upstream, plugins, SDK, MCP)

Para detalles de arquitectura y metodología de benchmarks: [Comparación de rendimiento](https://bavix.github.io/gripmock/guide/introduction/performance-comparison)

## 🖥️ Interfaz web

![gripmock-ui](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-ui.gif)

Acceda al panel web en `http://localhost:4771/` para gestionar sus stubs visualmente.

## 🚀 Inicio rápido

### Instalación

Elija su método de instalación preferido:

#### Homebrew (Recomendado)
```bash
brew tap gripmock/tap
brew install --cask gripmock
```

#### Script Shell
```bash
curl -s https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.sh | sh -s
```

#### PowerShell (Windows)
```powershell
irm https://raw.githubusercontent.com/bavix/gripmock/refs/heads/master/setup.ps1 | iex
```

#### Docker
```bash
docker pull bavix/gripmock
```

Para builds de plugins, use la imagen builder asociada:

```bash
docker pull bavix/gripmock:v3.17.2-builder
```

#### Instalación Go
```bash
go install github.com/bavix/gripmock/v3@latest
```

### Uso básico

**Iniciar con un archivo `.proto`:**
```bash
gripmock service.proto
```

**Añadir stubs estáticos:**
```bash
gripmock --stub stubs/ service.proto
```

**Cargar API directamente desde Buf Schema Registry (BSR):**
```bash
gripmock --stub third_party/bsr/eliza buf.build/connectrpc/eliza
```

**Cargar API desde reflection de un servidor gRPC en vivo:**
```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
```

Con opciones:
```bash
gripmock grpc://localhost:50051?timeout=10s
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
gripmock grpc://localhost:50051?bearer=<token>
```

**Usar modos upstream sobre reflection (Experimental):**
```bash
# Proxy inverso puro a través de GripMock
gripmock grpc+proxy://localhost:50051

# Stubs locales primero, luego fallback upstream si no hay match
gripmock grpc+replay://localhost:50051

# Replay + grabación automática de fallos upstream como stubs de GripMock
gripmock grpc+capture://localhost:50051
```

Para módulos BSR privados:
```bash
BSR_BUF_TOKEN=<token> gripmock --stub stubs/ buf.build/acme/private-api
```

Para BSR auto-alojado:
```bash
BSR_SELF_BASE_URL=https://bsr.company.local \
BSR_SELF_TOKEN=<token> \
gripmock --stub stubs/ bsr.company.local/team/payments
```

**Usando Docker:**
```bash
docker run -p 4770:4770 -p 4771:4771 -p 4769:4769 \
  -v $(pwd)/stubs:/stubs \
  -v $(pwd)/proto:/proto \
  bavix/gripmock --stub=/stubs /proto/service.proto
```

- **Puerto 4770**: Servidor gRPC
- **Puerto 4771**: Interfaz web y API REST
- **Puerto 4769**: Puerta de enlace (gRPC-web / ConnectRPC)

### Observabilidad (v3.10.0)

```bash
OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
gripmock --stub stubs/ service.proto
```

- `GET /metrics` está siempre disponible
- La exportación de tracing solo está habilitada cuando `OTEL_ENABLED=true`

## 🤖 GitHub Actions (CI/CD)

Use la acción oficial [`bavix/gripmock-action`](https://github.com/bavix/gripmock-action) para ejecutar GripMock en pipelines CI.

```yaml
name: test

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      - name: Start GripMock
        uses: bavix/gripmock-action@v1
        with:
          source: proto/service.proto
          stub: stubs

      - name: Run tests
        run: go test ./...
```

Lo que hace la acción:

- Descarga GripMock desde GitHub Releases (`latest` o `version` fija)
- Inicia GripMock en segundo plano y espera disponibilidad (`/api/health/readiness`)
- Expone direcciones via outputs (`grpc-addr`, `http-addr`) para pasos de prueba
- Detiene GripMock automáticamente en el paso post

Más ejemplos y entradas/salidas completas: [Guía de GitHub Actions](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 📖 Ejemplos

Consulte nuestros ejemplos completos en la carpeta [`examples`](https://github.com/bavix/gripmock/tree/master/examples):

- **Streaming** - Streaming de servidor, cliente y bidireccional
- **Subidas de archivos** - Pruebe subidas de archivos fragmentadas
- **Chat en tiempo real** - Comunicación bidireccional
- **Feeds de datos** - Streaming continuo de datos
- **Autenticación** - Pruebas de autenticación basada en cabeceras
- **Rendimiento** - Escenarios de alto rendimiento

### Greeter: demo de stub dinámico

Stub (universal):

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
# examples/projects/greeter/stub_say_hello.yaml
- service: helloworld.Greeter
  method: SayHello
  input:
    matches:
      name: ".+"
  output:
    data:
      message: "Hello, {{.Request.name}}!"
```

Notas:
- Coloque las plantillas dinámicas solo en `output` (ej.: `data`, `headers`, `stream`)
- Mantenga el matching de `input` estático (sin `{{ ... }}` en `equals`/`contains`/`matches`)

```bash
# Iniciar servidor
go run main.go examples/projects/greeter/service.proto --stub examples/projects/greeter

# Llamar via grpcurl
grpcurl -plaintext -d '{"name":"Alex"}' localhost:4770 helloworld.Greeter/SayHello
```

Respuesta esperada:

```json
{
  "message": "Hello, Alex!"
}
```

## 🔧 Stubbing

### Ejemplo básico de stub

```yaml
service: Greeter
method: SayHello
input:
  equals:
    name: "gripmock"
output:
  data:
    message: "Hello GripMock!"
```

### Características avanzadas

**Sistema de prioridad:**
```yaml
- service: UserService
  method: GetUser
  priority: 100
  input:
    equals:
      id: "admin"
  output:
    data:
      role: "administrator"

- service: UserService
  method: GetUser
  priority: 1
  input:
    contains:
      id: "user"
  output:
    data:
      role: "user"
```

**Soporte de streaming:**
```yaml
service: TrackService
method: StreamData
input:
  equals:
    sensor_id: "GPS001"
output:
  stream:
    - position: {"lat": 40.7128, "lng": -74.0060}
      timestamp: "2024-01-01T12:00:00Z"
    - position: {"lat": 40.7130, "lng": -74.0062}
      timestamp: "2024-01-01T12:00:05Z"
```

### Plantillas dinámicas

GripMock admite plantillas dinámicas en la sección `output` usando la sintaxis `text/template` de Go.

- Acceda a campos de solicitud: `{{.Request.field}}`
- Acceda a cabeceras: `{{.Headers.header_name}}`
- Contexto de streaming de cliente: `{{.Requests}}`, `{{len .Requests}}`, `{{(index .Requests 0).field}}`
- Streaming bidireccional: `{{.MessageIndex}}` da el índice de mensaje actual (basado en 0)
- Ayudantes matemáticos: `sum`, `avg`, `mul`, `min`, `max`, `add`, `sub`, `div`
- Utilidades: `json`, `split`, `join`, `upper`, `lower`, `title`, `sprintf`, `int`, `int64`, `float`, `round`, `floor`, `ceil`
- Faker incorporado: `faker.Person.*`, `faker.Contact.*`, `faker.Geo.*`, `faker.Network.*`, `faker.Identity.*`

Reglas importantes:
- No use plantillas dinámicas dentro de `input.equals`, `input.contains` o `input.matches` (el matching debe ser estático)
- Para server streaming, si tanto `output.stream` como `output.error`/`output.code` están configurados, los mensajes se envían primero y luego se devuelve el error. Si `output.stream` está vacío, el error se devuelve inmediatamente

**Matching de cabeceras:**
```yaml
service: AuthService
method: ValidateToken
headers:
  equals:
    authorization: "Bearer valid-token"
input:
  equals:
    token: "abc123"
output:
  data:
    valid: true
    user_id: "user123"
```

## 🔍 Matching de entrada

GripMock admite cuatro potentes estrategias de matching:

### 1. Coincidencia exacta (`equals`)
```yaml
input:
  equals:
    name: "gripmock"
    age: 25
    active: true
```

### 2. Coincidencia parcial (`contains`)
```yaml
input:
  contains:
    name: "grip"
```

### 3. Coincidencia regex (`matches`)
```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
```

### 4. Coincidencia glob (`glob`)
```yaml
input:
  glob:
    filename: "*.txt"
    path: "/usr/local/*"
```

## 🛠️ API

### Endpoints de API REST

- `GET /api/stubs` - Listar todos los stubs
- `POST /api/descriptors` - Cargar set de descriptores protobuf (`FileDescriptorSet`) en tiempo de ejecución
- `POST /api/stubs` - Añadir nuevo stub
- `POST /api/stubs/search` - Buscar stub coincidente
- `DELETE /api/stubs` - Limpiar todos los stubs
- `GET /api/health/liveness` - Health check
- `GET /api/health/readiness` - Comprobación de disponibilidad

### Ejemplo de uso de API

```bash
# Añadir un stub
curl -X POST http://localhost:4771/api/stubs \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "input": {"equals": {"name": "world"}},
    "output": {"data": {"message": "Hello World!"}}
  }'

# Buscar stub coincidente
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {"name": "world"}
  }'
```

## 📋 Soporte JSON Schema

Añada validación de esquema a sus archivos stub para soporte IDE:

**Archivos JSON:**
```json
{
  "$schema": "https://bavix.github.io/gripmock/schema/stub.json",
  "service": "MyService",
  "method": "MyMethod"
}
```

**Archivos YAML:**
```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
service: MyService
method: MyMethod
```

## 🌐 Integración BSR

GripMock admite integración simplificada con Buf Schema Registry:

### Configuración

```bash
# BSR pública (por defecto)
BSR_BUF_BASE_URL=https://buf.build
BSR_BUF_TOKEN=<token>

# BSR auto-alojada
BSR_SELF_BASE_URL=https://bsr.company.local
BSR_SELF_TOKEN=<token>
```

### Uso

```bash
# Módulo público
gripmock buf.build/connectrpc/eliza

# Módulo auto-alojado
gripmock bsr.company.local/team/payments:main

# Con stubs
gripmock --stub stubs/ bsr.company.local/team/payments
```

### Enrutamiento

GripMock enruta automáticamente los módulos:
- `buf.build/owner/repo` → usa perfil Buf
- `bsr.company.local/owner/repo` → usa perfil Self

Para más detalles, consulte la [Documentación BSR](https://bavix.github.io/gripmock/guide/sources/bsr)

## 🔎 Fuente de reflection gRPC

GripMock admite la carga de descriptores desde reflection gRPC usando esquemas de endpoint:

- `grpc://host:port` (no seguro)
- `grpcs://host:port` (TLS)

Parámetros de consulta admitidos:

- `timeout` (por defecto `5s`)
- `bearer` (token de Authorization)
- `serverName` (anulación SNI TLS)

Ejemplos:

```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
```

Guía completa: [Fuente de reflection gRPC](https://bavix.github.io/gripmock/guide/sources/grpc-reflection)

## 🔁 Modos Upstream (Experimental)

⚠️ **CARACTERÍSTICA EXPERIMENTAL**: Los modos upstream pueden cambiar sin previo aviso.

Los modos upstream funcionan sobre fuentes de reflection y definen el comportamiento en tiempo de ejecución:

- `proxy` - Proxy inverso puro
- `replay` - Local primero + fallback upstream
- `capture` - Replay + grabación automática de stubs desde upstream

Guías de modos:

- [Visión general de modos upstream](https://bavix.github.io/gripmock/guide/modes)
- [Modo Proxy](https://bavix.github.io/gripmock/guide/modes/proxy)
- [Modo Replay](https://bavix.github.io/gripmock/guide/modes/replay)
- [Modo Capture](https://bavix.github.io/gripmock/guide/modes/capture)

## 📊 Gráficos de benchmark

![Benchmark de tamaño de imagen](docs/public/bench/image-size.svg)
![Benchmark de preparación de inicio](docs/public/bench/startup-ready.svg)
![Benchmark de percentiles de latencia](docs/public/bench/latency-percentiles.svg)
![Benchmark de rendimiento](docs/public/bench/throughput-rps.svg)

## 🔗 Recursos útiles

- 📖 **[Documentación](https://bavix.github.io/gripmock)** - Guías completas y ejemplos
- 🧪 **[Pruebas gRPC con Testcontainers](https://medium.com/skyro-tech/testing-grpc-client-with-mock-server-and-testcontainers-f51cb8a6be9a)** - Artículo por [@AndrewIISM](https://github.com/AndrewIISM)
- 📋 **[JSON Schema](https://bavix.github.io/gripmock/schema/stub.json)** - Esquema de validación de stubs
- 🔗 **[OpenAPI](https://bavix.github.io/gripmock-openapi/)** - Documentación de API REST

## 🤝 Contribuir

¡Aceptamos contribuciones! Consulte nuestra [Guía de contribución](CONTRIBUTING.md) (Español: [CONTRIBUTING.es.md](CONTRIBUTING.es.md)).

## 📄 Licencia

Este proyecto está licenciado bajo la **Licencia MIT** — consulte el archivo [LICENSE](LICENSE) para más detalles.

---

**Hecho con ❤️ por la comunidad de GripMock**
