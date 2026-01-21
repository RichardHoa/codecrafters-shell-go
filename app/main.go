package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
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

		args := SplitArgs(input)
		noSpaceArgs := deleteSpace(args)

		command := strings.TrimSpace(args[0])

		switch command {
		case "cd":
			handleCD(noSpaceArgs)
		case "pwd":
			handlePWD()
		case "type":
			handleType(noSpaceArgs)
		case "exit":
			handleExit()
		case "echo":
			handleEcho(args)
		default:
			handleDefault(noSpaceArgs)
		}

	}

}
func handleCD(noSpaceArgs []string) {
	var path string
	argsLength := len(noSpaceArgs)

	switch argsLength {
	case 1:
		path = "~"
	case 2:
		path = noSpaceArgs[1]
	default:
		printErrorToConsole("Too many agrument, maximum of one agrument\n")
		return
	}

	if path == "~" {
		homePath := os.Getenv("HOME")

		err := os.Chdir(homePath)
		if err != nil {
			printErrorToConsole(
				fmt.Sprintf("Cannot change to home directory: %v", err),
			)
		}
		return
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			printErrorToConsole(
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
				printErrorToConsole(
					fmt.Sprintf("Cannot check current directory path: %v", err),
				)
			}

			joinedPath = filepath.Join(currentDirectory, path)
		}

		err = os.Chdir(joinedPath)
		if err != nil {
			printErrorToConsole(
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
func handleType(noSpaceArgs []string) {
	toolName := noSpaceArgs[1]
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

func handleEcho(args []string) {
	if strings.TrimSpace(args[1]) != "" {
		printErrorToConsole("wrong command format. echo command")
		return
	}

	var output []string

	restOfTheCommand := args[2:]

	emptySpace := " "
	for _, val := range restOfTheCommand {
		if val == "" {
			continue
		} else if strings.TrimSpace(val) == "" {
			output = append(output, emptySpace)
		} else {
			output = append(output, val)
		}
	}

	printToConsole(
		fmt.Sprintf("%s\n", strings.Join(output, "")),
	)

}

func SplitArgs(input string) (output []string) {
	var buffer strings.Builder
	var activeQuote rune
	var isSpaceOnly bool
	var isNextCharLiteral bool

	for _, char := range input {

		switch {

		case isNextCharLiteral:
			buffer.WriteRune(char)
			isNextCharLiteral = false

		case char == '\\':
			if buffer.Len() > 0 && isSpaceOnly {
				output = append(output, buffer.String())
				buffer.Reset()
			}

			isNextCharLiteral = true

		// if we are inside an active quote
		case activeQuote != 0:
			if char == activeQuote {
				output = append(output, buffer.String())
				buffer.Reset()
				activeQuote = 0
			} else {
				buffer.WriteRune(char)
			}

		// encounter an active quote for the first time
		case char == '"' || char == '\'':
			if buffer.Len() > 0 {
				output = append(output, buffer.String())
				buffer.Reset()
			}
			activeQuote = char

		// encounter a character
		case !unicode.IsSpace(char):
			if buffer.Len() > 0 && isSpaceOnly {
				output = append(output, buffer.String())
				buffer.Reset()
			}
			isSpaceOnly = false

			buffer.WriteRune(char)

		case unicode.IsSpace(char):
			if buffer.Len() > 0 && !isSpaceOnly {
				output = append(output, buffer.String())
				buffer.Reset()
			}
			isSpaceOnly = true

			buffer.WriteRune(char)
		}

	}

	if buffer.Len() > 0 {
		output = append(output, buffer.String())
		buffer.Reset()
	}

	return output

}

func handleDefault(args []string) {
	command := strings.TrimSpace(args[0])

	_, err := exec.LookPath(command)
	if err == nil {

		cmd := exec.Command(command, args[1:]...)

		output, err := cmd.Output()
		if err != nil {
			printErrorToConsole(
				fmt.Sprintf("Cannot execute command: %v\n", err),
			)
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

func deleteSpace(input []string) (output []string) {
	for _, val := range input {
		if strings.TrimSpace(val) != "" {
			output = append(output, val)
		}
	}

	return output
}

func debug(input any) {
	fmt.Printf("DEBUGGING: |%#v|\n", input)
}

func printToConsole(input string) {
	_, err := fmt.Fprintf(os.Stdout, "%s", input)

	if err != nil {
		str := fmt.Sprintf("Error while printing value to the console: %s", err)
		panic(str)
	}

}

func printErrorToConsole(input string) {
	_, err := fmt.Fprintf(os.Stderr, "%s", input)

	if err != nil {
		str := fmt.Sprintf("Error while printing value to the console: %s", err)
		panic(str)
	}

}
