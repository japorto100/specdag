package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeMerkleAttestationDeterministicAcrossOrder(t *testing.T) {
	tempDir := t.TempDir()
	evidencePath := filepath.Join(tempDir, "evidence.md")
	if err := os.WriteFile(evidencePath, []byte("accepted evidence\n"), 0644); err != nil {
		t.Fatalf("failed to write evidence file: %v", err)
	}

	firstMap := `
graph:
  id: merkle-test
  kind: spec_dependency
  status: accepted
  topology: dag
nodes:
  - id: intent.main
    type: intent
    title: Main intent
    status: accepted
  - id: expectation.ready
    type: expectation
    title: Ready expectation
  - id: verifier.evidence
    type: verifier
    title: Evidence verifier
    ref: evidence.md
edges:
  - from: intent.main
    to: expectation.ready
    type: defines_success_for
  - from: verifier.evidence
    to: expectation.ready
    type: verifies
`
	secondMap := `
graph:
  topology: dag
  status: accepted
  kind: spec_dependency
  id: merkle-test
nodes:
  - title: Evidence verifier
    ref: evidence.md
    type: verifier
    id: verifier.evidence
  - title: Ready expectation
    id: expectation.ready
    type: expectation
  - status: accepted
    title: Main intent
    id: intent.main
    type: intent
edges:
  - type: verifies
    to: expectation.ready
    from: verifier.evidence
  - type: defines_success_for
    to: expectation.ready
    from: intent.main
`

	firstPath := filepath.Join(tempDir, "first.yaml")
	secondPath := filepath.Join(tempDir, "second.yaml")
	if err := os.WriteFile(firstPath, []byte(firstMap), 0644); err != nil {
		t.Fatalf("failed to write first map: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(secondMap), 0644); err != nil {
		t.Fatalf("failed to write second map: %v", err)
	}

	first, err := ComputeMerkleAttestation(firstPath)
	if err != nil {
		t.Fatalf("ComputeMerkleAttestation(first) failed: %v", err)
	}
	second, err := ComputeMerkleAttestation(secondPath)
	if err != nil {
		t.Fatalf("ComputeMerkleAttestation(second) failed: %v", err)
	}

	if first.RootHash == "" {
		t.Fatal("expected non-empty Merkle root")
	}
	if first.RootHash != second.RootHash {
		t.Fatalf("expected deterministic Merkle root, got %s and %s", first.RootHash, second.RootHash)
	}
	if len(first.Nodes) != 3 {
		t.Fatalf("expected 3 node hashes, got %d", len(first.Nodes))
	}
	for _, node := range first.Nodes {
		if node.ContentHash == "" || node.MerkleHash == "" {
			t.Fatalf("expected hashes for node %s, got content=%q merkle=%q", node.ID, node.ContentHash, node.MerkleHash)
		}
	}
}

func TestComputeMerkleAttestationRefContentChangesRoot(t *testing.T) {
	tempDir := t.TempDir()
	evidencePath := filepath.Join(tempDir, "evidence.md")
	if err := os.WriteFile(evidencePath, []byte("first evidence\n"), 0644); err != nil {
		t.Fatalf("failed to write evidence file: %v", err)
	}

	mapPath := filepath.Join(tempDir, "dependency-map.yaml")
	content := `
graph:
  id: ref-content-test
  kind: spec_dependency
  status: accepted
nodes:
  - id: verifier.evidence
    type: verifier
    title: Evidence verifier
    ref: evidence.md
edges: []
`
	if err := os.WriteFile(mapPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write map: %v", err)
	}

	first, err := ComputeMerkleAttestation(mapPath)
	if err != nil {
		t.Fatalf("ComputeMerkleAttestation(first) failed: %v", err)
	}
	if writeErr := os.WriteFile(evidencePath, []byte("changed evidence\n"), 0644); writeErr != nil {
		t.Fatalf("failed to update evidence file: %v", writeErr)
	}
	second, err := ComputeMerkleAttestation(mapPath)
	if err != nil {
		t.Fatalf("ComputeMerkleAttestation(second) failed: %v", err)
	}

	if first.RootHash == second.RootHash {
		t.Fatalf("expected changed ref content to change Merkle root %s", first.RootHash)
	}
}

func TestComputeMerkleAttestationMissingRefWarns(t *testing.T) {
	tempDir := t.TempDir()
	mapPath := filepath.Join(tempDir, "dependency-map.yaml")
	content := `
graph:
  id: missing-ref-test
  kind: spec_dependency
  status: accepted
nodes:
  - id: verifier.missing
    type: verifier
    title: Missing verifier
    ref: missing.md
edges: []
`
	if err := os.WriteFile(mapPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write map: %v", err)
	}

	attestation, err := ComputeMerkleAttestation(mapPath)
	if err != nil {
		t.Fatalf("ComputeMerkleAttestation failed: %v", err)
	}

	if attestation.RootHash == "" {
		t.Fatal("expected non-empty Merkle root")
	}
	if len(attestation.Warnings) == 0 {
		t.Fatal("expected missing ref warning")
	}
	if !strings.Contains(attestation.Warnings[0], "missing.md") {
		t.Fatalf("expected warning to mention missing ref, got %q", attestation.Warnings[0])
	}
}
