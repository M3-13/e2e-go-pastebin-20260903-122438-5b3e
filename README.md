# Go-Pastebin-REST-API

Eine kleine Pastebin-REST-API in Go, die ausschließlich die Standardbibliothek
(`net/http`) nutzt. Pastes werden in einem thread-sicheren In-Memory-Store mit
Mutex gehalten und können optional nach einer konfigurierbaren Lebensdauer
ablaufen. Alle Antworten und Fehler sind sauberes JSON.

## Tech Stack

- **Sprache**: Go (1.22)
- **Framework**: nur `net/http` (Standardbibliothek)
- **Tests**: `go test` mit `net/http/httptest`
- **Storage**: In-Memory (map + `sync.RWMutex`)

## Installation

Voraussetzung: Go 1.22 oder neuer.

## Starten

```sh
go run .
```

Der Server lauscht standardmäßig auf Port **8080**. Über die Umgebungsvariable
`PORT` kann ein anderer Port gesetzt werden:

```sh
PORT=9090 go run .
```

## Endpunkte

Alle Antworten sind JSON; Fehler haben immer das Format `{"error":"<text>"}`.

| Methode | Pfad           | Beschreibung                                         |
|---------|----------------|------------------------------------------------------|
| GET     | `/healthz`     | Health-Check, liefert `200 {"status":"ok"}`          |
| POST    | `/pastes`      | Paste anlegen (`201 {"id":"..."}`)                   |
| GET     | `/pastes`      | Alle Pastes als Metadaten auflisten                  |
| GET     | `/pastes/{id}` | Einen Paste abrufen                                  |
| DELETE  | `/pastes/{id}` | Einen Paste löschen (`204`, leerer Body)             |

Fehlertexte: `400 "invalid request"`, `404 "paste not found"`,
`405 "method not allowed"`, `413 "request body too large"`,
`415 "unsupported content type"`, `500 "internal server error"`.

## Features

- Health-Check-Endpunkt
- Thread-sicherer In-Memory-Store
- JSON-API mit konsistentem Fehlerformat
- Konfigurierbarer Port über die Umgebungsvariable `PORT`
