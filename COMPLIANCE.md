VERDICT: CHANGES_REQUESTED

## 1. DSGVO

### D1 — Unbegrenzte Speicherdauer bei Pastes ohne `expires_in_seconds`
**Schweregrad:** hoch  
**Dateien:** `store.go`, `paste.go`  
**Befund:**  
Ein Paste ohne gültiges `expires_in_seconds` wird mit `ExpiresAt == nil` dauerhaft im In-Memory-Store gehalten. Es gibt weder eine Standardablaufzeit noch eine Obergrenze. Sofern Paste-Inhalte personenbezogene Daten enthalten, fehlt damit eine wirksame Speicherbegrenzung im Sinne von Art. 5 Abs. 1 lit. e DSGVO. Die optionale Ablauffrist ist vorhanden, aber nicht verpflichtend und nicht begrenzt.

**Konkrete Abhilfe:**  
In `CreatePasteHandler` bzw. `Store.Create` einen dokumentierten Standardablauf setzen, z. B.:

```go
const defaultPasteTTL = 24 * time.Hour
const maxPasteTTL = 30 * 24 * time.Hour

if expiresIn == 0 {
    expiresIn = defaultPasteTTL
}
if expiresIn > maxPasteTTL {
    writeError(w, http.StatusBadRequest, "invalid request")
    return
}
```

Zusätzlich in der README.md die Speicherdauer dokumentieren.

### D2 — Abgelaufene Pastes werden nur „lazy“ gelöscht
**Schweregrad:** mittel  
**Datei:** `store.go`  
**Befund:**  
Abgelaufene Einträge werden nur entfernt, wenn `Get`, `List` oder `Delete` auf sie zugreifen. Ein abgelaufener Paste bleibt sonst unbegrenzt im Arbeitsspeicher, obwohl die Speicherfrist abgelaufen ist. Das kann gegen die Grundsätze der Speicherbegrenzung und Datenminimierung (Art. 5 Abs. 1 lit. c, e DSGVO) verstoßen.

**Konkrete Abhilfe:**  
Einen periodischen Aufräum-Job ergänzen, z. B. in `NewStore`:

```go
func (s *Store) startJanitor(interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for range ticker.C {
            s.deleteExpired()
        }
    }()
}
```

Alternativ `time.AfterFunc` pro Eintrag verwenden. So werden abgelaufene Daten unabhängig von API-Zugriffen gelöscht.

### D3 — Keine dokumentierte Rechtsgrundlage und Datenschutzhinweise
**Schweregrad:** mittel  
**Datei:** `README.md`  
**Befund:**  
Der Dienst verarbeitet auf Veranlassung der aufrufenden Stelle übermittelte Inhalte, die personenbezogene Daten enthalten können. Eine Rechtsgrundlage nach Art. 6 Abs. 1 DSGVO und die Informationspflichten nach Art. 13 DSGVO sind im sichtbaren Projektzustand nicht dokumentiert. Da es sich um ein Backend ohne Endnutzer-UI handelt, können diese Hinweise nicht durch eine Web-Oberfläche erteilt werden; sie müssen dem Betreiber/Integrator zumindest als Dokumentation bereitstehen.

**Konkrete Abhilfe:**  
In `README.md` einen Abschnitt „Datenschutz / Privacy“ ergänzen mit:  
- Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Dienstleistung auf Anfrage)  
- verarbeitete Datenkategorien: freiwillig übermittelter Paste-Inhalt, Sprache, Zeitstempel, optionale Ablauffrist  
- Speicherdauer: Standard-/Maximal-TTL und sofortige Löschung mit `DELETE`  
- Betroffenenrechte: Löschung per `DELETE /pastes/{id}`, Hinweis auf Auskunft/Berichtigung an den Betreiber  
- ausdrücklicher Hinweis: keine IP-Adressen oder User-Agent-Header werden gespeichert.

### D4 — Fehlender `Cache-Control: no-store` für Paste-Inhalte
**Schweregrad:** mittel  
**Datei:** `paste.go`, `json.go`  
**Befund:**  
`GET /pastes/{id}` liefert den vollständigen, potenziell personenbezogenen oder vertraulichen Inhalt ohne `Cache-Control: no-store`. HTTP-Caches oder Proxys dürfen die Antwort damit zwischenspeichern. Das kann zur unkontrollierten Verbreitung von Inhalten führen und gefährdet die Vertraulichkeit.

**Konkrete Abhilfe:**  
In `GetPasteHandler` vor `writeJSON` setzen:

```go
w.Header().Set("Cache-Control", "no-store")
w.Header().Set("Pragma", "no-cache")
```

Alternativ zentral in `writeJSON` für den gesamten Dienst, sofern dies die übrigen Antworten nicht beeinträchtigt.

### D5 — Kein ausdrücklicher Ausschluss besonderer Datenkategorien
**Schweregrad:** niedrig  
**Datei:** `README.md`  
**Befund:**  
Die API nimmt beliebige Freitext-Inhalte an. Enthalten diese besondere Kategorien personenbezogener Daten nach Art. 9 DSGVO, wäre die Rechtsgrundlage des Betreibers regelmäßig nicht ohne weiteres gegeben.

**Konkrete Abhilfe:**  
In `README.md` eine Nutzungsbeschränkung dokumentieren, z. B.: „Es dürfen keine besonderen Kategorien personenbezogener Daten nach Art. 9 DSGVO eingestellt werden.“

---

## 2. EU Cyber Resilience Act (CRA)

### C1 — Mutierende Operationen ohne Authentifizierung/Autorisierung
**Schweregrad:** hoch  
**Dateien:** `main.go`, `delete.go`  
**Befund:**  
`DELETE /pastes/{id}` entfernt Pastes ohne jede Authentifizierung. Wer die ID kennt, kann fremde Inhalte löschen. Das beeinträchtigt Integrität und Verfügbarkeit und widerspricht dem Grundsatz „Security by design“ sowie dem Schutz vor unbefugtem Zugriff nach Anhang I CRA. Die zufällige 128-Bit-ID verringert das Risiko, kompensiert es aber nicht, sobald eine ID geteilt oder geleakt wurde.

**Konkrete Abhilfe:**  
Optionalen Admin-Schutz einführen, z. B. über eine Umgebungsvariable `PASTE_ADMIN_TOKEN`. In `DeletePasteHandler`:

```go
if token := os.Getenv("PASTE_ADMIN_TOKEN"); token != "" {
    if r.Header.Get("Authorization") != "Bearer "+token {
        writeError(w, http.StatusUnauthorized, "unauthorized")
        return
    }
}
```

So bleibt das Produkt ohne gesetztes Token abwärtskompatibel, kann aber sicher betrieben werden.

### C2 — Keine Gesamtbegrenzung der gespeicherten Pastes
**Schweregrad:** hoch  
**Dateien:** `store.go`, `paste.go`  
**Befund:**  
Die Anzahl der Pastes und der insgesamt belegte Speicher sind unbegrenzt. Ein Angreifer kann per `POST /pastes` den Prozessspeicher erschöpfen und die Verfügbarkeit des Dienstes beenden. Das ist ein CRA-relevanter Schwachpunkt hinsichtlich Verfügbarkeit und Ressourcenkontrolle.

**Konkrete Abhilfe:**  
In `Store` eine maximale Anzahl von Einträgen oder eine Gesamtgröße einführen:

```go
const maxPastes = 100_000
```

In `Create` bei Überschreitung einen Fehler zurückgeben; im Handler als `429 Too Many Requests` oder `503 Service Unavailable` im JSON-Fehlerformat antworten. Alternativ ein vorgelagertes Rate-Limit je Client.

### C3 — Keine dokumentierten Sicherheitseigenschaften, SBOM und Update-/Patch-Prozess
**Schweregrad:** mittel  
**Dateien:** `README.md`, `go.mod`, ggf. fehlendes `SECURITY.md`  
**Befund:**  
Der CRA verlangt für Produkte mit digitalen Elementen dokumentierte Sicherheitseigenschaften, eine SBOM (Software Bill of Materials) sowie einen Prozess für Sicherheitsupdates und Schwachstellenmeldungen. Im sichtbaren Stand existiert nur die Go-Standardbibliothek; es fehlen aber sichtbare Angaben zu Patch-Fähigkeit, Supportzeitraum und Meldestelle.

**Konkrete Abhilfe:**  
- `SECURITY.md` anlegen mit: Meldestelle für Schwachstellen, Reaktionszeit, Patch-/Release-Prozess, unterstützte Versionen.  
- In `README.md` dokumentieren, dass das Produkt ausschließlich die Go-Standardbibliothek nutzt und Updates über den regulären Go-/Betreiber-Depolymentprozess erfolgen.  
- Optional im CI eine SBOM erzeugen (z. B. `go list -m -json all` oder ein SBOM-Tool).

### C4 — `panic` bei Fehlschlag der Zufallsquelle kann den Prozess beenden
**Schweregrad:** niedrig  
**Datei:** `id.go`, `store.go`  
**Befund:**  
`NewID` ruft `panic(err)`, wenn `crypto/rand.Read` fehlschlägt. Ein solcher Fehler würde den gesamten Serverprozess beenden und die Verfügbarkeit beeinträchtigen.

**Konkrete Abhilfe:**  
`NewID` so umbauen, dass es einen Fehler zurückgibt:

```go
func NewID() (string, error) {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

`Store.Create` reicht den Fehler weiter; `CreatePasteHandler` antwortet im Fehlerfall mit `500` und generischer JSON-Fehlermeldung.

### C5 — Klartext-HTTP ohne dokumentierte TLS-Einordnung
**Schweregrad:** niedrig  
**Datei:** `main.go`, `README.md`  
**Befund:**  
Der Server lauscht unmittelbar auf HTTP. Das ist akzeptabel, wenn ein TLS-terminierender Reverse-Proxy vorgeschaltet ist; im sichtbaren Code fehlt jedoch jede Dokumentation dieser Betriebserwartung.

**Konkrete Abhilfe:**  
In `README.md` festhalten: „Der Dienst ist für den Betrieb hinter einem TLS-terminierenden Reverse-Proxy vorgesehen. Direktes HTTP nur in internen/Testumgebungen verwenden.“

---

## 3. EU AI Act

**Schweregrad:** entfällt  
**Befund:** Die Anwendung enthält keine KI-Funktion im Sinne des EU AI Act. Es sind keine Risikoklassen-, Transparenz- oder Kennzeichnungspflichten anwendbar.

---

## 4. Pflichttexte & Benutzeroberfläche

**Schweregrad:** entfällt  
**Befund:** Das Produkt ist ein reines Go-Backend ohne Endnutzer-Weboberfläche. Impressum, AGB, Cookie-Banner und Widerrufsbelehrung sind auf Produktebene nicht erforderlich. Datenschutzbezogene Dokumentationspflichten sind unter DSGVO (D3) abgehandelt.

---

## 5. Barrierefreiheit (WCAG/BITV/EAA)

**Schweregrad:** entfällt  
**Befund:** Keine öffentliche Web-UI vorhanden; es bestehen keine unmittelbaren Anforderungen an Barrierefreiheit.

---

## Zusammenfassung

Das Produkt erfüllt die wesentlichen technischen Sicherheitsanforderungen des Sprint-Specs: Begrenzung der Request-Größe, kryptografisch sichere IDs, generische Fehlerantworten, keine Erfassung von IP-Adressen oder User-Agent, keine Inhalte in Listen- oder Logausgaben. Es bestehen jedoch behebbare rechtliche Lücken, insbesondere die fehlende Speicherbegrenzung/Standardablaufzeit, die nur verzögerte Löschung abgelaufener Einträge, das Fehlen dokumentierter Datenschutzhinweise sowie CRA-relevante Sicherheitslücken bei Authentifizierung und Ressourcenbegrenzung. Die geforderten Änderungen sind umsetzbar, ohne die Kernfunktionen zu brechen.