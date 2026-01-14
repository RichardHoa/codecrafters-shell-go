package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

func main() {
	validCommands := []string{"type", "exit", "echo"}

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

		// fmt.Printf("DEBUGGING: |%v|\n", params)

		command := params[0]

		switch command {
		case "type":
			if slices.Contains(validCommands, params[1]) {
				fmt.Printf("%s is a shell builtin\n", params[1])
			} else {
				fmt.Printf("%s: not found\n", params[1])
			}
		case "exit":
			isBreak = true
		case "echo":
			restOfTheCommand := input[len(params[0])+1:] // the last 1 is for the space!
			fmt.Println(restOfTheCommand)
		default:
			fmt.Printf("%s: command not found\n", input)
		}

		if isBreak {
			break
		}

	}

}
