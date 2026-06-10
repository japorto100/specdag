package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ReportApproval struct {
	ApprovalNodeID string
	TargetNodeID   string
	Condition      string
}

type ReportData struct {
	GraphID          string
	Kind             string
	Topology         string
	ValidationStatus string
	ValidationMsg    string
	NodesCount       int
	EdgesCount       int
	OrphansCount     int
	ApprovalsCount   int
	UnverifiedCount  int
	Orphans          []dag.Node
	Approvals        []ReportApproval
	Unverified       []dag.Node
	CriticalPath     []string
	Nodes            []dag.Node
	MermaidCode      string
}

// GenerateReportData aggregates data from a file or directory for the HTML report.
func GenerateReportData(targetPath string) (*ReportData, error) {
	var depMap dag.DependencyMap
	var validationMsg = "Map is valid and cycle-free"
	var validationStatus = "PASS"

	fi, err := os.Stat(targetPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access path %s: %w", targetPath, err)
	}

	if fi.IsDir() {
		// Assembling directory
		jsonStr, err := AssembleDirectory(targetPath)
		if err != nil {
			validationStatus = "FAIL"
			validationMsg = err.Error()
			// minimal skeleton setup
			depMap.Graph.ID = "global-system-map"
			depMap.Graph.Kind = "global_dependency"
			depMap.Graph.Topology = "dag"
			depMap.Graph.Status = "draft"
		} else {
			if err := json.Unmarshal([]byte(jsonStr), &depMap); err != nil {
				return nil, fmt.Errorf("failed to parse assembled json: %w", err)
			}
		}
	} else {
		// Single file
		if err := ValidateFile(targetPath); err != nil {
			validationStatus = "FAIL"
			validationMsg = err.Error()
		}
		data, err := os.ReadFile(targetPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %s: %w", targetPath, err)
		}
		if err := yaml.Unmarshal(data, &depMap); err != nil {
			return nil, fmt.Errorf("invalid YAML file: %w", err)
		}
	}

	if depMap.Graph.Topology == "" {
		depMap.Graph.Topology = "dag"
	}

	// 1. Orphans
	hasEdges := make(map[string]bool)
	for _, e := range depMap.Edges {
		hasEdges[e.From] = true
		hasEdges[e.To] = true
	}
	var orphans []dag.Node
	for _, n := range depMap.Nodes {
		if !hasEdges[n.ID] {
			orphans = append(orphans, n)
		}
	}

	// 2. Approvals
	var approvals []ReportApproval
	for _, e := range depMap.Edges {
		if e.Type == "requires_approval" {
			approvals = append(approvals, ReportApproval{
				ApprovalNodeID: e.To,
				TargetNodeID:   e.From,
				Condition:      e.Condition,
			})
		}
	}

	// 3. Unverified Expectations
	verifiedExpectations := make(map[string]bool)
	for _, e := range depMap.Edges {
		if e.Type == "verifies" {
			var fromType string
			for _, n := range depMap.Nodes {
				if n.ID == e.From {
					fromType = n.Type
					break
				}
			}
			if fromType == "verifier" || fromType == "event" {
				verifiedExpectations[e.To] = true
			}
		}
	}
	var unverified []dag.Node
	for _, n := range depMap.Nodes {
		if n.Type == "expectation" {
			if !verifiedExpectations[n.ID] {
				unverified = append(unverified, n)
			}
		}
	}

	// 4. Critical Path
	adj := make(map[string][]string)
	for _, e := range depMap.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	var paths []string
	visited := make(map[string]bool)
	var currentPath []string

	var dfs func(u string)
	dfs = func(u string) {
		currentPath = append(currentPath, u)
		visited[u] = true

		var isExpectation bool
		for _, n := range depMap.Nodes {
			if n.ID == u && n.Type == "expectation" {
				isExpectation = true
				break
			}
		}

		if isExpectation {
			paths = append(paths, strings.Join(currentPath, "\n→ "))
		} else {
			for _, v := range adj[u] {
				if !visited[v] {
					dfs(v)
				}
			}
		}

		currentPath = currentPath[:len(currentPath)-1]
		visited[u] = false
	}

	for _, n := range depMap.Nodes {
		if n.Type == "intent" {
			dfs(n.ID)
		}
	}

	// Mermaid Rendering
	mermaidCode := ""
	mCode, err := RenderMap(&depMap, "full")
	if err == nil {
		mCode = strings.TrimPrefix(mCode, "```mermaid\n")
		mCode = strings.TrimSuffix(mCode, "```")
		mermaidCode = mCode
	}

	return &ReportData{
		GraphID:          depMap.Graph.ID,
		Kind:             depMap.Graph.Kind,
		Topology:         depMap.Graph.Topology,
		ValidationStatus: validationStatus,
		ValidationMsg:    validationMsg,
		NodesCount:       len(depMap.Nodes),
		EdgesCount:       len(depMap.Edges),
		OrphansCount:     len(orphans),
		ApprovalsCount:   len(approvals),
		UnverifiedCount:  len(unverified),
		Orphans:          orphans,
		Approvals:        approvals,
		Unverified:       unverified,
		CriticalPath:     paths,
		Nodes:            depMap.Nodes,
		MermaidCode:      mermaidCode,
	}, nil
}

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>SpecDAG Dependency Report - {{.GraphID}}</title>
  <style>
    :root {
      --bg: #0b1020;
      --panel: #11172a;
      --panel2: #151d33;
      --text: #e8edf7;
      --muted: #aab6cc;
      --border: #26344f;
      --good: #69db7c;
      --warn: #ffd43b;
      --bad: #ff8787;
      --accent: #74c0fc;
      --code: #0a0f1d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.5;
    }
    header {
      padding: 32px;
      border-bottom: 1px solid var(--border);
      background: linear-gradient(180deg, #111a31 0%, #0b1020 100%);
    }
    h1 { margin: 0 0 8px; font-size: 28px; }
    h2 { margin: 0 0 16px; font-size: 20px; }
    h3 { margin: 0 0 10px; font-size: 16px; }
    p { color: var(--muted); }
    main {
      display: grid;
      grid-template-columns: minmax(320px, 460px) 1fr;
      gap: 20px;
      padding: 20px;
    }
    section, .card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 18px;
      margin-bottom: 20px;
    }
    .metrics {
      display: grid;
      grid-template-columns: repeat(4, minmax(120px, 1fr));
      gap: 12px;
      margin-top: 18px;
    }
    .metric {
      background: var(--panel2);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 12px;
    }
    .metric strong { display: block; font-size: 22px; }
    .metric span { color: var(--muted); font-size: 12px; }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
    }
    th, td {
      border-bottom: 1px solid var(--border);
      padding: 9px 8px;
      text-align: left;
      vertical-align: top;
    }
    th { color: var(--muted); font-weight: 600; }
    code, pre {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    }
    pre {
      background: var(--code);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px;
      overflow: auto;
      color: #dbe7ff;
      white-space: pre-wrap;
    }
    .badge {
      display: inline-block;
      border: 1px solid var(--border);
      border-radius: 999px;
      padding: 2px 8px;
      font-size: 12px;
      color: var(--muted);
    }
    .good { color: var(--good); }
    .warn { color: var(--warn); }
    .bad { color: var(--bad); }
    .toolbar {
      display: flex;
      gap: 10px;
      margin-bottom: 12px;
      flex-wrap: wrap;
    }
    input, select {
      background: var(--panel2);
      border: 1px solid var(--border);
      color: var(--text);
      border-radius: 10px;
      padding: 9px 10px;
    }
    .mermaid-box {
      background: white;
      color: #111;
      border-radius: 12px;
      padding: 16px;
      overflow: auto;
    }
    @media (max-width: 1100px) {
      main { grid-template-columns: 1fr; }
      .metrics { grid-template-columns: repeat(2, minmax(120px, 1fr)); }
    }
  </style>
</head>
<body>
  <header>
    <h1>SpecDAG Dependency Report</h1>
    <p>Generated by specdag. Do not edit manually. Source of truth: local feature maps, domains, events, contracts, and foundations.</p>
    <div class="metrics">
      <div class="metric"><strong>{{.GraphID}}</strong><span>Graph ID</span></div>
      <div class="metric"><strong>{{.Kind}}</strong><span>Kind</span></div>
      <div class="metric"><strong>{{.Topology}}</strong><span>Topology</span></div>
      <div class="metric"><strong class="{{if eq .ValidationStatus "PASS"}}good{{else}}bad{{end}}">{{.ValidationStatus}}</strong><span>Validation</span></div>
    </div>
  </header>

  <main>
    <aside>
      <section>
        <h2>Review summary</h2>
        <table>
          <tr><th>Check</th><th>Status</th></tr>
          <tr><td>Schema valid</td><td class="{{if eq .ValidationStatus "PASS"}}good{{else}}bad{{end}}">{{.ValidationStatus}}</td></tr>
          <tr><td>Cycle check</td><td class="{{if eq .ValidationStatus "PASS"}}good{{else}}bad{{end}}">{{.ValidationStatus}}</td></tr>
          <tr><td>Orphan nodes</td><td class="{{if gt .OrphansCount 0}}warn{{else}}good{{end}}">{{if gt .OrphansCount 0}}{{.OrphansCount}} warning{{if gt .OrphansCount 1}}s{{end}}{{else}}PASS{{end}}</td></tr>
          <tr><td>Approval gates</td><td class="good">{{.ApprovalsCount}} found</td></tr>
          <tr><td>Verification coverage</td><td class="{{if gt .UnverifiedCount 0}}warn{{else}}good{{end}}">{{if gt .UnverifiedCount 0}}{{.UnverifiedCount}} unverified{{else}}Covered{{end}}</td></tr>
        </table>
        {{if ne .ValidationStatus "PASS"}}
          <p class="bad" style="margin-top: 14px; font-weight: bold;">Validation Error:</p>
          <pre style="border-color: var(--bad); color: var(--bad);">{{.ValidationMsg}}</pre>
        {{end}}
      </section>

      <section>
        <h2>Critical path</h2>
        {{if .CriticalPath}}
          {{range .CriticalPath}}
            <pre>{{.}}</pre>
          {{end}}
        {{else}}
          <p>No causal path from Intent to Expectation found.</p>
        {{end}}
      </section>

      <section>
        <h2>Approval gates</h2>
        <table>
          <tr><th>Approval</th><th>Required by</th><th>Condition</th></tr>
          {{if .Approvals}}
            {{range .Approvals}}
              <tr>
                <td><code>{{.ApprovalNodeID}}</code></td>
                <td><code>{{.TargetNodeID}}</code></td>
                <td><code>{{if .Condition}}{{.Condition}}{{else}}none{{end}}</code></td>
              </tr>
            {{end}}
          {{else}}
            <tr><td colspan="3">No approval gates defined.</td></tr>
          {{end}}
        </table>
      </section>

      {{if .Orphans}}
      <section>
        <h2>Orphan Nodes</h2>
        <p>These nodes exist but have no edges connecting them to the dependency graph.</p>
        <table>
          <tr><th>ID</th><th>Type</th></tr>
          {{range .Orphans}}
            <tr>
              <td><code>{{.ID}}</code></td>
              <td><span class="badge">{{.Type}}</span></td>
            </tr>
          {{end}}
        </table>
      </section>
      {{end}}

      {{if .Unverified}}
      <section>
        <h2>Unverified Expectations</h2>
        <p>These expectations are not verified by any verifier or event node.</p>
        <table>
          <tr><th>ID</th><th>Title</th></tr>
          {{range .Unverified}}
            <tr>
              <td><code>{{.ID}}</code></td>
              <td>{{.Title}}</td>
            </tr>
          {{end}}
        </table>
      </section>
      {{end}}
    </aside>

    <div>
      {{if .MermaidCode}}
      <section>
        <h2>Diagram</h2>
        <p>This view is generated from <code>dependency-map.yaml</code>. It is for review only.</p>
        <div class="mermaid-box">
          <pre class="mermaid">
{{.MermaidCode}}
          </pre>
        </div>
      </section>
      {{end}}

      <section>
        <h2>Nodes</h2>
        <div class="toolbar">
          <input id="search" placeholder="Filter nodes..." oninput="filterRows()" />
          <select id="typeFilter" onchange="filterRows()">
            <option value="">All types</option>
            <option>intent</option>
            <option>expectation</option>
            <option>event</option>
            <option>command</option>
            <option>query</option>
            <option>contract</option>
            <option>job</option>
            <option>artifact</option>
            <option>verifier</option>
            <option>approval</option>
          </select>
        </div>
        <table id="nodeTable">
          <thead>
            <tr><th>ID</th><th>Type</th><th>Owner</th><th>Status</th><th>Title</th></tr>
          </thead>
          <tbody>
            {{range .Nodes}}
              <tr>
                <td><code>{{.ID}}</code></td>
                <td><span class="badge">{{.Type}}</span></td>
                <td>{{if .Owner}}{{.Owner}}{{else}}-{{end}}</td>
                <td>{{if .Status}}{{.Status}}{{else}}-{{end}}</td>
                <td>{{.Title}}</td>
              </tr>
            {{end}}
          </tbody>
        </table>
      </section>
    </div>
  </main>

  <script type="module">
    import mermaid from "https://cdn.jsdelivr.net/npm/mermaid/+esm";
    mermaid.initialize({ startOnLoad: true, theme: "default" });
  </script>
  <script>
    function filterRows() {
      const q = document.getElementById('search').value.toLowerCase();
      const type = document.getElementById('typeFilter').value.toLowerCase();
      const rows = document.querySelectorAll('#nodeTable tbody tr');
      rows.forEach(row => {
        const text = row.innerText.toLowerCase();
        const rowType = row.children[1].innerText.toLowerCase();
        row.style.display = text.includes(q) && (!type || rowType === type) ? '' : 'none';
      });
    }
  </script>
</body>
</html>`

var reportOutputFlag string

var reportCmd = &cobra.Command{
	Use:   "report [file.yaml | dir]",
	Short: "Generates a beautiful static HTML review report for a dependency map or directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := args[0]
		data, err := GenerateReportData(targetPath)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}

		tmpl, err := template.New("report").Parse(htmlTemplate)
		if err != nil {
			fmt.Printf("FAIL: failed to parse HTML template: %v\n", err)
			os.Exit(1)
		}

		var out *os.File
		if reportOutputFlag != "" {
			out, err = os.Create(reportOutputFlag)
			if err != nil {
				fmt.Printf("FAIL: failed to create output file %s: %v\n", reportOutputFlag, err)
				os.Exit(1)
			}
			defer out.Close()
		} else {
			out = os.Stdout
		}

		if err := tmpl.Execute(out, data); err != nil {
			fmt.Printf("FAIL: failed to execute template: %v\n", err)
			os.Exit(1)
		}

		if reportOutputFlag != "" {
			fmt.Printf("PASS: Report successfully generated at %s\n", reportOutputFlag)
		}
	},
}

func init() {
	reportCmd.Flags().StringVarP(&reportOutputFlag, "output", "o", "", "Output HTML file path (default: stdout)")
}
