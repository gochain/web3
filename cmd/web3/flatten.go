package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type importRec struct {
	FullPath  string
	Code      []string
	Uses      map[string]bool
	Created   bool // has been depended on or processed
	Processed bool // has been processed for includes
	Resolved  bool // all includes have been resolved
	Written   bool // written out
}

func extractFilePath(line string) string {
	line = strings.Replace(line, "import ", "", 2)
	line = strings.Replace(line, "\"", "", 2)
	line = strings.Replace(line, "'", "", 2)
	line = strings.Replace(line, ";", "", 2)
	line = strings.TrimSpace(filepath.Clean(line))
	return line
}

func loadAndSplitFile(imports map[string]importRec, fileName string) (name string, newFiles bool, openZeppelinVersion, pragma string, err error) {
	thisPath := filepath.Dir(fileName)
	shortName := filepath.Base(fileName)
	if imports[shortName].Processed {
		return
	}
	thisRec := importRec{FullPath: fileName, Created: true, Uses: make(map[string]bool)}
	data, err := ioutil.ReadFile(fileName)
	contents := string(data)
	lines := strings.Split(contents, "\n")
	noImports := true
	for li, line := range lines {
		if strings.Contains(line, "@openzeppelin") {
			// get openzep version
			ozi := strings.Index(line, "@openzeppelin")
			vi := strings.Index(line[ozi:], " v")
			openZeppelinVersion = line[ozi+vi+2:]
			// fmt.Println("VERSION:", openZeppelinVersion)
		}
		if strings.HasPrefix(line, "pragma solidity") {
			pragma = line
			continue
		}
		if strings.HasPrefix(line, "import") {
			noImports = false
			fpath := thisPath + "/" + extractFilePath(line)
			fname := filepath.Base(fpath)
			if !imports[fname].Created {
				newFiles = true
				imports[fname] = importRec{
					FullPath: fpath,
					Created:  true,
					Uses:     make(map[string]bool),
				}
			}
			thisRec.Uses[fname] = true
		}
		if strings.HasPrefix(line, "contract") {
			// grab name
			s := strings.Split(line, " ")
			name = s[1]
		}
		if strings.HasPrefix(line, "abstract contract") || strings.HasPrefix(line, "contract") || strings.HasPrefix(line, "library") || strings.HasPrefix(line, "interface") {
			thisRec.Code = lines[li:]
			break
		}
	}
	thisRec.Processed = true
	imports[shortName] = thisRec
	thisRec.Resolved = noImports
	return
}

// FlattenSourceFile flattens the source solidity file, but only if it has imports.
// The flattened file will be generated at output, or in the current directory
// as <source_name>_flatten.sol.
func FlattenSourceFile(ctx context.Context, source, output string) (string, string, error) {
	if output == "" {
		basename := filepath.Base(source)
		output = strings.TrimSuffix(basename, filepath.Ext(basename)) + "_flatten.sol"
	}
	if _, err := os.Stat(source); err != nil {
		return "", "", fmt.Errorf("failed to find source file: %v", err)
	}
	imports := make(map[string]importRec)
	name, newFiles, openZeppelinVersion, pragma, err := loadAndSplitFile(imports, source)
	if err != nil {
		return "", "", err
	}
	_ = newFiles
	_ = openZeppelinVersion
	_ = pragma
	return name, source, err

}
