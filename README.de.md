![GripMock](https://github.com/bavix/gripmock/assets/5111255/d1fc10ef-2149-4302-8e24-aef4fdfe043c)

[![Coverage Status](https://coveralls.io/repos/github/bavix/gripmock/badge.svg?branch=master)](https://coveralls.io/github/bavix/gripmock?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/bavix/gripmock/v3)](https://goreportcard.com/report/github.com/bavix/gripmock/v3)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# GripMock 🚀

**Sprachen:** [English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja-JP.md) | Deutsch | [Español](README.es.md)

> Hinweis: Diese Seite wurde maschinell übersetzt. Der Inhalt kann ungenau oder unvollständig sein. Bitte verwenden Sie das englische Original [`README.md`](README.md) als Referenz.

**Der schnellste und zuverlässigste gRPC-Mock-Server für Tests und Entwicklung.**

GripMock erstellt einen Mock-Server aus Ihren `.proto`-Dateien oder kompilierten `.pb`-Deskriptoren und macht gRPC-Tests einfach und effizient. Perfekt für End-to-End-Tests, Entwicklungsumgebungen und CI/CD-Pipelines.

![greeter](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-greeter.gif)

## ✨ Funktionen

- **Native Laufzeit** - Single-In-Process-Engine ohne gRPC-Codegenerierung zur Laufzeit
- **Deskriptorquellen** - API laden aus `.proto`, kompilierten `.pb`, BSR-Modulen oder gRPC-Reflection
- **Dynamisches `.pb`-Service-Laden** - Kompilierte Protobuf-Deskriptoren zur Laufzeit über API ohne Neustart laden
- **Hot-Stub-Management** - Stubs über API/UI ohne Serverneustart erstellen, aktualisieren und entfernen
- **Flexibles Matching** - `equals`, `contains`, `matches`, `glob`, Header, Priorität und Match-Limits
- **Array-Reihenfolge-Flexibilität** - Optionale Array-Reihenfol- Ignorierung zur Reduzierung fragiler Test-Assertions
- **Dynamische Templates** - Antworten aus Request-Payload, Headern und Stream-Kontext erstellen
- **Vollständige gRPC-Abdeckung** - Unary, Server-Streaming, Client-Streaming und bidirektionales Streaming
- **Fehler-, Detail- und Verzögerungssimulation** - Realistische gRPC-Statuscodes, Details (`Any`) und Antwortzeitsteuerung
- **TLS- und mTLS-Unterstützung** - Sichere gRPC/HTTP-Testumgebungen mit nativen TLS-Optionen
- **Fortgeschrittene Protobuf-Typunterstützung** - Unterstützung für Well-Known und Extended Protobuf-Typen (`google.protobuf.*`, `google.type.*`)
- **YAML/JSON + Schema** - Stubs in beiden Formaten mit JSON-Schema-IDE-Validierung
- **Plugin-Ökosystem** - Funktionen mit Go-Plugins erweitern, Builder-Image-Tags unterstützt
- **Eingebaute Faker-Templates** - Realistische Personen-/Kontakt-/Geo-/Netzwerkdaten direkt in Templates generieren (`faker.*`)
- **OpenTelemetry-Tracing** - OTLP-Tracing für gRPC- und HTTP-Pfade (`otelgrpc` + `otelhttp`)
- **Prometheus-Metriken (`/metrics`)** - Laufzeit-/Prozessmetriken (`go_*`, `process_*`) plus GripMock-Metriken
- **Betriebs-APIs** - Health-Endpunkte, Descriptors-API, Stubs-API und Web-Dashboard
- **Embedded SDK (Experimentell)** - GripMock in Go-Tests mit Verifikationshilfen ausführen
- **MCP-API (Experimentell)** - Streamable MCP-Endpunkt für Agenten- und Tool-Integration
- **Upstream-Modi (Experimentell)** - `proxy`, `replay`, `capture`-Modi für schrittweise Migration von Live-Upstream-Diensten zu lokalen Mocks

## 📚 Dokumentation

**[Vollständige Dokumentation](https://bavix.github.io/gripmock)** - Komplette Anleitung mit Beispielen

- **Descriptor-API (`/api/descriptors`)** : Laden kompilierter Proto-Deskriptoren (`.pb`) zur Laufzeit mit validiertem curl-Workflow: [Dokumentation](https://bavix.github.io/gripmock/guide/api/descriptors)
- **Upstream-Modi (Experimentell)**: `proxy`, `replay`, `capture` mit praktischer Rollout-Anleitung: [Dokumentation](https://bavix.github.io/gripmock/guide/modes)
- **Embedded SDK (Experimentell)**: In-Process-Tests mit `sdk.NewServer`, `Match()`, `Return()` und Verifikation: [Dokumentation](https://bavix.github.io/gripmock/guide/embedded-sdk)
- **Faker-Referenz**: Eingebauter Faker Schlüssel-für-Schlüssel-Katalog mit Beispielen: [Dokumentation](https://bavix.github.io/gripmock/guide/stubs/faker)
- **OpenTelemetry + Metriken**: Tracing-Umgebungsvariablen und `/metrics`-Verhalten: [Dokumentation](https://bavix.github.io/gripmock/guide/introduction/advanced-usage)
- **GitHub Actions (CI/CD)**: Offizielle Workflow-Action zum Herunterladen, Starten, Warten auf Bereitschaft und Stoppen von GripMock: [Dokumentation](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 🧬 Projektentwicklung

GripMock begann als Fork von [tokopedia/gripmock](https://github.com/tokopedia/gripmock) und entwickelte sich zu einem unabhängigen, vollständig neu geschriebenen Projekt.

Heute konzentriert sich GripMock auf praktische Test-Workflows:

- Native In-Process-Architektur (keine Codegenerierung zur Laufzeit)
- Flexible Deskriptorquellen und Laufzeitoperationen (Hot-Stubs + Descriptors-API)
- Produktionsnahe Testfunktionen (Streaming, Templates, Upstream-Modi, Plugins, SDK, MCP)

Für Architekturdaten und Benchmarks: [Leistungsvergleich](https://bavix.github.io/gripmock/guide/introduction/performance-comparison)

## 🖥️ Weboberfläche

![gripmock-ui](https://raw.githubusercontent.com/bavix/.github/master/svgs/gripmock-ui.gif)

Greifen Sie auf das Web-Dashboard unter `http://localhost:4771/` zu, um Ihre Stubs visuell zu verwalten.

## 🚀 Schnellstart

### Installation

Wählen Sie Ihre bevorzugte Installationsmethode:

#### Homebrew (Empfohlen)
```bash
brew tap gripmock/tap
brew install --cask gripmock
```

#### Shell-Skript
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

Für Plugin-Builds verwenden Sie das zugehörige Builder-Image:

```bash
docker pull bavix/gripmock:v3.7.1-builder
```

#### Go-Installation
```bash
go install github.com/bavix/gripmock/v3@latest
```

### Grundlegende Verwendung

**Starten mit einer `.proto`-Datei:**
```bash
gripmock service.proto
```

**Statische Stubs hinzufügen:**
```bash
gripmock --stub stubs/ service.proto
```

**API direkt aus Buf Schema Registry (BSR) laden:**
```bash
gripmock --stub third_party/bsr/eliza buf.build/connectrpc/eliza
```

**API aus Live-gRPC-Server-Reflection laden:**
```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
```

Mit Optionen:
```bash
gripmock grpc://localhost:50051?timeout=10s
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
gripmock grpc://localhost:50051?bearer=<token>
```

**Upstream-Modi über Reflection (Experimentell):**
```bash
# Reiner Reverse-Proxy durch GripMock
gripmock grpc+proxy://localhost:50051

# Lokale Stubs zuerst, dann Upstream-Fallback bei Matcher-Fehlschlag
gripmock grpc+replay://localhost:50051

# Replay + automatische Aufzeichnung von Upstream-Fehlschlägen als GripMock-Stubs
gripmock grpc+capture://localhost:50051
```

Für private BSR-Module:
```bash
BSR_BUF_TOKEN=<token> gripmock --stub stubs/ buf.build/acme/private-api
```

Für selbstgehostete BSR:
```bash
BSR_SELF_BASE_URL=https://bsr.company.local \
BSR_SELF_TOKEN=<token> \
gripmock --stub stubs/ bsr.company.local/team/payments
```

**Docker verwenden:**
```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v $(pwd)/stubs:/stubs \
  -v $(pwd)/proto:/proto \
  bavix/gripmock --stub=/stubs /proto/service.proto
```

- **Port 4770**: gRPC-Server
- **Port 4771**: Web-UI und REST-API

### Observability (v3.10.0)

```bash
OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
gripmock --stub stubs/ service.proto
```

- `GET /metrics` ist immer verfügbar
- Tracing-Export ist nur aktiviert, wenn `OTEL_ENABLED=true`

## 🤖 GitHub Actions (CI/CD)

Verwenden Sie die offizielle Action [`bavix/gripmock-action`](https://github.com/bavix/gripmock-action) zur Ausführung von GripMock in CI-Pipelines.

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

Was die Action tut:

- Lädt GripMock von GitHub Releases herunter (`latest` oder festgelegte `version`)
- Startet GripMock im Hintergrund und wartet auf Bereitschaft (`/api/health/readiness`)
- Gibt Adressen über Outputs frei (`grpc-addr`, `http-addr`) für Testschritte
- Stoppt GripMock automatisch im Post-Step

Weitere Beispiele und vollständige Inputs/Outputs: [GitHub Actions-Anleitung](https://bavix.github.io/gripmock/guide/ci-cd/github-actions)

## 📖 Beispiele

Sehen Sie sich unsere umfassenden Beispiele im [`examples`](https://github.com/bavix/gripmock/tree/master/examples)-Ordner an:

- **Streaming** - Server-, Client- und bidirektionales Streaming
- **Datei-Uploads** - Getaktete Datei-Uploads testen
- **Echtzeit-Chat** - Bidirektionale Kommunikation
- **Datenfeeds** - Kontinuierliches Daten-Streaming
- **Authentifizierung** - Header-basierte Authentifizierungstests
- **Leistung** - Hochdurchsatz-Szenarien

### Greeter: Dynamische Stub-Demo

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

Hinweise:
- Dynamische Templates nur in `output` platzieren (z.B. `data`, `headers`, `stream`)
- `input`-Matching statisch halten (kein `{{ ... }}` in `equals`/`contains`/`matches`)

```bash
# Server starten
go run main.go examples/projects/greeter/service.proto --stub examples/projects/greeter

# Über grpcurl aufrufen
grpcurl -plaintext -d '{"name":"Alex"}' localhost:4770 helloworld.Greeter/SayHello
```

Erwartete Antwort:

```json
{
  "message": "Hello, Alex!"
}
```

## 🔧 Stubbing

### Grundlegendes Stub-Beispiel

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

### Erweiterte Funktionen

**Prioritätssystem:**
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

**Streaming-Unterstützung:**
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

### Dynamische Templates

GripMock unterstützt dynamische Templates im `output`-Bereich mit Go's `text/template`-Syntax.

- Zugriff auf Request-Felder: `{{.Request.field}}`
- Zugriff auf Header: `{{.Headers.header_name}}`
- Client-Streaming-Kontext: `{{.Requests}}`, `{{len .Requests}}`, `{{(index .Requests 0).field}}`
- Bidirektionales Streaming: `{{.MessageIndex}}` gibt den aktuellen Nachrichtenindex (0-basiert)
- Mathe-Hilfen: `sum`, `avg`, `mul`, `min`, `max`, `add`, `sub`, `div`
- Dienstprogramme: `json`, `split`, `join`, `upper`, `lower`, `title`, `sprintf`, `int`, `int64`, `float`, `round`, `floor`, `ceil`
- Eingebauter Faker: `faker.Person.*`, `faker.Contact.*`, `faker.Geo.*`, `faker.Network.*`, `faker.Identity.*`

Wichtige Regeln:
- Keine dynamischen Templates in `input.equals`, `input.contains` oder `input.matches` verwenden (Matching muss statisch sein)
- Für Server-Streaming: Wenn sowohl `output.stream` als auch `output.error`/`output.code` gesetzt sind, werden zuerst Nachrichten gesendet, dann der Fehler zurückgegeben. Wenn `output.stream` leer ist, wird der Fehler sofort zurückgegeben

**Header-Matching:**
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

## 🔍 Eingabe-Matching

GripMock unterstützt vier leistungsstarke Matching-Strategien:

### 1. Exakte Übereinstimmung (`equals`)
```yaml
input:
  equals:
    name: "gripmock"
    age: 25
    active: true
```

### 2. Teilweise Übereinstimmung (`contains`)
```yaml
input:
  contains:
    name: "grip"
```

### 3. Regex-Übereinstimmung (`matches`)
```yaml
input:
  matches:
    email: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    phone: "^\\+?[1-9]\\d{1,14}$"
```

### 4. Glob-Übereinstimmung (`glob`)
```yaml
input:
  glob:
    filename: "*.txt"
    path: "/usr/local/*"
```

## 🛠️ API

### REST-API-Endpunkte

- `GET /api/stubs` - Alle Stubs auflisten
- `POST /api/descriptors` - Protobuf-Deskriptor-Set (`FileDescriptorSet`) zur Laufzeit laden
- `POST /api/stubs` - Neuen Stub hinzufügen
- `POST /api/stubs/search` - Passenden Stub finden
- `DELETE /api/stubs` - Alle Stubs löschen
- `GET /api/health/liveness` - Health Check
- `GET /api/health/readiness` - Bereitschaftsprüfung

### API-Verwendungsbeispiel

```bash
# Stub hinzufügen
curl -X POST http://localhost:4771/api/stubs \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "input": {"equals": {"name": "world"}},
    "output": {"data": {"message": "Hello World!"}}
  }'

# Nach passendem Stub suchen
curl -X POST http://localhost:4771/api/stubs/search \
  -H "Content-Type: application/json" \
  -d '{
    "service": "Greeter",
    "method": "SayHello",
    "data": {"name": "world"}
  }'
```

## 📋 JSON-Schema-Unterstützung

Fügen Sie Schema-Validierung zu Ihren Stub-Dateien für IDE-Unterstützung hinzu:

**JSON-Dateien:**
```json
{
  "$schema": "https://bavix.github.io/gripmock/schema/stub.json",
  "service": "MyService",
  "method": "MyMethod"
}
```

**YAML-Dateien:**
```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
service: MyService
method: MyMethod
```

## 🌐 BSR-Integration

GripMock unterstützt die vereinfachte Integration mit Buf Schema Registry:

### Konfiguration

```bash
# Öffentliche BSR (Standard)
BSR_BUF_BASE_URL=https://buf.build
BSR_BUF_TOKEN=<token>

# Selbstgehostete BSR
BSR_SELF_BASE_URL=https://bsr.company.local
BSR_SELF_TOKEN=<token>
```

### Verwendung

```bash
# Öffentliches Modul
gripmock buf.build/connectrpc/eliza

# Selbstgehostetes Modul
gripmock bsr.company.local/team/payments:main

# Mit Stubs
gripmock --stub stubs/ bsr.company.local/team/payments
```

### Routing

GripMock leitet Module automatisch weiter:
- `buf.build/owner/repo` → verwendet Buf-Profil
- `bsr.company.local/owner/repo` → verwendet Self-Profil

Details unter: [BSR-Dokumentation](https://bavix.github.io/gripmock/guide/sources/bsr)

## 🔎 gRPC-Reflection-Quelle

GripMock unterstützt das Laden von Deskriptoren aus gRPC-Reflection über Endpunkt-Schemata:

- `grpc://host:port` (unsicher)
- `grpcs://host:port` (TLS)

Unterstützte Abfrageparameter:

- `timeout` (Standard `5s`)
- `bearer` (Authorization-Token)
- `serverName` (TLS-SNI-Override)

Beispiele:

```bash
gripmock grpc://localhost:50051
gripmock grpcs://api.company.local:443
gripmock grpcs://10.0.0.5:8443?serverName=api.company.local
```

Vollständige Anleitung: [gRPC-Reflection-Quelle](https://bavix.github.io/gripmock/guide/sources/grpc-reflection)

## 🔁 Upstream-Modi (Experimentell)

⚠️ **EXPERIMENTELLE FUNKTION**: Upstream-Modi können ohne Vorankündigung geändert werden.

Upstream-Modi arbeiten auf Basis von Reflection-Quellen und definieren das Laufzeitverhalten:

- `proxy` - Reiner Reverse-Proxy
- `replay` - Lokal zuerst + Upstream-Fallback
- `capture` - Replay + automatische Stub-Aufzeichnung aus Upstream

Modusanleitungen:

- [Upstream-Modi Übersicht](https://bavix.github.io/gripmock/guide/modes)
- [Proxy-Modus](https://bavix.github.io/gripmock/guide/modes/proxy)
- [Replay-Modus](https://bavix.github.io/gripmock/guide/modes/replay)
- [Capture-Modus](https://bavix.github.io/gripmock/guide/modes/capture)

## 📊 Benchmark-Diagramme

![Bildgrößen-Benchmark](docs/public/bench/image-size.svg)
![Startbereitschafts-Benchmark](docs/public/bench/startup-ready.svg)
![Latenz-Perzentil-Benchmark](docs/public/bench/latency-percentiles.svg)
![Durchsatz-Benchmark](docs/public/bench/throughput-rps.svg)

## 🔗 Nützliche Ressourcen

- 📖 **[Dokumentation](https://bavix.github.io/gripmock)** - Vollständige Anleitungen und Beispiele
- 🧪 **[gRPC-Testen mit Testcontainers](https://medium.com/skyro-tech/testing-grpc-client-with-mock-server-and-testcontainers-f51cb8a6be9a)** - Artikel von [@AndrewIISM](https://github.com/AndrewIISM)
- 📋 **[JSON-Schema](https://bavix.github.io/gripmock/schema/stub.json)** - Stub-Validierungsschema
- 🔗 **[OpenAPI](https://bavix.github.io/gripmock-openapi/)** - REST-API-Dokumentation

## 🤝 Mitwirken

Wir begrüßen Beiträge! Bitte lesen Sie unseren [Leitfaden zum Mitwirken](CONTRIBUTING.md) (Deutsch: [CONTRIBUTING.de.md](CONTRIBUTING.de.md)).

## 📄 Lizenz

Dieses Projekt ist unter der **MIT-Lizenz** lizenziert. Siehe [LICENSE](LICENSE)-Datei für Details.

---

**Mit ❤️ von der GripMock-Community erstellt**
