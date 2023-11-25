package vyper

import (
	"fmt"
	"os/exec"
)

// TODO
func InstallVyper(contractName string) {
	cmd := exec.Command("pip", "install", "vyper")
	output, err := cmd.Output()
	fmt.Println(output)
	if err != nil {
		fmt.Println("*Web3.go Vyper Engine Error* Unable to install vyper compilier")
	}
}

// TODO
// NOTE this function does not work, it should call the actual local compiler function
// located in /vyper-go
func SubProcessCompilerLocal(fileName string) {
	cmd := exec.Command("python3", fileName)
	output, err := cmd.Output()
	fmt.Println(output)
	if err != nil {
		// add error handling file
	}
}

func DetectVyper(langOption string) bool {
	fmt.Println(langOption)
	return true
}

func VyperVersion() string {
	cmd := exec.Command("vyper", "--version")
	output, err := cmd.Output()
	if err != nil {

	}
	return ConvertByteArray(output)

}

func Compile(contractNamePath string) string {
	cmd := exec.Command("vyper", contractNamePath)
	output, err := cmd.Output()
	if err != nil {

	}

	return ConvertByteArray(output)
}

// TODO add switch statement
// Current Options:
// abi, bytecode, ir, asm, source_map
func CompileWithOptions(contractNamePath string, option string) {
	cmd := exec.Command("vyper", "-f", option, contractNamePath)
	output, err := cmd.Output()
	if err != nil {

	}
	fmt.Println(ConvertByteArray(output))
}

var localVyperVersion string = VyperVersion()

func CompileFromCLI(filepath string, option string) {
	fmt.Println("Vyper Compiler v", localVyperVersion)
	fmt.Println("Building", filepath, "\t", "with output options")
	CompileWithOptions(filepath, option)

}

//func main() {

//	fmt.Println("test")
//	fmt.Println("mod")
//	subProcessCompile()

//}
