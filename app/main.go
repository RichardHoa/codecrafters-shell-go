package main

import (
	"fmt"
)

func main() {
	var input string
	fmt.Print("$ ")
	fmt.Scanln(&input)
	fmt.Printf("%s: command not found\n", input)

}
