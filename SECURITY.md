VERDICT: CHANGES_REQUESTED

## Sicherheitsbericht

### Bewertungsgrundlage
Es wurden keine externen Security-Scanner ausgeführt (`no applicable security scanners for this project type`). Die Bewertung basiert daher ausschließlich auf manueller Codeanalyse der vorliegenden Go-Quelldateien. Die Abwesenheit von Scanner-Ergebnissen ist kein Befund, hinterlässt aber eine Prüflücke bei automatisierten Checks.

### Positivbefunde
- **Secrets:** Es sind keine hartkodierten Schlüssel, Passwörter, Token oder URLs im Code sichtbar. Logausgaben (`main.go`) beschränken sich auf den Port und enthalten keine Paste-Inhalte.
- **Injection:** Keine SQL-, Command-, Path- oder Template-Injection erkennbar. Es werden keine Dateien, Datenbanken oder externe Befehle verwendet. `encoding/json` deserialisiert ausschließlich in eine bekannte Struktur (`createPasteRequest`), keine unsichere Typverarbeitung.
- **AuthN/AuthZ:** Für die öffentliche Pastebin-Funktion ist keine Authentifizierung vorgesehen; es gibt keine geschützten Verwaltungsendpunkte. Keine Session-/Token-Verwaltung, daher kein Angriffsvektor.
- **Datenschutz:** AC-12/13/14 sind erfüllt: Es werden keine Client-Metadaten (IP, User-Agent) erfasst, `GET /pastes` und Fehlerantworten enthalten kein `content`, gelöschte oder abgelaufene Pastes werden vollständig aus der Map entfernt.
- **Eingabevalidierung:** JSON-Größenlimit (1 MB) via `http.MaxBytesReader`, Content-Type-Prüfung über `mime.ParseMediaType`, Hex-Validierung für IDs, positive Ablaufzeitprüfung. Fehlerantworten für 5xx sind generisch ohne Stacktrace oder interne Pfade.
- **Zufalls-IDs:** `crypto/rand` mit 16 Bytes, hex-kodiert (32 Zeichen), kryptographisch sicher.

### Findings

#### 1. Mittel: Unbegrenzte Ressourcen für Pastes (Speicher-DoS)
**Betroffene Stelle:** `store.go` (Store-Map), `paste.go` (POST-Handler), `list.go` (globaler Store)

**Beschreibung:** Obwohl der Request-Body auf 1 MB begrenzt ist, gibt es keine Begrenzung für die Anzahl der Pastes oder den gesamten belegten Speicher. Jeder erfolgreiche POST erzeugt dauerhaft einen neuen Eintrag in der In-Memory-Map. Ein Angreifer kann ohne Authentifizierung beliebig viele 1‑MB-Pastes erstellen, bis der Arbeitsspeicher des Prozesses erschöpft ist (Ressourcen-Exhaustion / Denial of Service). Da der Dienst öffentlich und zustandslos ist, ist dies ein praktikabler Angriff.

**Konkreter Fix:**
- Einführung eines globalen Limits, z. B. maximale Anzahl aktiver Pastes (`maxPastes`) oder maximaler Gesamtspeicher (`maxTotalBytes`).
- Im Store `Create` prüfen und bei Überschreitung einen Fehler zurückgeben, den der Handler als `503 Service Unavailable` oder `507 Insufficient Storage` im JSON-Fehlerformat ausgibt.
- Optional zusätzlich Rate-Limiting pro Client-IP (z. B. Token Bucket im Handler) oder eine maximale Ablaufzeit erzwingen.
- Sicherstellen, dass abgelaufene Einträge zeitnah aus der Map entfernt werden (bisher nur lazy bei `Get`/`List`/`Delete`), um Speicher zu begrenzen.

#### 2. Mittel: HTTP-Server ohne Zeitlimits (Slowloris / Ressourcen-Exhaustion)
**Betroffene Stelle:** `main.go` (Zeile `http.ListenAndServe(":"+port, router)`)

**Beschreibung:** Der HTTP-Server wird ohne `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout` oder `MaxHeaderBytes` gestartet. Dadurch kann ein Angreifer langsame Verbindungen offen halten (Slowloris) oder Header mit sehr langer Lesezeit senden, wodurch Server-Ressourcen blockiert werden. Go's `net/http` setzt ohne explizite Timeouts keine ausreichenden Schutzmechanismen.

**Konkreter Fix:**
Statt des einfachen Aufrufs einen `http.Server` konfigurieren:
```go
srv := &http.Server{
    Addr:              ":" + port,
    Handler:           router,
    ReadTimeout:       5 * time.Second,
    ReadHeaderTimeout: 2 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20, // z. B. 1 MB
}
log.Printf("listening on :%s", port)
if err := srv.ListenAndServe(); err != nil {
    log.Fatal(err)
}
```
Die Timeouts müssen mit den legitimen Client-Anforderungen kompatibel sein; für eine Pastebin-API sind Werte von 5–15 s üblich.

#### 3. Niedrig: Exklusive Sperren bei Leseoperationen (Lock-Contention)
**Betroffene Stelle:** `store.go` (`Get`, `List`)

**Beschreibung:** `Get` und `List` verwenden `s.mu.Lock()` statt `s.mu.RLock()`, obwohl sie überwiegend lesende Operationen sind. Dadurch werden parallele GET-/List-Requests serialisiert. Ein Angreifer kann durch viele gleichzeitige GET-Anfragen die Sperre lange halten und die Antwortzeiten für alle Nutzer verschlechtern (Performance-DoS). Da `Get`/`List` bei abgelaufenen Einträgen auch schreiben müssen, ist die Umstellung nicht trivial, aber lösbar.

**Konkreter Fix:**
- `Get`: `RLock` halten, prüfen ob vorhanden und nicht abgelaufen; bei abgelaufenem Eintrag `RUnlock`, dann exklusiven `Lock` nehmen, den Eintrag erneut prüfen und löschen.
- `List`: `RLock` verwenden, Metadaten einsammeln; benötigte Löschungen separat unter exklusivem `Lock` durchführen (z. B. IDs abgelaufener Einträge sammeln und danach löschen).
- Alternativ einen separaten `sync.Mutex` nur für Bereinigungsoperationen verwenden, um Leser nicht zu blockieren.
- Vorsicht vor Deadlocks durch Lock-Reihenfolge.

#### 4. Niedrig: Fehlende Obergrenze für `expires_in_seconds` (Overflow)
**Betroffene Stelle:** `paste.go` (Zeile `expiresIn = time.Duration(*req.ExpiresInSeconds) * time.Second`)

**Beschreibung:** `req.ExpiresInSeconds` ist ein `*int64`. Bei sehr großen Werten (z. B. `math.MaxInt64`) führt die Multiplikation mit `time.Second` zu einem Integer-Overflow, sodass `expiresIn` negativ oder ein unerwarteter kleiner Wert wird. In beiden Fällen wird kein `ExpiresAt` gesetzt (`expiresIn > 0` ist false), der Paste bleibt also dauerhaft bestehen. Dies ist kein direkter Exploit (da `expires_in_seconds=0` ebenfalls dauerhaft bedeutet), aber ein Robustheits-/Validierungsproblem, das zu unerwartetem Verhalten führen kann.

**Konkreter Fix:**
- Vor der Umwandlung eine Obergrenze validieren, z. B. `if *req.ExpiresInSeconds > maxExpirySeconds { 400 }` (max z. B. 365*24*3600 = 31.536.000).
- Oder mit sicherem Check: `if *req.ExpiresInSeconds > math.MaxInt64/int64(time.Second) { ... }`.
- Alternativ `time.Duration(*req.ExpiresInSeconds)` nach Sekunden begrenzt berechnen, bevor multipliziert wird.

#### 5. Niedrig: Fehlende optionale Härtungs-Header
**Betroffene Stelle:** `json.go` (`writeJSON`)

**Beschreibung:** Antworten setzen korrekt `Content-Type: application/json`, setzen aber keine zusätzlichen Header wie `X-Content-Type-Options: nosniff`. Da alle Endpunkte ausschließlich JSON ausliefern und der Content-Type gesetzt ist, ist das Risiko gering, aber der Header ist eine kostengünstige Härtung gegen MIME-Sniffing in Browsern.

**Konkreter Fix:**
In `writeJSON` vor dem `WriteHeader` ergänzen:
```go
w.Header().Set("X-Content-Type-Options", "nosniff")
```
Für eine reine JSON-API sind weitere Header wie `Content-Security-Policy` oder `Cache-Control` abhängig von Anforderungen optional.

### Abhängigkeiten
Laut Projektbeschreibung werden ausschließlich Go-Standardbibliotheken (`net/http`, `sync`, `time`, `crypto/rand`, `encoding/json`, `mime`, `os`, `log`) verwendet. Die `go.mod` ist nicht vollständig einsehbar; sofern keine externen Module importiert werden, bestehen keine bekannten Drittanbieter-Schwachstellen. Die eingesetzten Go-Standardbibliotheken gelten als sicher, sofern eine gepflegte Go-Version verwendet wird.

### Zusammenfassung
Das Produkt erfüllt die sicherheitsrelevanten Anforderungen (AC-08 bis AC-14) weitgehend. Es wurden keine kritischen oder hohen Schwachstellen festgestellt. Die identifizierten Punkte betreffen Ressourcenbegrenzung, Server-Härtung und Lock-Optimierung und rechtfertigen ein `CHANGES_REQUESTED`. Nach Umsetzung der mittleren Findings (Speicherlimit und Server-Timeouts) ist das Produkt aus Sicherheitssicht einsatzbereit.