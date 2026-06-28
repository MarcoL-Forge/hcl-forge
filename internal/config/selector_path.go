package config

import (
	"fmt"
	"strings"
)

type resolvedSelector struct {
	Type    string
	Labels  []string
	Parents []ParentSelector
}

type pathNode struct {
	Type   string
	Labels []string
}

func selectorFromPath(path string) (resolvedSelector, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return resolvedSelector{}, fmt.Errorf("block.path cannot be empty")
	}

	parts := strings.Split(trimmed, ".")
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return resolvedSelector{}, fmt.Errorf("block.path contains empty segment at position %d", i)
		}
	}

	nodes, err := parsePathNodes(parts)
	if err != nil {
		return resolvedSelector{}, err
	}
	if len(nodes) == 0 {
		return resolvedSelector{}, fmt.Errorf("block.path %q does not resolve to any selector", path)
	}

	target := nodes[len(nodes)-1]
	parents := make([]ParentSelector, 0, len(nodes)-1)
	for _, node := range nodes[:len(nodes)-1] {
		parents = append(parents, ParentSelector{
			BlockType: node.Type,
			Labels:    append([]string(nil), node.Labels...),
		})
	}

	return resolvedSelector{
		Type:    target.Type,
		Labels:  append([]string(nil), target.Labels...),
		Parents: parents,
	}, nil
}

func parsePathNodes(parts []string) ([]pathNode, error) {
	nodes := make([]pathNode, 0, len(parts))

	for i := 0; i < len(parts); {
		segment := parts[i]

		switch segment {
		case "resource", "data":
			if i+2 >= len(parts) {
				return nil, fmt.Errorf("block.path requires %s.<type>.<name>", segment)
			}
			nodes = append(nodes, pathNode{Type: segment, Labels: []string{parts[i+1], parts[i+2]}})
			i += 3
		case "module", "variable", "output", "provider":
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("block.path requires %s.<label>", segment)
			}
			nodes = append(nodes, pathNode{Type: segment, Labels: []string{parts[i+1]}})
			i += 2
		case "locals", "terraform":
			nodes = append(nodes, pathNode{Type: segment})
			i++
		default:
			nodes = append(nodes, pathNode{Type: segment})
			i++
		}
	}

	return nodes, nil
}
