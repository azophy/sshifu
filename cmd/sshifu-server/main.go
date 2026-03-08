package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("sshifu-server - SSH Certificate Authority and OAuth Gateway")
	fmt.Println("This is a placeholder. Implementation in progress.")
	
	if len(os.Args) > 1 {
		fmt.Printf("Arguments: %v\n", os.Args[1:])
	}
}
