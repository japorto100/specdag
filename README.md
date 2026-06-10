# specdag

A fast, offline CLI and Model Context Protocol (MCP) server for validating, assembling, and analyzing dependency maps in Event-Spec-Driven Development (ESDD).

## Features

- **Deterministic Guardian:** Validates declarative feature maps for cyclic graphs, node types, and edge constraints.
- **Global Assembly:** Recursively merges decentralized, feature-local dependency maps, detects naming conflicts, and validates system-wide relations.
- **Impact Analysis:** Traverses downstream paths via Depth-First Search (DFS) to determine all system components affected by modifying a specific node (e.g., an event or API contract).
- **Visualization:** Generates filterable Mermaid diagrams and beautiful, static HTML review reports.
- **MCP Server:** Exposes all analytical capabilities as Stdio tools for Cursor, Claude Desktop, and other agent runtimes.

---

## Example Diagram

GitHub natively renders Mermaid code blocks. Below is how a typical feature dependency map is visualized:

```mermaid
graph TD
    n_4130ca023c(["User can import research documents (intent)"])
    n_be4a54f2ae{{"Generated output contains only sourced claims (expectation)"}}
    n_27732a3ec4[/"document.uploaded (event)"/]
    n_282f1b4028["Extract text (job)"]
    n_db97ffea6f[("Extracted text (artifact)")]
    n_a12ba00100("Citation check (verifier)")

    n_4130ca023c -->|"defines_success_for"| n_be4a54f2ae
    n_27732a3ec4 -->|"triggers"| n_282f1b4028
    n_282f1b4028 -->|"produces"| n_db97ffea6f
    n_db97ffea6f -->|"verified_by"| n_a12ba00100
    n_a12ba00100 -->|"verifies"| n_be4a54f2ae
```

---

## Installation

### A) Via npm/npx (Recommended for Cursor / Claude Desktop)

You do not need to install specdag globally. You can run it directly:

```bash
npx @japorto100/specdag --help
```

Or install it globally on your system:

```bash
npm install -g @japorto100/specdag
specdag --help
```

*Note: The npm wrapper automatically downloads the appropriate precompiled native Go binary for your operating system (Linux, macOS, Windows) and CPU architecture (amd64, arm64) during installation.*

### B) Via Go

If you have Go installed on your system:

```bash
go install github.com/japorto100/specdag@latest
```

---

## MCP Server Configuration

Add specdag to your MCP configuration file (e.g., `mcpServerConfig.json` for Cursor or Claude Desktop):

### A) Via npx (No Go installation required):

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

### B) Via Go (If go install was used):

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

## CLI Commands

### 1. Validate locally
Checks a local feature map file for syntactic and topological validity (cycle checking):
```bash
specdag validate specs/features/012-agent-run/dependency-map.yaml
```

### 2. Assemble globally
Recursively searches a directory for all local maps, validates them, and merges them conflict-free:
```bash
specdag assemble specs/features/ -o specs/_generated/dependency-map.global.json
```

### 3. Render Mermaid
Generates Mermaid markdown diagram code from a map. Use `--view` to filter specific sub-views (`full`, `critical-path`, `approvals`, `events`, `verification`, `orphans`):
```bash
specdag render specs/features/012-agent-run/dependency-map.yaml --view critical-path
```

### 4. Output Summary
Calculates KPIs, approval gates, orphan nodes, and the critical path:
```bash
specdag summary specs/features/012-agent-run/dependency-map.yaml
```

### 5. Perform Impact Analysis
Finds all transitively affected downstream system components when a node ID changes:
```bash
specdag impact specs/features/012-agent-run/dependency-map.yaml event.document.uploaded
```

### 6. Generate HTML Report
Generates a beautiful static HTML review page (KPIs, tables, Mermaid diagrams, filters) for a feature map or an assembled directory:
```bash
specdag report specs/features/ -o specs/_generated/dependency-report.html
```

---

## License

MIT License.
