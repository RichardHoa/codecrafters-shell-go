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

	var buffer strings.Builder
	var activeQuote rune
	var isSpaceOnly bool

	for _, char := range input {
		// if we are in a quote string
		if activeQuote != 0 {
			// if we reach the closing quote
			if char == activeQuote {
				// save the buffer to the result
				result = append(result, buffer.String())
				// reset it
				buffer.Reset()
				// set the active quote to nil (which is 0 in this case)
				activeQuote = 0
			} else {
				// if we do not reach the closing quote, keep adding rune
				buffer.WriteRune(char)
			}
			// because we already process the rune, we skip to the next one, no need to do further check
			continue
		}

		switch {
		// if we encounter the quote
		case char == '"' || char == '\'':
			// if we have anything before the quote, push it to the result
			if buffer.Len() > 0 {
				result = append(result, buffer.String())
				buffer.Reset()
			}
			// set the active quote to the char
			activeQuote = char

		case unicode.IsSpace(char):
			// if we have anything before the quote AND our string in buffer is not all space
			if buffer.Len() > 0 && !isSpaceOnly {
				// push the current string to the result and then reset it
				result = append(result, buffer.String())
				buffer.Reset()
			}
			// set the space to true
			isSpaceOnly = true
			// write to the buffer
			buffer.WriteRune(char)

		default:
			// this case is for normal character OUTSIDE of quote
			// if we encounter a character and the buffer contains all space
			if buffer.Len() > 0 && isSpaceOnly {
				// we add the space buffer to the result
				result = append(result, buffer.String())
				buffer.Reset()
			}
			// remember to set the space to false
			isSpaceOnly = false

			buffer.WriteRune(char)
		}
	}

	if buffer.Len() > 0 {
		result = append(result, buffer.String())
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
