# specdag

A fast, offline CLI and Model Context Protocol (MCP) server for validating, assembling, and analyzing dependency maps in Event-Spec-Driven Development (ESDD).

## Features

- **deterministischer Wächter:** Validiert deklarative Feature-Maps auf Zyklen, Knotentypen und Edge-Constraints.
- **globales Assembly:** Führt dezentral gepflegte lokale dependency maps zusammen, erkennt Konflikte und verifiziert systemweite Beziehungen.
- **Impact-Analyse:** Ermittelt per Downstream-DFS alle nachgelagerten Systemkomponenten, die von einer Änderung eines bestimmten Knotens (z. B. eines Events oder Vertrags) betroffen wären.
- **Visualisierung:** Generiert filterbare Mermaid-Diagramme und statische HTML-Review-Berichte.
- **MCP-Server:** Exponiert CLI-Funktionalitäten als Stdio-Tools für Cursor, Claude Desktop und andere Agent-Runtimes.

---

## Installation

### A) Via npm/npx (Empfohlen für Cursor / Claude Desktop)

Du musst specdag nicht global installieren. Du kannst es direkt ausführen:

```bash
npx @japorto100/specdag --help
```

Oder installiere es global auf deinem System:

```bash
npm install -g @japorto100/specdag
specdag --help
```

*Hinweis: Der npm-Wrapper lädt während der Installation automatisch das passende, vorkompilierte native Go-Binary für dein Betriebssystem (Linux, macOS, Windows) und deine CPU-Architektur (amd64, arm64).*

### B) Via Go

Falls du Go auf deinem System installiert hast:

```bash
go install github.com/japorto100/specdag@latest
```

---

## MCP-Server Konfiguration

Trage specdag in deine MCP-Konfigurationsdatei (z. B. `mcpServerConfig.json` für Cursor oder Claude Desktop) ein:

### A) Für npx (keine Go-Installation erforderlich):

```json
{
  "mcpServers": {
    "specdag": {
      "command": "npx",
      "args": ["-y", "@japorto100/specdag", "mcp"]
    }
  }
}
```

### B) Für Go (falls go install verwendet wurde):

```json
{
  "mcpServers": {
    "specdag": {
      "command": "specdag",
      "args": ["mcp"]
    }
  }
}
```

---

## CLI-Befehle

### 1. Lokal validieren
Prüft eine lokale Feature-Map auf syntaktische und topologische Gültigkeit (Zyklenprüfung):
```bash
specdag validate specs/features/012-agent-run/dependency-map.yaml
```

### 2. Global assemblieren
Durchsucht ein Verzeichnis rekursiv nach allen lokalen Maps, validiert diese und führt sie konfliktfrei zusammen:
```bash
specdag assemble specs/features/ -o specs/_generated/dependency-map.global.json
```

### 3. Mermaid rendern
Erzeugt Mermaid-Markdown-Code aus einer Map. Mit `--view` lassen sich spezifische Teilausschnitte filtern (`full`, `critical-path`, `approvals`, `events`, `verification`, `orphans`):
```bash
specdag render specs/features/012-agent-run/dependency-map.yaml --view critical-path
```

### 4. Text-Zusammenfassung ausgeben
Berechnet KPIs, Freigabeschranken (Approvals), verwaiste Knoten und den kritischen Pfad:
```bash
specdag summary specs/features/012-agent-run/dependency-map.yaml
```

### 5. Impact-Analyse durchführen
Ermittelt alle transitiv betroffenen, nachgelagerten Systemkomponenten bei Änderung einer Node-ID:
```bash
specdag impact specs/features/012-agent-run/dependency-map.yaml event.document.uploaded
```

### 6. HTML-Report generieren
Generiert eine schöne, statische HTML-Review-Seite (KPIs, Tabellen, Mermaid-Diagramme, Filter) für ein Feature oder ein assembliertes Gesamtverzeichnis:
```bash
specdag report specs/features/ -o specs/_generated/dependency-report.html
```

---

## Lizenz

MIT License.
