package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sshifu <sshifu-server> [ssh arguments]")
		os.Exit(1)
	}

	fmt.Println("sshifu CLI - SSH Certificate Authentication Client")
	fmt.Println("This is a placeholder. Implementation in progress.")
}
