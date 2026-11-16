package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	importPathRegexp   = regexp.MustCompile(`(?s)^\s*import\s+(?:[^"']*from\s+)?["']([^"']+)["']\s*;`)
	spdxRegexp         = regexp.MustCompile(`(?i)^\s*//\s*SPDX-License-Identifier:\s*(.+)$`)
	pragmaLineRegexp   = regexp.MustCompile(`^\s*pragma\s+solidity\s+[^;]+;`)
	ozCommentRegexp    = regexp.MustCompile(`@openzeppelin\s+v([0-9]+\.[0-9]+\.[0-9]+)`)
	contractDeclRegexp = regexp.MustCompile(`(?m)^\s*contract\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
)

type flatNode struct {
	canonPath string
	deps      []string
	bodyLines []string
}

// FlattenSourceFile flattens a Solidity file and its transitive imports in topological order.
// If the file has no imports, it returns (primaryContractName, source, nil) without writing a new file
// unless output was explicitly specified.
func FlattenSourceFile(ctx context.Context, source, output string) (string, string, error) {
	if _, err := os.Stat(source); err != nil {
		return "", "", fmt.Errorf("failed to find source file: %w", err)
	}
	rawRoot, err := os.ReadFile(source)
	if err != nil {
		return "", "", err
	}
	rootStr := string(rawRoot)
	primaryName := detectPrimaryContract(rootStr)

	// Check if root has any imports
	if !hasImports(rootStr) && output == "" {
		return primaryName, source, nil
	}

	// Ensure OpenZeppelin contracts exist if referenced
	if strings.Contains(rootStr, "lib/oz/") || strings.Contains(rootStr, "@openzeppelin/contracts/") {
		ozVer := OpenZeppelinVersion
		if m := ozCommentRegexp.FindStringSubmatch(rootStr); len(m) == 2 {
			ozVer = strings.TrimSpace(m[1])
		}
		if err := getOpenZeppelinLib(ctx, ozVer); err != nil {
			return primaryName, "", fmt.Errorf("failed to fetch OpenZeppelin v%s: %w", ozVer, err)
		}
	}

	if output == "" {
		basename := filepath.Base(source)
		output = strings.TrimSuffix(basename, filepath.Ext(basename)) + "_flatten.sol"
	}

	nodes := make(map[string]*flatNode)
	var topSPDX, topPragma string

	var visitLoad func(filePath string) (string, error)
	visitLoad = func(filePath string) (string, error) {
		canon := filepath.Clean(filePath)
		if _, exists := nodes[canon]; exists {
			return canon, nil
		}
		contentBytes, err := os.ReadFile(canon)
		if err != nil {
			return "", fmt.Errorf("cannot read imported file %q: %w", canon, err)
		}
		node := &flatNode{canonPath: canon}
		nodes[canon] = node

		dir := filepath.Dir(canon)
		lines := strings.Split(strings.ReplaceAll(string(contentBytes), "\r\n", "\n"), "\n")
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			if m := spdxRegexp.FindStringSubmatch(line); len(m) == 2 {
				if topSPDX == "" {
					topSPDX = strings.TrimSpace(m[1])
				}
				continue
			}
			if pragmaLineRegexp.MatchString(line) {
				if topPragma == "" {
					topPragma = strings.TrimSpace(line)
				}
				continue
			}
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import{") || strings.HasPrefix(trimmed, "import\"") || strings.HasPrefix(trimmed, "import'") || trimmed == "import" {
				stmt := line
				for !strings.Contains(stmt, ";") && i+1 < len(lines) {
					i++
					stmt += "\n" + lines[i]
				}
				if m := importPathRegexp.FindStringSubmatch(stmt); len(m) == 2 {
					imp := m[1]
					resolved := resolveImportPath(dir, imp)
					depCanon, err := visitLoad(resolved)
					if err != nil {
						return "", err
					}
					node.deps = append(node.deps, depCanon)
					continue
				}
			}
			node.bodyLines = append(node.bodyLines, line)
		}
		return canon, nil
	}

	rootCanon, err := visitLoad(source)
	if err != nil {
		return primaryName, "", err
	}

	// Topological sort with cycle detection
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := make(map[string]int, len(nodes))
	var order []string
	var topo func(k string) error
	topo = func(k string) error {
		switch state[k] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("circular Solidity import involving %q", k)
		}
		state[k] = visiting
		for _, dep := range nodes[k].deps {
			if err := topo(dep); err != nil {
				return err
			}
		}
		state[k] = visited
		order = append(order, k)
		return nil
	}
	if err := topo(rootCanon); err != nil {
		return primaryName, "", err
	}

	var out bytes.Buffer
	if topSPDX != "" {
		fmt.Fprintf(&out, "// SPDX-License-Identifier: %s\n", topSPDX)
	}
	if topPragma != "" {
		fmt.Fprintf(&out, "%s\n\n", topPragma)
	}
	for _, k := range order {
		body := strings.TrimSpace(strings.Join(nodes[k].bodyLines, "\n"))
		if body == "" {
			continue
		}
		fmt.Fprintf(&out, "// File: %s\n\n%s\n\n", filepath.ToSlash(k), body)
	}

	if parent := filepath.Dir(output); parent != "" && parent != "." {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return primaryName, "", fmt.Errorf("failed to create parent directory: %w", err)
		}
	}
	if err := os.WriteFile(output, out.Bytes(), 0644); err != nil {
		return primaryName, "", fmt.Errorf("failed to write flattened file %q: %w", output, err)
	}
	return primaryName, output, nil
}

func resolveImportPath(currentDir, imp string) string {
	if strings.HasPrefix(imp, "@openzeppelin/contracts/") {
		rel := strings.TrimPrefix(imp, "@openzeppelin/contracts/")
		return filepath.Join("lib", "oz", "contracts", filepath.FromSlash(rel))
	}
	return filepath.Join(currentDir, filepath.FromSlash(imp))
}

var anyImportRegexp = regexp.MustCompile(`(?ms)^\s*import\s+(?:[^"']*from\s+)?["']([^"']+)["']\s*;`)

func hasImports(src string) bool {
	return anyImportRegexp.MatchString(src)
}

func detectPrimaryContract(src string) string {
	matches := contractDeclRegexp.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		return ""
	}
	// Return the last concrete `contract Name` declared in the root source file.
	return matches[len(matches)-1][1]
}
