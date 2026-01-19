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

		args := SplitByContext(input)

		debug(args)
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

func deleteExtraWhiteSpace(inputSlice []string) []string {
	for i := 1; i < len(inputSlice)-1; i++ {
		current := inputSlice[i]
		before := inputSlice[i-1]
		after := inputSlice[i+1]

		if before != "" && after != "" && isPureWhitespace(current) && len(current) >= 2 {
			inputSlice[i] = " "
		}
	}

	return inputSlice
}

func handleEcho(input string) {
	var output string
	args := strings.SplitAfterN(input, " ", 2)

	restOfTheCommand := args[1]

	slice := SplitByContext(restOfTheCommand)
	debug(slice)

	printToConsole(
		fmt.Sprintln(output),
	)
}

func SplitByContext(input string) []string {
	var result []string
	var buffer string
	var activeQuote rune // Stores the quote character (' or ") currently being handled

	for _, char := range input {
		switch {
		// 1. HIGHEST PRIORITY: If we are already inside a quote block
		case activeQuote != 0:
			if char == activeQuote {
				result = append(result, buffer) // Save what's inside
				buffer = ""                     // Clear buffer
				activeQuote = 0                 // "Exit" quote mode
			} else {
				buffer += string(char)
			}

		// 2. TRIGGER: If we encounter a new opening quote
		case char == '"' || char == '\'':
			if buffer != "" {
				result = append(result, buffer)
			}
			buffer = ""        // Start fresh for the inside of the quote
			activeQuote = char // "Enter" quote mode

		// 3. WHITESPACE: If we are outside quotes and hit a space
		case unicode.IsSpace(char):
			// If the buffer currently holds text, save it before starting space segment
			if buffer != "" && !unicode.IsSpace(rune(buffer[0])) {
				result = append(result, buffer)
				buffer = ""
			}
			buffer += string(char)

		// 4. PLAIN TEXT: Normal characters outside of quotes
		default:
			// If the buffer currently holds spaces, save them before starting text segment
			if buffer != "" && unicode.IsSpace(rune(buffer[0])) {
				result = append(result, buffer)
				buffer = ""
			}
			buffer += string(char)
		}
	}

	// Catch any remaining content left in the buffer at the end
	if buffer != "" {
		result = append(result, buffer)
	}

	return result
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

func isPureWhitespace(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func printToConsole(input string) {
	_, err := fmt.Printf("%s", input)

	if err != nil {
		str := fmt.Sprintf("Error while printing value to the console: %s", err)
		panic(str)
	}

}
