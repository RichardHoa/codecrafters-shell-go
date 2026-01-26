package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

var redirectionsOperators = []string{">", "1>", "2>"}

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

		if len(args) == 0 {
			printErr("There must be a command\n")
			continue
		}

		if strings.TrimSpace(args[0]) == "" {
			args = args[1:]
		}

		noSpaceArgs := deleteSpaceArgs(args)

		command := args[0]

		switch command {
		case "cd":
			handleCD(noSpaceArgs)
		case "pwd":
			handlePWD(noSpaceArgs)
		case "type":
			handleType(noSpaceArgs)
		case "exit":
			handleExit()
		case "echo":
			handleEcho(args)
		default:
			handleDefault(args)
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
		outputError("Too many agrument, maximum of one agrument\n", noSpaceArgs)
		return
	}

	if path == "~" {
		homePath := os.Getenv("HOME")

		err := os.Chdir(homePath)
		if err != nil {
			outputError(
				fmt.Sprintf("Cannot change to home directory: %v", err),
				noSpaceArgs,
			)
		}
		return
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			outputError(
				fmt.Sprintf("cd: %s: No such file or directory\n", path),
				noSpaceArgs,
			)
		} else {
			outputError(
				fmt.Sprintf("Unexpected error when checking path status: %v", err),
				noSpaceArgs,
			)
		}
		return
	}

	joinedPath, err := joinPath(path)
	if err != nil {
		outputError(
			fmt.Sprintf("Error while join file path: %v\n", err),
			noSpaceArgs,
		)
	}

	err = os.Chdir(joinedPath)
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot change to specified location although it exists: %v\n", err),
			noSpaceArgs,
		)
	}

}

func handlePWD(noSpaceArgs []string) {
	currentDir, err := filepath.Abs("./")
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot find current directory path: %v", err),
			noSpaceArgs,
		)
		return
	}

	outputSuccess(
		fmt.Sprintln(currentDir),
		noSpaceArgs,
	)

}
func handleType(noSpaceArgs []string) {
	toolName := noSpaceArgs[1]
	validTools := []string{"type", "exit", "echo", "pwd"}

	if slices.Contains(validTools, toolName) {
		outputSuccess(
			fmt.Sprintf("%s is a shell builtin\n", toolName),
			noSpaceArgs,
		)
		return
	}

	toolAbsPath, err := exec.LookPath(toolName)
	if err != nil {
		outputError(
			fmt.Sprintf("%s: not found\n", toolName),
			noSpaceArgs,
		)
		return
	}

	outputSuccess(
		fmt.Sprintf("%s is %s\n", toolName, toolAbsPath),
		noSpaceArgs,
	)
}

func handleExit() {
	os.Exit(0)
}

func handleEcho(args []string) {
	noSpaceArgs := deleteSpaceArgs(args)

	var output []string

	restOfTheCommand := args[2:]
	breakLoop := false

	for _, val := range restOfTheCommand {
		switch {
		case slices.Contains(redirectionsOperators, val):
			breakLoop = true
		case val == "":
			continue
		case strings.TrimSpace(val) == "":
			output = append(output, " ")
		default:
			output = append(output, val)
		}

		if breakLoop {
			break
		}
	}

	outputError("", noSpaceArgs)

	outputSuccess(
		fmt.Sprintf("%s\n", strings.Join(output, "")),
		noSpaceArgs,
	)

}

func SplitArgs(input string) (output []string) {
	var buffer strings.Builder
	var activeQuote rune
	var isSpaceOnly bool
	var isNextCharLiteral bool

	for index, char := range input {

		switch {

		case isNextCharLiteral:
			buffer.WriteRune(char)
			isNextCharLiteral = false

		case char == '\\':

			switch activeQuote {
			// we treat everything as literal inside single quote
			case '\'':
				buffer.WriteRune(char)
			// inside double quote we only escape some special characters
			case '"':
				specialCharacters := []string{"\"", "\\"}

				var nextChar string
				if index+1 <= len(input) {
					nextChar = string(input[index+1])
				} else {
					nextChar = string(char)
				}

				// if the next char is one of the special character, we escape the next char
				if slices.Contains(specialCharacters, nextChar) {
					isNextCharLiteral = true
				} else {
					buffer.WriteRune(char)
				}
			// if we are not inside a quote string, we escape the next char
			case 0:
				isNextCharLiteral = true
			}

			if buffer.Len() > 0 && isSpaceOnly {
				output = append(output, buffer.String())
				buffer.Reset()
			}

		// if we are inside an active quote
		case activeQuote != 0:
			isSpaceOnly = false

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
	processArgs := processArgsForDefaultFunc(args[1:])
	noSpaceArgs := deleteSpaceArgs(args)

	var stdout, stderr bytes.Buffer
	_, err := exec.LookPath(command)
	if err == nil {

		cmd := exec.Command(command, processArgs...)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		_ = cmd.Run()

		outputError(
			stderr.String(),
			noSpaceArgs,
		)

		outputSuccess(
			stdout.String(),
			noSpaceArgs,
		)
	} else {
		outputError(
			fmt.Sprintf("%s: not found\n", command),
			noSpaceArgs,
		)
	}
}

func deleteSpaceArgs(input []string) (output []string) {
	for _, val := range input {
		if strings.TrimSpace(val) != "" {
			output = append(output, val)
		}
	}

	return output
}

func processArgsForDefaultFunc(rawArgs []string) (output []string) {
	var buffer strings.Builder

	for _, val := range rawArgs {
		isEmtpySpace := strings.TrimSpace(val) == ""

		if slices.Contains(redirectionsOperators, val) {
			break
		}

		if !isEmtpySpace {
			buffer.WriteString(val)
		} else {
			if buffer.Len() > 0 {
				output = append(output, buffer.String())
				buffer.Reset()
			}
		}

	}

	if buffer.Len() > 0 {
		output = append(output, buffer.String())
		buffer.Reset()
	}

	return output
}

func debug(input any) {
	fmt.Printf("DEBUGGING: |%#v|\n", input)
}

func printErr(errString string) {
	_, err := fmt.Fprintf(os.Stderr, "%s", errString)

	if err != nil {
		panic(fmt.Sprintf("Error while printing error to the console: %s", err))
	}

}

func outputError(errorOutput string, noSpaceArgs []string) {
	// debug("ERROR")
	filePath, err := findRedirectStderr(noSpaceArgs)
	if err != nil {
		printErr(err.Error())
		return
	}

	// debug(filePath)
	// debug(errorOutput)

	if filePath == "" {
		printErr(errorOutput)

		return
	}

	joinedPath, err := joinPath(filePath)
	if err != nil {
		printErr(fmt.Sprintf("Cannot join path: %v", err))
	}

	file, err := os.Create(joinedPath)
	if err != nil {
		printErr(fmt.Sprintf("Cannot create error file: %v", err))
	}

	defer file.Close()

	_, err = file.WriteString(errorOutput)
	if err != nil {
		printErr(fmt.Sprintf("Unable to write content to file: %v", err))
	}

}

func outputSuccess(successOutput string, noSpaceArgs []string) {
	// debug("SUCESS")

	filePath, err := findRedirectStdout(noSpaceArgs)
	if err != nil {
		outputError(err.Error(), noSpaceArgs)
		return
	}

	// debug(filePath)
	// debug(successOutput)

	if filePath == "" {
		_, err := fmt.Fprintf(os.Stdout, "%s", successOutput)

		if err != nil {
			str := fmt.Sprintf("Error while printing value to the console: %s", err)
			outputError(str, noSpaceArgs)
		}

		return
	}

	joinedPath, err := joinPath(filePath)
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot join path: %v", err),
			noSpaceArgs,
		)
		return
	}

	file, err := os.Create(joinedPath)
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot create sucess file: %v", err),
			noSpaceArgs,
		)
		return
	}

	defer file.Close()

	_, err = file.WriteString(successOutput)
	if err != nil {
		outputError(
			fmt.Sprintf("Unable to write content to file: %v", err),
			noSpaceArgs,
		)
	}

}

func findRedirectStderr(noSpaceArgs []string) (filePath string, err error) {
	counter := 0

	for index, val := range noSpaceArgs {
		if val == "2>" {
			counter++
			if index == len(noSpaceArgs)-1 {
				return "", errors.New("stderr sign but no file\n")
			}

			filePath = noSpaceArgs[index+1]

		}
	}

	if counter > 1 {
		return "", errors.New("Can only pipe to one file only\n")
	}

	return filePath, nil
}

func findRedirectStdout(noSpaceArgs []string) (filePath string, err error) {
	counter := 0

	for index, val := range noSpaceArgs {
		if val == ">" || val == "1>" {
			counter++
			if index == len(noSpaceArgs)-1 {
				return "", errors.New("Direct stdout sign but no file\n")
			}

			filePath = noSpaceArgs[index+1]

		}
	}

	if counter > 1 {
		return "", errors.New("Can only pipe to one file only\n")
	}

	return filePath, nil
}

func joinPath(filePath string) (joinedPath string, err error) {

	if filepath.IsAbs(filePath) {
		joinedPath = filepath.Join(filePath)
	} else {
		currentDirectory, err := filepath.Abs("./")
		if err != nil {
			return "", err
		}

		joinedPath = filepath.Join(currentDirectory, filePath)
	}

	return joinedPath, nil

}
