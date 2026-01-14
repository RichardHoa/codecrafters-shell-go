package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	fmt.Print("$ ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	err := scanner.Err()
	if err != nil {
		fmt.Printf("Fatal error: %v\n", err)
	}
	fmt.Printf("%s: command not found\n", scanner.Text())

}
