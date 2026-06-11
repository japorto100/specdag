package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/japorto100/specdag/dag"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const merkleAlgorithm = "sha256"

type MerkleAttestation struct {
	GraphID   string           `json:"graphId" yaml:"graphId"`
	Algorithm string           `json:"algorithm" yaml:"algorithm"`
	RootHash  string           `json:"rootHash" yaml:"rootHash"`
	Nodes     []MerkleNodeHash `json:"nodes" yaml:"nodes"`
	Warnings  []string         `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

type MerkleNodeHash struct {
	ID          string `json:"id" yaml:"id"`
	ContentHash string `json:"contentHash" yaml:"contentHash"`
	MerkleHash  string `json:"merkleHash" yaml:"merkleHash"`
	RefHash     string `json:"refHash,omitempty" yaml:"refHash,omitempty"`
}

type merkleVerificationResult struct {
	GraphID      string           `json:"graphId" yaml:"graphId"`
	Algorithm    string           `json:"algorithm" yaml:"algorithm"`
	RootHash     string           `json:"rootHash" yaml:"rootHash"`
	ExpectedHash string           `json:"expectedHash" yaml:"expectedHash"`
	Verified     bool             `json:"verified" yaml:"verified"`
	Nodes        []MerkleNodeHash `json:"nodes" yaml:"nodes"`
	Warnings     []string         `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

type merkleCalculator struct {
	depMap        *dag.DependencyMap
	baseDir       string
	nodes         map[string]dag.Node
	children      map[string][]merkleEdgeRef
	contentHashes map[string]MerkleNodeHash
	merkleHashes  map[string]string
	visiting      map[string]bool
	visited       map[string]bool
	warnings      []string
}

type merkleEdgeRef struct {
	To        string `json:"to"`
	Type      string `json:"type"`
	Condition string `json:"condition,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

type merkleChildPayload struct {
	To         string `json:"to"`
	Type       string `json:"type"`
	Condition  string `json:"condition,omitempty"`
	Required   bool   `json:"required,omitempty"`
	MerkleHash string `json:"merkleHash"`
}

type merkleEdgePayload struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"`
	Condition string `json:"condition,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

// ComputeMerkleAttestation builds deterministic SHA-256 hashes for a dependency
// map. Node content hashes include structural node fields and referenced file
// content; node Merkle hashes recursively include outgoing child hashes.
func ComputeMerkleAttestation(filePath string) (*MerkleAttestation, error) {
	depMap, err := LoadDependencyMap(filePath)
	if err != nil {
		return nil, err
	}
	if err := ValidateDependencyMap(depMap, false); err != nil {
		return nil, err
	}

	calc := newMerkleCalculator(depMap, filepath.Dir(filePath))
	return calc.compute()
}

func VerifyMerkleRoot(filePath, expectedHash string) (*MerkleAttestation, bool, error) {
	attestation, err := ComputeMerkleAttestation(filePath)
	if err != nil {
		return nil, false, err
	}

	expected := normalizeMerkleHash(expectedHash)
	return attestation, attestation.RootHash == expected, nil
}

func newMerkleCalculator(depMap *dag.DependencyMap, baseDir string) *merkleCalculator {
	nodes := make(map[string]dag.Node, len(depMap.Nodes))
	for _, node := range depMap.Nodes {
		nodes[node.ID] = node
	}

	children := make(map[string][]merkleEdgeRef)
	for _, edge := range depMap.Edges {
		children[edge.From] = append(children[edge.From], merkleEdgeRef{
			To:        edge.To,
			Type:      edge.Type,
			Condition: edge.Condition,
			Required:  edge.Required,
		})
	}
	for from := range children {
		sortMerkleEdgeRefs(children[from])
	}

	return &merkleCalculator{
		depMap:        depMap,
		baseDir:       baseDir,
		nodes:         nodes,
		children:      children,
		contentHashes: make(map[string]MerkleNodeHash, len(depMap.Nodes)),
		merkleHashes:  make(map[string]string, len(depMap.Nodes)),
		visiting:      make(map[string]bool, len(depMap.Nodes)),
		visited:       make(map[string]bool, len(depMap.Nodes)),
	}
}

func (c *merkleCalculator) compute() (*MerkleAttestation, error) {
	nodeIDs := sortedNodeIDs(c.depMap.Nodes)
	for _, nodeID := range nodeIDs {
		if _, err := c.computeNodeMerkleHash(nodeID); err != nil {
			return nil, err
		}
	}

	nodes := make([]MerkleNodeHash, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeHash := c.contentHashes[nodeID]
		nodeHash.MerkleHash = c.merkleHashes[nodeID]
		nodes = append(nodes, nodeHash)
	}

	edges := make([]merkleEdgePayload, 0, len(c.depMap.Edges))
	for _, edge := range c.depMap.Edges {
		edges = append(edges, merkleEdgePayload{
			From:      edge.From,
			To:        edge.To,
			Type:      edge.Type,
			Condition: edge.Condition,
			Required:  edge.Required,
		})
	}
	sortMerkleEdgePayloads(edges)

	rootHash, err := hashCanonical(struct {
		Schema string              `json:"schema"`
		Graph  merkleGraphPayload  `json:"graph"`
		Nodes  []MerkleNodeHash    `json:"nodes"`
		Edges  []merkleEdgePayload `json:"edges"`
	}{
		Schema: "specdag.merkle.root.v1",
		Graph: merkleGraphPayload{
			ID:        c.depMap.Graph.ID,
			Kind:      c.depMap.Graph.Kind,
			Topology:  c.depMap.Graph.Topology,
			Status:    c.depMap.Graph.Status,
			Scope:     c.depMap.Graph.Scope,
			Generated: c.depMap.Graph.Generated,
		},
		Nodes: nodes,
		Edges: edges,
	})
	if err != nil {
		return nil, err
	}

	return &MerkleAttestation{
		GraphID:   c.depMap.Graph.ID,
		Algorithm: merkleAlgorithm,
		RootHash:  rootHash,
		Nodes:     nodes,
		Warnings:  c.warnings,
	}, nil
}

type merkleGraphPayload struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Topology  string `json:"topology,omitempty"`
	Status    string `json:"status"`
	Scope     string `json:"scope,omitempty"`
	Generated bool   `json:"generated,omitempty"`
}

func (c *merkleCalculator) computeNodeMerkleHash(nodeID string) (string, error) {
	if c.visited[nodeID] {
		return c.merkleHashes[nodeID], nil
	}
	if c.visiting[nodeID] {
		return "", fmt.Errorf("cannot compute Merkle-DAG hash for cyclic graph at node %s", nodeID)
	}

	node, ok := c.nodes[nodeID]
	if !ok {
		return "", fmt.Errorf("cannot compute Merkle-DAG hash: missing node %s", nodeID)
	}

	contentHash, err := c.computeNodeContentHash(node)
	if err != nil {
		return "", err
	}

	c.visiting[nodeID] = true
	children := make([]merkleChildPayload, 0, len(c.children[nodeID]))
	for _, child := range c.children[nodeID] {
		childHash, childErr := c.computeNodeMerkleHash(child.To)
		if childErr != nil {
			return "", childErr
		}
		children = append(children, merkleChildPayload{
			To:         child.To,
			Type:       child.Type,
			Condition:  child.Condition,
			Required:   child.Required,
			MerkleHash: childHash,
		})
	}
	sortMerkleChildPayloads(children)

	merkleHash, err := hashCanonical(struct {
		Schema      string               `json:"schema"`
		ID          string               `json:"id"`
		ContentHash string               `json:"contentHash"`
		Children    []merkleChildPayload `json:"children"`
	}{
		Schema:      "specdag.merkle.node.v1",
		ID:          node.ID,
		ContentHash: contentHash.ContentHash,
		Children:    children,
	})
	if err != nil {
		return "", err
	}

	c.visiting[nodeID] = false
	c.visited[nodeID] = true
	c.merkleHashes[nodeID] = merkleHash
	return merkleHash, nil
}

func (c *merkleCalculator) computeNodeContentHash(node dag.Node) (MerkleNodeHash, error) {
	if cached, ok := c.contentHashes[node.ID]; ok {
		return cached, nil
	}

	refState := "none"
	refHash := ""
	if node.Ref != "" {
		refState = "present"
		refPath := node.Ref
		if !filepath.IsAbs(refPath) {
			refPath = filepath.Join(c.baseDir, node.Ref)
		}

		data, err := os.ReadFile(refPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				refState = "missing"
				c.warnings = append(c.warnings, fmt.Sprintf("node %s references missing file %s", node.ID, node.Ref))
			} else {
				return MerkleNodeHash{}, fmt.Errorf("cannot read ref for node %s (%s): %w", node.ID, node.Ref, err)
			}
		} else {
			refHash = hashBytes(data)
		}
	}

	contentHash, err := hashCanonical(struct {
		Schema   string `json:"schema"`
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   string `json:"status,omitempty"`
		Ref      string `json:"ref,omitempty"`
		RefState string `json:"refState"`
		RefHash  string `json:"refHash,omitempty"`
	}{
		Schema:   "specdag.merkle.node-content.v1",
		Type:     node.Type,
		Title:    node.Title,
		Status:   node.Status,
		Ref:      node.Ref,
		RefState: refState,
		RefHash:  refHash,
	})
	if err != nil {
		return MerkleNodeHash{}, err
	}

	nodeHash := MerkleNodeHash{
		ID:          node.ID,
		ContentHash: contentHash,
		RefHash:     refHash,
	}
	c.contentHashes[node.ID] = nodeHash
	return nodeHash, nil
}

func hashCanonical(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal canonical hash payload: %w", err)
	}
	return hashBytes(data), nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedNodeIDs(nodes []dag.Node) []string {
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	sort.Strings(ids)
	return ids
}

func sortMerkleEdgeRefs(edges []merkleEdgeRef) {
	sort.Slice(edges, func(i, j int) bool {
		return compareEdgeParts(edges[i].To, edges[i].Type, edges[i].Condition, edges[i].Required, edges[j].To, edges[j].Type, edges[j].Condition, edges[j].Required)
	})
}

func sortMerkleChildPayloads(children []merkleChildPayload) {
	sort.Slice(children, func(i, j int) bool {
		if children[i].To != children[j].To {
			return children[i].To < children[j].To
		}
		if children[i].Type != children[j].Type {
			return children[i].Type < children[j].Type
		}
		if children[i].Condition != children[j].Condition {
			return children[i].Condition < children[j].Condition
		}
		if children[i].Required != children[j].Required {
			return !children[i].Required && children[j].Required
		}
		return children[i].MerkleHash < children[j].MerkleHash
	})
}

func sortMerkleEdgePayloads(edges []merkleEdgePayload) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return compareEdgeParts(edges[i].To, edges[i].Type, edges[i].Condition, edges[i].Required, edges[j].To, edges[j].Type, edges[j].Condition, edges[j].Required)
	})
}

func compareEdgeParts(leftTo, leftType, leftCondition string, leftRequired bool, rightTo, rightType, rightCondition string, rightRequired bool) bool {
	if leftTo != rightTo {
		return leftTo < rightTo
	}
	if leftType != rightType {
		return leftType < rightType
	}
	if leftCondition != rightCondition {
		return leftCondition < rightCondition
	}
	if leftRequired != rightRequired {
		return !leftRequired && rightRequired
	}
	return false
}

func normalizeMerkleHash(hash string) string {
	normalized := strings.TrimSpace(strings.ToLower(hash))
	normalized = strings.TrimPrefix(normalized, "sha256:")
	return normalized
}

func renderMerkleText(attestation *MerkleAttestation, verification *merkleVerificationResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Root: %s\n", attestation.RootHash)
	fmt.Fprintf(&builder, "Algorithm: %s\n", attestation.Algorithm)
	fmt.Fprintf(&builder, "Graph: %s\n", attestation.GraphID)
	if verification != nil {
		if verification.Verified {
			builder.WriteString("Verification: PASS\n")
		} else {
			fmt.Fprintf(&builder, "Verification: FAIL (expected %s)\n", verification.ExpectedHash)
		}
	}
	if len(attestation.Warnings) > 0 {
		builder.WriteString("Warnings:\n")
		for _, warning := range attestation.Warnings {
			fmt.Fprintf(&builder, "- %s\n", warning)
		}
	}
	return builder.String()
}

func renderMerkleOutput(attestation *MerkleAttestation, verification *merkleVerificationResult, format string) (string, error) {
	switch format {
	case "text", "":
		return renderMerkleText(attestation, verification), nil
	case "json":
		if verification != nil {
			data, err := json.MarshalIndent(verification, "", "  ")
			if err != nil {
				return "", fmt.Errorf("marshal Merkle verification JSON: %w", err)
			}
			return string(data) + "\n", nil
		}
		data, err := json.MarshalIndent(attestation, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal Merkle attestation JSON: %w", err)
		}
		return string(data) + "\n", nil
	case "yaml":
		var data []byte
		var err error
		if verification != nil {
			data, err = yaml.Marshal(verification)
		} else {
			data, err = yaml.Marshal(attestation)
		}
		if err != nil {
			return "", fmt.Errorf("marshal Merkle output YAML: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported format %q (expected text, json, or yaml)", format)
	}
}

func writeMerkleOutput(output, outputPath string) error {
	if outputPath == "" {
		fmt.Print(output)
		return nil
	}
	if err := os.WriteFile(outputPath, []byte(output), 0600); err != nil {
		return fmt.Errorf("write Merkle output %s: %w", outputPath, err)
	}
	return nil
}

func buildVerificationResult(attestation *MerkleAttestation, expectedHash string, verified bool) *merkleVerificationResult {
	return &merkleVerificationResult{
		GraphID:      attestation.GraphID,
		Algorithm:    attestation.Algorithm,
		RootHash:     attestation.RootHash,
		ExpectedHash: normalizeMerkleHash(expectedHash),
		Verified:     verified,
		Nodes:        attestation.Nodes,
		Warnings:     attestation.Warnings,
	}
}

var hashFormat string
var hashOutput string
var hashExpected string

var hashCmd = &cobra.Command{
	Use:   "hash [file.yaml|file.json]",
	Short: "Computes a deterministic Merkle-DAG attestation for a dependency map",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		attestation, err := ComputeMerkleAttestation(filePath)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}

		var verification *merkleVerificationResult
		if hashExpected != "" {
			verified := attestation.RootHash == normalizeMerkleHash(hashExpected)
			verification = buildVerificationResult(attestation, hashExpected, verified)
		}

		output, err := renderMerkleOutput(attestation, verification, hashFormat)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		if err := writeMerkleOutput(output, hashOutput); err != nil {
			fmt.Printf("FAIL: cannot write output: %v\n", err)
			os.Exit(1)
		}
		if verification != nil && !verification.Verified {
			os.Exit(1)
		}
	},
}

var verifyHash string
var verifyFormat string
var verifyOutput string

var verifyCmd = &cobra.Command{
	Use:   "verify [file.yaml|file.json] --hash <expected-root>",
	Short: "Validates a dependency map and verifies its Merkle root",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verifyHash == "" {
			fmt.Println("FAIL: --hash is required")
			os.Exit(1)
		}

		filePath := args[0]
		attestation, verified, err := VerifyMerkleRoot(filePath, verifyHash)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}

		verification := buildVerificationResult(attestation, verifyHash, verified)
		output, err := renderMerkleOutput(attestation, verification, verifyFormat)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		if err := writeMerkleOutput(output, verifyOutput); err != nil {
			fmt.Printf("FAIL: cannot write output: %v\n", err)
			os.Exit(1)
		}
		if !verified {
			os.Exit(1)
		}
	},
}

func init() {
	hashCmd.Flags().StringVar(&hashFormat, "format", "text", "Output format: text, json, or yaml")
	hashCmd.Flags().StringVarP(&hashOutput, "output", "o", "", "Write attestation output to a file")
	hashCmd.Flags().StringVar(&hashExpected, "expected", "", "Expected Merkle root to verify after hashing")

	verifyCmd.Flags().StringVar(&verifyHash, "hash", "", "Expected Merkle root hash")
	verifyCmd.Flags().StringVar(&verifyFormat, "format", "text", "Output format: text, json, or yaml")
	verifyCmd.Flags().StringVarP(&verifyOutput, "output", "o", "", "Write verification output to a file")
}
