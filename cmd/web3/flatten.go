package main

import (
	"bufio"
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
	if newFiles { //file has imports
		err = getOpenZeppelinLib(ctx, openZeppelinVersion)
		if err != nil {
			return name, "", fmt.Errorf("failed to get openzeppelin lib: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(output), 0777); err != nil {
			return name, "", fmt.Errorf("failed to make parent directories: %v", err)
		}
		f, _ := os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		defer f.Close()
		w := bufio.NewWriter(f)
		for {
			repeat := false
			for _, iRec := range imports {
				if iRec.Processed {
					continue
				}
				// fmt.Println("handling:", iRec.FullPath)
				_, newFiles2, _, _, err2 := loadAndSplitFile(imports, iRec.FullPath)
				if err2 != nil {
					return name, output, err2
				}
				repeat = repeat || newFiles2
			}
			if !repeat {
				break
			}
		}
		fmt.Fprintln(w, pragma)
		for {
			completed := true
			for key, mp := range imports {
				if mp.Written {
					continue
				}
				completed = false
				if mp.Resolved {
					for _, line := range mp.Code {
						fmt.Fprintln(w, line)
					}
					mp.Written = true
					imports[key] = mp
					continue
				}
				amResolved := true
				for k2 := range mp.Uses {
					if !imports[filepath.Base(k2)].Written {
						amResolved = false
					}
				}
				if amResolved {
					mp.Resolved = true
					imports[key] = mp
					continue
				}
			}
			if completed {
				break
			}
		}
		return name, output, w.Flush()
	} //file doesn't have any imports, so just return the same file
	return name, source, err

}
