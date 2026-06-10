package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/japorto100/specdag/dag"
)

func TestLoadAndValidateDependencyMap(t *testing.T) {
	// Test LoadDependencyMap mit Whitespace-Trimming
	t.Run("JSON with whitespaces", func(t *testing.T) {
		tempDir := t.TempDir()
		jsonFile := filepath.Join(tempDir, "dependency-map.json")
		content := `
		{
			"graph": {
				"id": "test-map",
				"kind": "spec_dependency",
				"status": "accepted"
			},
			"nodes": [
				{
					"id": "intent.1",
					"type": "intent",
					"title": "Test Intent"
				}
			],
			"edges": []
		}
		`
		if err := os.WriteFile(jsonFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		dm, err := LoadDependencyMap(jsonFile)
		if err != nil {
			t.Fatalf("LoadDependencyMap failed: %v", err)
		}
		if dm.Graph.ID != "test-map" {
			t.Errorf("expected graph id 'test-map', got '%s'", dm.Graph.ID)
		}
	})

	t.Run("Invalid YAML structure", func(t *testing.T) {
		tempDir := t.TempDir()
		yamlFile := filepath.Join(tempDir, "dependency-map.yaml")
		content := `
graph:
  id: test-map
  kind: spec_dependency
  status: invalid-status
nodes: []
`
		if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		dm, err := LoadDependencyMap(yamlFile)
		if err != nil {
			t.Fatalf("LoadDependencyMap failed: %v", err)
		}
		err = ValidateDependencyMap(dm, false)
		if err == nil {
			t.Error("expected validation to fail for invalid graph status and empty nodes")
		}
	})
}

func TestStrictValidationRules(t *testing.T) {
	tests := []struct {
		name        string
		fromType    string
		toType      string
		edgeType    string
		expectError bool
	}{
		// defines_success_for: intent -> expectation
		{"defines_success_for correct", "intent", "expectation", "defines_success_for", false},
		{"defines_success_for wrong", "event", "expectation", "defines_success_for", true},

		// triggers: event/approval -> job/command
		{"triggers event->job", "event", "job", "triggers", false},
		{"triggers approval->command", "approval", "command", "triggers", false},
		{"triggers wrong", "intent", "job", "triggers", true},

		// produces: job/command/contract -> artifact/event
		{"produces job->artifact", "job", "artifact", "produces", false},
		{"produces command->event", "command", "event", "produces", false},
		{"produces contract->artifact", "contract", "artifact", "produces", false},
		{"produces wrong", "verifier", "event", "produces", true},

		// consumes: job/command -> artifact/event/contract
		{"consumes job->artifact", "job", "artifact", "consumes", false},
		{"consumes command->event", "command", "event", "consumes", false},
		{"consumes job->contract", "job", "contract", "consumes", false},
		{"consumes wrong", "intent", "artifact", "consumes", true},

		// verified_by: artifact/event/job -> verifier
		{"verified_by artifact->verifier", "artifact", "verifier", "verified_by", false},
		{"verified_by event->verifier", "event", "verifier", "verified_by", false},
		{"verified_by job->verifier", "job", "verifier", "verified_by", false},
		{"verified_by wrong", "approval", "verifier", "verified_by", true},

		// verifies: verifier/event -> expectation
		{"verifies verifier->expectation", "verifier", "expectation", "verifies", false},
		{"verifies event->expectation", "event", "expectation", "verifies", false},
		{"verifies wrong", "intent", "expectation", "verifies", true},

		// requires_approval: job/event/command -> approval
		{"requires_approval job->approval", "job", "approval", "requires_approval", false},
		{"requires_approval event->approval", "event", "approval", "requires_approval", false},
		{"requires_approval command->approval", "command", "approval", "requires_approval", false},
		{"requires_approval wrong", "artifact", "approval", "requires_approval", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dm := &dag.DependencyMap{}
			dm.Graph.ID = "strict-test"
			dm.Graph.Kind = "spec_dependency"
			dm.Graph.Status = "accepted"

			dm.Nodes = []dag.Node{
				{ID: "node1", Type: tc.fromType, Title: "Node 1"},
				{ID: "node2", Type: tc.toType, Title: "Node 2"},
			}
			dm.Edges = []dag.Edge{
				{From: "node1", To: "node2", Type: tc.edgeType},
			}

			err := ValidateDependencyMap(dm, true)
			if tc.expectError && err == nil {
				t.Errorf("expected error for %s (%s) -> %s (%s) with edge %s, but got none", tc.fromType, tc.fromType, tc.toType, tc.toType, tc.edgeType)
			}
			if !tc.expectError && err != nil {
				t.Errorf("expected no error for %s (%s) -> %s (%s) with edge %s, but got: %v", tc.fromType, tc.fromType, tc.toType, tc.toType, tc.edgeType, err)
			}
		})
	}
}

func TestTopologyPolicyAndCycles(t *testing.T) {
	t.Run("Cycle detection on DAG", func(t *testing.T) {
		dm := &dag.DependencyMap{}
		dm.Graph.ID = "cycle-test"
		dm.Graph.Kind = "spec_dependency"
		dm.Graph.Status = "accepted"
		dm.Graph.Topology = "dag"

		dm.Nodes = []dag.Node{
			{ID: "A", Type: "intent", Title: "A"},
			{ID: "B", Type: "expectation", Title: "B"},
		}
		dm.Edges = []dag.Edge{
			{From: "A", To: "B", Type: "defines_success_for"},
			{From: "B", To: "A", Type: "verifies"},
		}

		err := ValidateDependencyMap(dm, false)
		if err == nil {
			t.Error("expected cycle validation error for DAG, got nil")
		}
	})

	t.Run("No cycle detection on Graph topology", func(t *testing.T) {
		dm := &dag.DependencyMap{}
		dm.Graph.ID = "cycle-test-graph"
		dm.Graph.Kind = "spec_dependency"
		dm.Graph.Status = "accepted"
		dm.Graph.Topology = "graph"

		dm.Nodes = []dag.Node{
			{ID: "A", Type: "intent", Title: "A"},
			{ID: "B", Type: "expectation", Title: "B"},
		}
		dm.Edges = []dag.Edge{
			{From: "A", To: "B", Type: "defines_success_for"},
			{From: "B", To: "A", Type: "verifies"},
		}

		err := ValidateDependencyMap(dm, false)
		if err != nil {
			t.Errorf("expected no cycle validation error for Graph, got: %v", err)
		}
	})

	t.Run("Assemble Topology Policy", func(t *testing.T) {
		tempDir := t.TempDir()

		// 1. DAG Map
		dagContent := `
graph:
  id: map-dag
  kind: spec_dependency
  status: accepted
  topology: dag
nodes:
  - id: A
    type: intent
    title: A
  - id: B
    type: expectation
    title: B
edges:
  - from: A
    to: B
    type: defines_success_for
`
		// 2. Graph Map (mit Zyklus)
		graphContent := `
graph:
  id: map-graph
  kind: spec_dependency
  status: accepted
  topology: graph
nodes:
  - id: C
    type: intent
    title: C
  - id: D
    type: expectation
    title: D
edges:
  - from: C
    to: D
    type: defines_success_for
  - from: D
    to: C
    type: verifies
`
		dirA := filepath.Join(tempDir, "feature-dag")
		dirB := filepath.Join(tempDir, "feature-graph")
		os.MkdirAll(dirA, 0755)
		os.MkdirAll(dirB, 0755)

		os.WriteFile(filepath.Join(dirA, "dependency-map.yaml"), []byte(dagContent), 0644)
		os.WriteFile(filepath.Join(dirB, "dependency-map.yaml"), []byte(graphContent), 0644)

		// Ohne includeGraphs (standardmäßig) -> map-graph überspringen
		result1, err := AssembleDirectory(tempDir, "json", false)
		if err != nil {
			t.Fatalf("AssembleDirectory failed: %v", err)
		}

		if !containsNode(result1, "A") || containsNode(result1, "C") {
			t.Errorf("expected only A to be assembled, got: %s", result1)
		}

		// Mit includeGraphs -> map-graph einbinden -> Zyklenfehler beim globalen DAG-Check
		_, err = AssembleDirectory(tempDir, "json", true)
		if err == nil {
			t.Error("expected global cycle detection error when including cyclic graphs, but got nil")
		}
	})
}

func containsNode(jsonStr, nodeID string) bool {
	return strings.Contains(jsonStr, `"id": "`+nodeID+`"`) || strings.Contains(jsonStr, `"id":"`+nodeID+`"`)
}

func TestDoctorAndCatalogs(t *testing.T) {
	t.Run("Doctor checks and severity warning", func(t *testing.T) {
		tempDir := t.TempDir()
		
		featuresDir := filepath.Join(tempDir, "features")
		os.MkdirAll(featuresDir, 0755)

		// 1. Feature 1: Keine spec.md oder Map (severity unknown)
		f1Dir := filepath.Join(featuresDir, "001-f1")
		os.MkdirAll(f1Dir, 0755)

		// 2. Feature 2: spec.md mit severity 3, aber keine Map
		f2Dir := filepath.Join(featuresDir, "002-f2")
		os.MkdirAll(f2Dir, 0755)
		specContent := `---
severity: 3
---
# Feature 2 spec
`
		os.WriteFile(filepath.Join(f2Dir, "spec.md"), []byte(specContent), 0644)

		// Doctor ausführen
		errors, warnings, err := RunDoctor(tempDir)
		if err != nil {
			t.Fatalf("RunDoctor failed: %v", err)
		}

		// Feature 2 verlangt Map, da severity >= 2, also Warnung
		if warnings == 0 {
			t.Errorf("expected warnings for missing dependency maps, got 0")
		}
		if errors > 0 {
			t.Errorf("expected 0 errors, got %d", errors)
		}
	})

	t.Run("Check catalogs name mismatch and status conflict", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mapContent := `
graph:
  id: test-catalog-map
  kind: spec_dependency
  status: accepted
nodes:
  - id: event.user.registered
    type: event
    title: User Registered Event
    status: accepted
    ref: catalog/user-registered.md
edges: []
`
		os.MkdirAll(filepath.Join(tempDir, "catalog"), 0755)
		os.WriteFile(filepath.Join(tempDir, "dependency-map.yaml"), []byte(mapContent), 0644)

		// Catalog-Datei mit Mismatch und Statuskonflikt erstellen
		catalogContent := `---
name: user.signed_up
status: draft
---
# User Registered
`
		os.WriteFile(filepath.Join(tempDir, "catalog", "user-registered.md"), []byte(catalogContent), 0644)

		// CheckCatalogs ausführen
		errors, warnings, err := RunCheckCatalogs(tempDir)
		if err != nil {
			t.Fatalf("RunCheckCatalogs failed: %v", err)
		}

		// Name Mismatch und Statuskonflikt müssen Warnungen sein
		if warnings == 0 {
			t.Errorf("expected warnings for mismatches, got 0")
		}
		if errors > 0 {
			t.Errorf("expected 0 errors, got %d", errors)
		}
	})
}
