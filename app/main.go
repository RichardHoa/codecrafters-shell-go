package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	isBreak := false
	for {
		fmt.Print("$ ")

		scanner := bufio.NewScanner(os.Stdin)

		scanner.Scan()

		err := scanner.Err()
		if err != nil {
			fmt.Printf("Fatal error: %v\n", err)
		}

		input := scanner.Text()

		params := strings.Split(input, " ")

		switch params[0] {
		case "exit":
			isBreak = true
		case "echo":
			theRestOfCommand := input[len(params[0])+1:] // the last 1 is for the space!
			fmt.Println(theRestOfCommand)
		default:
			fmt.Printf("%s: command not found\n", input)
		}

		if isBreak {
			break
		}

	}

}
