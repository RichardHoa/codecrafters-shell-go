package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		if strings.Contains(input, "'") {
			args = strings.Split(input, "'")
		}

		command := strings.TrimSpace(args[0])

		switch command {
		case "cd":
			handleCD(args)
		case "pwd":
			handlePWD()
		case "type":
			handleType(args)
		case "exit":
			handleExit()
		case "echo":
			handleEcho(input)
		default:
			handleDefault(args)
		}

	}

}
func handleCD(args []string) {
	var path string
	if len(args) > 1 {
		path = args[1]
	} else {
		path = "~"
	}

	if path == "~" {
		homePath := os.Getenv("HOME")

		err := os.Chdir(homePath)
		if err != nil {
			printToConsole(
				fmt.Sprintf("Cannot change to home directory: %v", err),
			)
		}
		return
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			printToConsole(
				fmt.Sprintf("cd: %s: No such file or directory\n", path),
			)
		} else {
			panic(fmt.Sprintf("Unexpected error: %v", err))
		}
	}

	if err == nil {
		var joinedPath string

		if filepath.IsAbs(path) {
			joinedPath = filepath.Join(path)
		} else {
			// the path can be a local path or a ../ path
			currentDirectory, err := filepath.Abs("./")
			if err != nil {
				printToConsole(
					fmt.Sprintf("Cannot check current directory path: %v", err),
				)
			}

			joinedPath = filepath.Join(currentDirectory, path)
		}

		err = os.Chdir(joinedPath)
		if err != nil {
			printToConsole(
				fmt.Sprintf("Cannot change to specified location although it exists: %v\n", err),
			)
		}

	}

}

func handlePWD() {
	dir, err := filepath.Abs("./")
	if err != nil {
		printToConsole(
			fmt.Sprintf("Cannot check current directory path: %v", err),
		)
	}

	printToConsole(
		fmt.Sprintln(dir),
	)

}
func handleType(args []string) {
	toolName := args[1]
	validTools := []string{"type", "exit", "echo", "pwd"}

	if slices.Contains(validTools, toolName) {
		printToConsole(
			fmt.Sprintf("%s is a shell builtin\n", toolName),
		)
	} else {
		absPath, err := exec.LookPath(toolName)
		if err == nil {
			printToConsole(
				fmt.Sprintf("%s is %s\n", toolName, absPath),
			)
		} else {
			printToConsole(
				fmt.Sprintf("%s: not found\n", toolName),
			)
		}
	}

}

func handleExit() {
	os.Exit(0)
}

func handleEcho(input string) {
	var output string
	args := strings.SplitAfterN(input, " ", 2)

	restOfTheCommand := args[1]

	singleQuoteCounts := strings.Count(restOfTheCommand, "'")

	// check for valid single quotes string
	if singleQuoteCounts > 0 && singleQuoteCounts%2 == 0 {
		singleQuoteParts := strings.Split(restOfTheCommand, "'")
		output = strings.Join(singleQuoteParts, "")
	} else {
		//remove all space
		outputFields := strings.Fields(restOfTheCommand)
		output = strings.Join(outputFields, " ")
	}

	printToConsole(
		fmt.Sprintln(output),
	)

}

func handleDefault(args []string) {
	command := strings.TrimSpace(args[0])

	_, err := exec.LookPath(command)
	if err == nil {
		var otherArgs []string

		for _, val := range args[1:] {
			if strings.TrimSpace(val) != "" {
				otherArgs = append(otherArgs, val)
			}
		}

		cmd := exec.Command(command, otherArgs...)

		output, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
		}

		printToConsole(
			fmt.Sprintf("%s", output),
		)
	} else {
		printToConsole(
			fmt.Sprintf("%s: not found\n", command),
		)
	}
}

func debug(input any) {
	fmt.Printf("DEBUGGING: |%#v|\n", input)
}

func printToConsole(input string) {
	_, err := fmt.Printf("%s", input)

	if err != nil {
		str := fmt.Sprintf("Error while printing value to the console: %s", err)
		panic(str)
	}

}
