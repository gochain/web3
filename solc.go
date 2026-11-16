package web3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	pragmaRegexp = regexp.MustCompile(`(?m)^\s*pragma\s+solidity\s+([^;]+);`)
	semverRegexp = regexp.MustCompile(`([0-9]+)\.([0-9]+)(?:\.([0-9]+))?`)
)

// Contract contains information about a compiled contract, alongside its creation and runtime bytecode.
type Contract struct {
	Code        string            `json:"code"`
	RuntimeCode string            `json:"runtime-code"`
	Info        ContractInfo      `json:"info"`
	Hashes      map[string]string `json:"hashes"`
}

// ContractInfo contains ABI, source maps, docs, and compiler metadata for a compiled contract.
type ContractInfo struct {
	Source          string      `json:"source"`
	Language        string      `json:"language"`
	LanguageVersion string      `json:"languageVersion"`
	CompilerVersion string      `json:"compilerVersion"`
	CompilerOptions string      `json:"compilerOptions"`
	SrcMap          interface{} `json:"srcMap"`
	SrcMapRuntime   string      `json:"srcMapRuntime"`
	AbiDefinition   interface{} `json:"abiDefinition"`
	UserDoc         interface{} `json:"userDoc"`
	DeveloperDoc    interface{} `json:"developerDoc"`
	Metadata        string      `json:"metadata"`
}

// Solidity specifies the Solidity compiler configuration.
type Solidity struct {
	Path, Version, EVMVersion string
	Major, Minor, Patch       int
	Optimize                  bool
	UseDocker                 bool
}

type solcOutput struct {
	Contracts map[string]struct {
		BinRuntime                                  string `json:"bin-runtime"`
		SrcMapRuntime                               string `json:"srcmap-runtime"`
		Bin, SrcMap, Abi, Devdoc, Userdoc, Metadata string
	}
	Version string
}

type solcOutputV8 struct {
	Contracts map[string]struct {
		BinRuntime            string `json:"bin-runtime"`
		SrcMapRuntime         string `json:"srcmap-runtime"`
		Bin, SrcMap, Metadata string
		Abi                   interface{}
		Devdoc                interface{}
		Userdoc               interface{}
		Hashes                map[string]string
	}
	Version string
}

func (s *Solidity) solcFlags() []string {
	flags := []string{
		"--combined-json",
		"bin,bin-runtime,srcmap,srcmap-runtime,abi,userdoc,devdoc,metadata",
	}
	if s.EVMVersion != "" {
		flags = append(flags, "--evm-version", s.EVMVersion)
	}
	if s.Optimize {
		flags = append(flags, "--optimize")
	}
	return flags
}

func (s *Solidity) makeArgs() ([]string, error) {
	if !s.UseDocker {
		return s.solcFlags(), nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	args := []string{
		"run", "-i", "--rm", "-v", dir + ":/workdir", "-w", "/workdir", "ethereum/solc:" + s.Version,
	}
	args = append(args, s.solcFlags()...)
	return args, nil
}

// SolidityVersion extracts a concrete Solidity compiler version from pragma solidity in source.
func SolidityVersion(source string) (*Solidity, error) {
	matches := pragmaRegexp.FindStringSubmatch(source)
	if len(matches) < 2 {
		return nil, errors.New("can't find pragma solidity in source")
	}
	constraint := strings.TrimSpace(matches[1])

	allMatches := semverRegexp.FindAllStringSubmatch(constraint, -1)
	if len(allMatches) == 0 {
		return nil, fmt.Errorf("invalid pragma solidity version: %q", constraint)
	}

	// Pick the first lower-bound / caret version; if range like >=0.7.0 <0.9.0, prefer 0.8.24 if allowed.
	m := allMatches[0]
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch := 0
	if m[3] != "" {
		patch, _ = strconv.Atoi(m[3])
	} else if major == 0 && minor == 8 {
		patch = 24
	}
	if len(allMatches) >= 2 && strings.Contains(constraint, "<0.9") && major == 0 && minor < 8 {
		minor = 8
		patch = 24
	}

	ver := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	return &Solidity{
		Path:      "docker",
		UseDocker: true,
		Version:   ver,
		Major:     major,
		Minor:     minor,
		Patch:     patch,
	}, nil
}

// CompileSolidityString compiles Solidity source using a local solc binary if compatible or Docker ethereum/solc.
func CompileSolidityString(ctx context.Context, source, solcVersion, evmVersion string, optimize bool) (map[string]*Contract, error) {
	if len(source) == 0 {
		return nil, errors.New("solc: empty source string")
	}
	var s *Solidity
	var err error
	if solcVersion != "" {
		s = &Solidity{Path: "docker", UseDocker: true, Version: solcVersion}
	} else {
		s, err = SolidityVersion(source)
		if err != nil {
			return nil, err
		}
	}
	s.EVMVersion = evmVersion
	s.Optimize = optimize

	// Prefer native solc if installed and matches major.minor when solcVersion wasn't pinned to a different version.
	if solcPath, lookErr := exec.LookPath("solc"); lookErr == nil {
		if verOut, verErr := exec.CommandContext(ctx, solcPath, "--version").Output(); verErr == nil {
			if strings.Contains(string(verOut), s.Version) || (solcVersion == "" && strings.Contains(string(verOut), fmt.Sprintf("%d.%d.", s.Major, s.Minor))) {
				s.Path = solcPath
				s.UseDocker = false
			}
		}
	}

	args, err := s.makeArgs()
	if err != nil {
		return nil, err
	}
	args = append(args, "--", "-")
	cmd := exec.CommandContext(ctx, s.Path, args...)
	cmd.Stdin = strings.NewReader(source)
	return s.run(cmd, source)
}

func (s *Solidity) run(cmd *exec.Cmd, source string) (map[string]*Contract, error) {
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("solc: %w\n%s", err, stderr.String())
	}
	return ParseCombinedJSON(stdout.Bytes(), source, s.Version, s.Version, strings.Join(s.solcFlags(), " "))
}

func ParseCombinedJSON(combinedJSON []byte, source, languageVersion, compilerVersion, compilerOptions string) (map[string]*Contract, error) {
	var output solcOutput
	if err := json.Unmarshal(combinedJSON, &output); err != nil {
		return parseCombinedJSONV8(combinedJSON, source, languageVersion, compilerVersion, compilerOptions)
	}

	contracts := make(map[string]*Contract)
	for name, info := range output.Contracts {
		var abiDef interface{}
		if err := json.Unmarshal([]byte(info.Abi), &abiDef); err != nil {
			return nil, fmt.Errorf("solc: error reading abi definition (%w)", err)
		}
		var userdoc interface{}
		if info.Userdoc != "" {
			_ = json.Unmarshal([]byte(info.Userdoc), &userdoc)
		}
		var devdoc interface{}
		if info.Devdoc != "" {
			_ = json.Unmarshal([]byte(info.Devdoc), &devdoc)
		}
		contracts[name] = &Contract{
			Code:        "0x" + info.Bin,
			RuntimeCode: "0x" + info.BinRuntime,
			Info: ContractInfo{
				Source:          source,
				Language:        "Solidity",
				LanguageVersion: languageVersion,
				CompilerVersion: compilerVersion,
				CompilerOptions: compilerOptions,
				SrcMap:          info.SrcMap,
				SrcMapRuntime:   info.SrcMapRuntime,
				AbiDefinition:   abiDef,
				UserDoc:         userdoc,
				DeveloperDoc:    devdoc,
				Metadata:        info.Metadata,
			},
		}
	}
	return contracts, nil
}

func parseCombinedJSONV8(combinedJSON []byte, source, languageVersion, compilerVersion, compilerOptions string) (map[string]*Contract, error) {
	var output solcOutputV8
	if err := json.Unmarshal(combinedJSON, &output); err != nil {
		return nil, err
	}
	contracts := make(map[string]*Contract)
	for name, info := range output.Contracts {
		contracts[name] = &Contract{
			Code:        "0x" + info.Bin,
			RuntimeCode: "0x" + info.BinRuntime,
			Hashes:      info.Hashes,
			Info: ContractInfo{
				Source:          source,
				Language:        "Solidity",
				LanguageVersion: languageVersion,
				CompilerVersion: compilerVersion,
				CompilerOptions: compilerOptions,
				SrcMap:          info.SrcMap,
				SrcMapRuntime:   info.SrcMapRuntime,
				AbiDefinition:   info.Abi,
				UserDoc:         info.Userdoc,
				DeveloperDoc:    info.Devdoc,
				Metadata:        info.Metadata,
			},
		}
	}
	return contracts, nil
}
