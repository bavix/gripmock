# Zu GripMock beitragen

**Sprachen:** [English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md) | [日本語](CONTRIBUTING.ja-JP.md) | Deutsch | [Español](CONTRIBUTING.es.md)

> Hinweis: Diese Seite wurde maschinell übersetzt. Der Inhalt kann ungenau oder unvollständig sein. Bitte verwenden Sie das englische Original [`CONTRIBUTING.md`](CONTRIBUTING.md) als Referenz.

Vielen Dank für Ihr Interesse an einem Beitrag zu GripMock! Dieses Dokument enthält Richtlinien für die Mitarbeit am Projekt.

## Erste Schritte

1. **Repository forken** und den Fork lokal klonen
2. **Entwicklungsumgebung einrichten**:
   - [grpctestify](https://github.com/gripmock/grpctestify-rust) für Integrationstests installieren (siehe [grpctestify-Dokumentation](https://gripmock.github.io/grpctestify-rust/) für Installationsanweisungen)
   - Stellen Sie sicher, dass Go installiert und konfiguriert ist

### ConnectRPC-Tests

**HTTP-Client-Tests** (`.http`-Dateien) für den ConnectRPC-Server befinden sich in `examples/projects/*/connectrpc-tests.http`.

**Ausführung mit httpyac:**
```bash
npx httpyac run examples/projects/greeter/connectrpc-tests/ --all
```

Sie können `.http`-Dateien auch in JetBrains-IDEs (GoLand, IntelliJ) öffnen und auf das Ausführungssymbol neben jeder Anfrage klicken.

## Testanforderungen

### ⚠️ Kritische Regeln

#### 1. gRPC-Serveränderungen erfordern Integrationstests

**Wenn Sie etwas an der gRPC-Serverfunktionalität ändern, hinzufügen oder reparieren, MÜSSEN Sie Integrationstests mit grpctestify im `.gctf`-Format schreiben.**

Integrationstests befinden sich im `examples/`-Verzeichnis. Beispiel einer `.gctf`-Datei:

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**Tests ausführen:**
```bash
make test              # Unit-Tests
grpctestify examples/  # Integrationstests
make lint              # Linter
```

**Wo Tests platziert werden:**
- Integrationstests: `examples/projects/*/case_*.gctf`
- Unit-Tests: `internal/app/*_internal_test.go`

#### 2. Jeder PR muss Tests enthalten

Alle Pull-Requests müssen geeignete Tests enthalten, insbesondere für Fehlerbehebungen und neue Funktionen.

#### 3. Tests lokal ausführen

Stellen Sie vor dem Einreichen eines PRs sicher, dass alle Tests bestanden werden:

**Für Integrationstests mit grpctestify:**
```bash
# Server starten (in einem separaten Terminal)
go run main.go examples -s examples

# Integrationstests ausführen
grpctestify examples/
```

**Für Unit-Tests:**
```bash
make test
make lint
```

## Rückwärtskompatibilität

**Alle Änderungen MÜSSEN rückwärtskompatibel sein**, sofern nicht ausdrücklich durch ein Issue diskutiert und genehmigt.

### Prozess für Breaking Changes

Wenn Sie einen Breaking Change einführen müssen:

1. **Zuerst ein Issue erstellen**: Öffnen Sie ein Issue mit einem detaillierten Vorschlag, der Folgendes enthält:
   - Beschreibung des zu lösenden Problems
   - Warum der Breaking Change notwendig ist
   - Vorgeschlagener Migrationspfad für bestehende Benutzer

2. **Auf Genehmigung warten**: Implementieren Sie keine Breaking Changes ohne Diskussion und Genehmigung durch die Maintainer

3. **Migrationsanleitung bereitstellen**: Fügen Sie bei Genehmigung klare Migrationsanweisungen in Ihren PR ein

## Pull-Request-Prozess

### Vor dem Einreichen

- [ ] Alle Tests bestehen lokal
- [ ] Code folgt den Projektstilrichtlinien (`make lint`)
- [ ] Dokumentation wurde bei Bedarf aktualisiert
- [ ] Ihr Branch ist mit dem Hauptbranch auf dem neuesten Stand

### PR-Beschreibung

Wenn Sie einen PR erstellen, fügen Sie bitte Folgendes hinzu:
- Beschreibung der Änderungen
- Art der Änderung (Fehlerbehebung, neue Funktion usw.)
- Testinformationen (Unit-Tests, Integrationstests bei gRPC-Serveränderungen)
- Rückwärtskompatibilitätsstatus
- Verwandte Issues

## Code-Stil

- Standard Go-Formatierung befolgen: `gofmt` und `goimports`
- Linter ausführen: `make lint`
- Sinnvolle Variablen- und Funktionsnamen verwenden
- Kommentare für exportierte Funktionen und Typen hinzufügen
- Neuen Code in geeigneten Paketen unter `internal/` platzieren

## Dokumentation

Dokumentation aktualisieren, wenn:
- Neue Funktionen hinzugefügt werden
- Bestehendes Verhalten geändert wird
- Fehler behoben werden, die Benutzer-Workflows betreffen

Dokumentationsorte:
- Benutzerdokumentation: `docs/guide/`
- Beispiele: `examples/`-Verzeichnis
- Haupt-README: `README.md`

## Fragen?

- Vorhandene Issues und Diskussionen überprüfen
- Ein neues Issue mit dem Label `question` öffnen
- Die [Dokumentation](https://bavix.github.io/gripmock/) durchsehen

## Zusätzliche Ressourcen

- [Projektdokumentation](https://bavix.github.io/gripmock/)
- [grpctestify-Dokumentation](https://gripmock.github.io/grpctestify-rust/)

Vielen Dank, dass Sie zu GripMock beitragen! 🚀
