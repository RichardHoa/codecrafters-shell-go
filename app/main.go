package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	_ "path/filepath"
	"slices"
	"strings"
)

func main() {

	for {
		fmt.Print("$ ")

		scanner := bufio.NewScanner(os.Stdin)

		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			fmt.Printf("Fatal error: %v\n", err)
		}

		input := scanner.Text()

		args := strings.Split(input, " ")

		command := args[0]

		switch command {
		case "type":
			handleType(args)
		case "exit":
			handleExit()
		case "echo":
			handleEcho(args)
		default:
			handleDefault(args)
		}

	}

}

func handleType(args []string) {
	toolName := args[1]
	validTools := []string{"type", "exit", "echo"}

	if slices.Contains(validTools, toolName) {
		fmt.Printf("%s is a shell builtin\n", toolName)
	} else {
		absPath, err := exec.LookPath(toolName)
		if err == nil {
			fmt.Printf("%s is %s\n", toolName, absPath)
		} else {
			fmt.Printf("%s: not found\n", toolName)
		}
	}

}

func handleExit() {
	os.Exit(0)
}

func handleEcho(args []string) {
	restOfTheCommand := strings.Join(args[1:], " ")
	fmt.Println(restOfTheCommand)

}

func handleDefault(args []string) {
	command := args[0]

	_, err := exec.LookPath(command)
	if err == nil {
		otherArgs := args[1:]

		cmd := exec.Command(command, otherArgs...)

		output, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("%s", output)
	} else {
		fmt.Printf("%s: not found\n", command)
	}
}

func debug(input any) {
	fmt.Printf("DEBUGGING: |%#v|\n", input)
}
