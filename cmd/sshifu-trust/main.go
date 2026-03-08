package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sudo sshifu-trust <sshifu-server>")
		os.Exit(1)
	}

	fmt.Println("sshifu-trust - SSH Server Trust Configuration Tool")
	fmt.Println("This is a placeholder. Implementation in progress.")
}
