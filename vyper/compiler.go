package vyper

import (
	"fmt"
	"os/exec"
)
//TODO  
func subProcessCompilePIP(contractName string) {
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
func subProcessCompilerLocal(fileName string) {
cmd := exec.Command("python3", fileName)
output, err := cmd.Output()
fmt.Println(output)
if err != nil {
	// add error handling file
}
}



//func main() {
	
//	fmt.Println("test")
//	fmt.Println("mod")
//	subProcessCompile()

//}
