package dag

import (
	"crypto/sha256"
	"fmt"
)

type Color int

const (
	White Color = iota // Unvisited
	Gray               // Visiting
	Black              // Visited
)

type Node struct {
	ID     string `yaml:"id" json:"id"`
	Type   string `yaml:"type" json:"type"`
	Title  string `yaml:"title" json:"title"`
	Ref    string `yaml:"ref,omitempty" json:"ref,omitempty"`
	Owner  string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Status string `yaml:"status,omitempty" json:"status,omitempty"`
	Hash   string `yaml:"hash,omitempty" json:"hash,omitempty"`
}

type Edge struct {
	From      string `yaml:"from" json:"from"`
	To        string `yaml:"to" json:"to"`
	Type      string `yaml:"type" json:"type"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
	Required  bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

type DependencyMap struct {
	Graph struct {
		ID        string `yaml:"id" json:"id"`
		Kind      string `yaml:"kind" json:"kind"`
		Topology  string `yaml:"topology,omitempty" json:"topology,omitempty"` // "dag" (default) oder "graph"
		Status    string `yaml:"status" json:"status"`
		Scope     string `yaml:"scope,omitempty" json:"scope,omitempty"` // "feature" oder "global"
		Generated bool   `yaml:"generated,omitempty" json:"generated,omitempty"`
	} `yaml:"graph" json:"graph"`
	Nodes []Node `yaml:"nodes" json:"nodes"`
	Edges []Edge `yaml:"edges" json:"edges"`
}

type Graph struct {
	Nodes map[string]Node
	Adj   map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]Node),
		Adj:   make(map[string][]string),
	}
}

func (g *Graph) AddNode(n Node) {
	g.Nodes[n.ID] = n
}

func (g *Graph) AddEdge(from, to string) {
	g.Adj[from] = append(g.Adj[from], to)
}

// GenerateMermaidID erzeugt eine stabile, kollisionsarme 10-stellige ID
// für Mermaid-Diagramme, basierend auf dem SHA-256 Hash der Node-ID.
func GenerateMermaidID(nodeID string) string {
	h := sha256.New()
	h.Write([]byte(nodeID))
	return fmt.Sprintf("n_%x", h.Sum(nil))[:12]
}

// FindCycles prüft den Graphen auf Zyklen.
// Gibt bei Erfolg den zyklischen Pfad zurück (z.B. [A, B, C, A]).
func (g *Graph) FindCycles() ([]string, error) {
	colors := make(map[string]Color)
	parent := make(map[string]string)
	var cyclePath []string

	var dfs func(u string) bool
	dfs = func(u string) bool {
		colors[u] = Gray
		for _, v := range g.Adj[u] {
			if colors[v] == Gray {
				// Zyklus gefunden! Wir rekonstruieren den Pfad.
				curr := u
				cyclePath = []string{v, curr}
				for curr != v && parent[curr] != "" {
					curr = parent[curr]
					cyclePath = append([]string{curr}, cyclePath...)
				}
				return true
			}
			if colors[v] == White {
				parent[v] = u
				if dfs(v) {
					return true
				}
			}
		}
		colors[u] = Black
		return false
	}

	for u := range g.Nodes {
		if colors[u] == White {
			if dfs(u) {
				return cyclePath, fmt.Errorf("cycle detected")
			}
		}
	}
	return nil, nil
}
