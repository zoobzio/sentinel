package sentinel

import (
	"fmt"
	"reflect"
	"strings"
)

// ERDFormat represents the output format for ERD generation.
type ERDFormat string

const (
	// ERDFormatMermaid generates Mermaid diagram syntax.
	ERDFormatMermaid ERDFormat = "mermaid"
	// ERDFormatDOT generates GraphViz DOT syntax.
	ERDFormatDOT ERDFormat = "dot"
)

// GenerateERD creates an Entity Relationship Diagram from cached type metadata.
// It returns a string representation in the specified format.
func GenerateERD(format ERDFormat) string {
	switch format {
	case ERDFormatMermaid:
		return generateMermaidERD()
	case ERDFormatDOT:
		return generateDOTERD()
	default:
		return generateMermaidERD()
	}
}

// GenerateERDFromRoot creates an ERD starting from a specific root type.
// It only includes types reachable from the root through relationships.
func GenerateERDFromRoot[T any](format ERDFormat) string {
	var zero T
	rootType := getTypeName(reflect.TypeOf(zero))

	// Build reachable set using BFS
	visited := make(map[string]bool)
	queue := []string{rootType}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		// Get relationships for current type
		if metadata, found := instance.cache.Get(current); found {
			for _, rel := range metadata.Relationships {
				if !visited[rel.To] {
					queue = append(queue, rel.To)
				}
			}
		}
	}

	switch format {
	case ERDFormatMermaid:
		return generateMermaidERDFiltered(visited)
	case ERDFormatDOT:
		return generateDOTERDFiltered(visited)
	default:
		return generateMermaidERDFiltered(visited)
	}
}

// generateMermaidERD creates a Mermaid diagram from all cached types.
func generateMermaidERD() string {
	visited := make(map[string]bool)
	for _, typeName := range instance.cache.Keys() {
		visited[typeName] = true
	}
	return generateMermaidERDFiltered(visited)
}

// generateMermaidERDFiltered creates a Mermaid diagram from specified types.
func generateMermaidERDFiltered(includeTypes map[string]bool) string {
	var sb strings.Builder
	sb.WriteString("erDiagram\n")

	// First, declare all entities with their fields
	for typeName := range includeTypes {
		if metadata, found := instance.cache.Get(typeName); found {
			sb.WriteString(fmt.Sprintf("    %s {\n", sanitizeName(typeName)))
			for _, field := range metadata.Fields {
				fieldType := sanitizeType(field.Type)
				sb.WriteString(fmt.Sprintf("        %s %s\n", fieldType, field.Name))
			}
			sb.WriteString("    }\n")
		}
	}

	// Then, declare relationships
	for typeName := range includeTypes {
		if metadata, found := instance.cache.Get(typeName); found {
			for _, rel := range metadata.Relationships {
				if includeTypes[rel.To] {
					relSymbol := getMermaidRelationship(rel.Kind)
					sb.WriteString(fmt.Sprintf("    %s %s %s : %s\n",
						sanitizeName(rel.From),
						relSymbol,
						sanitizeName(rel.To),
						rel.Field))
				}
			}
		}
	}

	return sb.String()
}

// generateDOTERD creates a GraphViz DOT diagram from all cached types.
func generateDOTERD() string {
	visited := make(map[string]bool)
	for _, typeName := range instance.cache.Keys() {
		visited[typeName] = true
	}
	return generateDOTERDFiltered(visited)
}

// generateDOTERDFiltered creates a GraphViz DOT diagram from specified types.
func generateDOTERDFiltered(includeTypes map[string]bool) string {
	var sb strings.Builder
	sb.WriteString("digraph ERD {\n")
	sb.WriteString("    rankdir=LR;\n")
	sb.WriteString("    node [shape=record];\n\n")

	// Declare all entities with their fields
	for typeName := range includeTypes {
		metadata, found := instance.cache.Get(typeName)
		if !found {
			continue
		}
		sb.WriteString(fmt.Sprintf("    %s [label=\"{%s|",
			sanitizeName(typeName),
			typeName))

		var fields []string
		for _, field := range metadata.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s",
				field.Name,
				sanitizeType(field.Type)))
		}
		sb.WriteString(strings.Join(fields, "\\l"))
		sb.WriteString("\\l}\"];\n")
	}

	sb.WriteString("\n")

	// Declare relationships
	for typeName := range includeTypes {
		if metadata, found := instance.cache.Get(typeName); found {
			for _, rel := range metadata.Relationships {
				if includeTypes[rel.To] {
					edgeStyle := getDOTEdgeStyle(rel.Kind)
					sb.WriteString(fmt.Sprintf("    %s -> %s [%s label=%q];\n",
						sanitizeName(rel.From),
						sanitizeName(rel.To),
						edgeStyle,
						rel.Field))
				}
			}
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// getMermaidRelationship converts relationship kind to Mermaid syntax.
func getMermaidRelationship(kind string) string {
	switch kind {
	case RelationshipReference:
		return "||--||" // One-to-one
	case RelationshipCollection:
		return "||--o{" // One-to-many
	case RelationshipEmbedding:
		return "}|--|{" // Embedding/composition
	case RelationshipMap:
		return "||--o{" // Map treated as one-to-many
	default:
		return "||--||"
	}
}

// getDOTEdgeStyle returns GraphViz edge styling for relationship kind.
func getDOTEdgeStyle(kind string) string {
	switch kind {
	case RelationshipReference:
		return "arrowhead=normal"
	case RelationshipCollection:
		return "arrowhead=crow"
	case RelationshipEmbedding:
		return "arrowhead=diamond"
	case RelationshipMap:
		return "arrowhead=crow, style=dashed"
	default:
		return "arrowhead=normal"
	}
}

// sanitizeName ensures names are valid for diagram syntax.
func sanitizeName(name string) string {
	// Replace spaces and special characters
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

// sanitizeType simplifies type names for display.
func sanitizeType(typeName string) string {
	// Remove package prefixes for readability
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		typeName = parts[len(parts)-1]
	}

	// Simplify common types
	typeName = strings.ReplaceAll(typeName, "[]", "Array_")
	typeName = strings.ReplaceAll(typeName, "*", "Ptr_")
	typeName = strings.ReplaceAll(typeName, " ", "_")

	return typeName
}

// GetRelationshipGraph returns the complete relationship graph as a map.
// This is useful for custom visualizations or analysis.
func GetRelationshipGraph() map[string][]TypeRelationship {
	graph := make(map[string][]TypeRelationship)

	for _, typeName := range instance.cache.Keys() {
		if metadata, found := instance.cache.Get(typeName); found {
			graph[typeName] = metadata.Relationships
		}
	}

	return graph
}
