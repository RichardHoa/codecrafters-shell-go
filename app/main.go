package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

var redirectionsOperators = []string{">", "1>", ">>", "1>>", "2>", "2>>"}

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

		noSpaceArgs := filterEmptyArgs(args)

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
	redirectionTargets := findRedirectionTargets(noSpaceArgs)

	defer outputSuccess("", redirectionTargets)

	var path string

	if len(noSpaceArgs) == 1 {
		path = "~"
	} else {
		path = noSpaceArgs[1]
	}

	if path == "~" {
		homePath := os.Getenv("HOME")

		err := os.Chdir(homePath)
		if err != nil {
			outputError(
				fmt.Sprintf("Cannot change to home directory: %v", err),
				redirectionTargets,
			)
		}
		return
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			outputError(
				fmt.Sprintf("cd: %s: No such file or directory\n", path),
				redirectionTargets,
			)
		} else {
			outputError(
				fmt.Sprintf("Unexpected error when checking path status: %v", err),
				redirectionTargets,
			)
		}
		return
	}

	absPath, err := absolutePath(path)
	if err != nil {
		outputError(
			fmt.Sprintf("Error while join file path: %v\n", err),
			redirectionTargets,
		)
	}

	err = os.Chdir(absPath)
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot change to specified location although it exists: %v\n", err),
			redirectionTargets,
		)
	}

}

func handlePWD(noSpaceArgs []string) {
	redirectTargets := findRedirectionTargets(noSpaceArgs)

	currentDir, err := filepath.Abs("./")
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot find current directory path: %v", err),
			redirectTargets,
		)
		return
	}

	defer outputError("", redirectTargets)
	outputSuccess(
		fmt.Sprintln(currentDir),
		redirectTargets,
	)

}
func handleType(noSpaceArgs []string) {
	redirectionTargets := findRedirectionTargets(noSpaceArgs)
	defer outputError("", redirectionTargets)

	toolName := noSpaceArgs[1]
	validTools := []string{"type", "exit", "echo", "pwd"}

	if slices.Contains(validTools, toolName) {
		outputSuccess(
			fmt.Sprintf("%s is a shell builtin\n", toolName),
			redirectionTargets,
		)
		return
	}

	toolAbsPath, err := exec.LookPath(toolName)
	if err != nil {
		outputError(
			fmt.Sprintf("%s: not found\n", toolName),
			redirectionTargets,
		)
		return
	}

	outputSuccess(
		fmt.Sprintf("%s is %s\n", toolName, toolAbsPath),
		redirectionTargets,
	)
}

func handleExit() {
	os.Exit(0)
}

func handleEcho(args []string) {
	noSpaceArgs := filterEmptyArgs(args)
	redirectionTargets := findRedirectionTargets(noSpaceArgs)

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

	/*
		Handle stderr redirection. Even if echo doesn't fail,
		a command like 'echo text 2> error.log' must still create 'error.log'
	*/
	outputError("", redirectionTargets)

	outputSuccess(
		fmt.Sprintf("%s\n", strings.Join(output, "")),
		redirectionTargets,
	)

}

func handleDefault(args []string) {
	command := strings.TrimSpace(args[0])
	cleanedArgs := filterAndJoinArgs(args[1:])
	noSpaceArgs := filterEmptyArgs(args)
	redirectionTargets := findRedirectionTargets(noSpaceArgs)

	_, err := exec.LookPath(command)
	if err != nil {
		outputError(
			fmt.Sprintf("%s: not found\n", command),
			redirectionTargets,
		)
		return
	}

	var stdout, stderr bytes.Buffer

	//NOTE: Consider doing file expansion (*.txt) here
	cmd := exec.Command(command, cleanedArgs...)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	outputError(
		stderr.String(),
		redirectionTargets,
	)

	outputSuccess(
		stdout.String(),
		redirectionTargets,
	)
}

func filterEmptyArgs(input []string) (output []string) {
	for _, val := range input {
		if strings.TrimSpace(val) != "" {
			output = append(output, val)
		}
	}

	return output
}

func filterAndJoinArgs(rawArgs []string) (output []string) {
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

func outputError(errorOutput string, redirectTarget redirectionTargets) {
	redirectPath := redirectTarget.errRedirect
	appendPath := redirectTarget.errAppend

	if appendPath == "" && redirectPath == "" {
		printErr(errorOutput)
		return
	}

	writeToFile(redirectPath, errorOutput, false)

	writeToFile(appendPath, errorOutput, true)
}

func outputSuccess(successOutput string, redirectTarget redirectionTargets) {

	redirectPath := redirectTarget.outputRedirect
	appendPath := redirectTarget.outputAppend

	if redirectPath == "" && appendPath == "" {
		_, err := fmt.Fprintf(os.Stdout, "%s", successOutput)

		if err != nil {
			str := fmt.Sprintf("Error while printing sucess value to the console: %s", err)
			outputError(str, redirectTarget)
		}

		return
	}

	writeToFile(redirectPath, successOutput, false)

	writeToFile(appendPath, successOutput, true)

}

type redirectionTargets struct {
	outputRedirect string
	errRedirect    string
	outputAppend   string
	errAppend      string
}

func findRedirectionTargets(noSpaceArgs []string) redirectionTargets {
	var output redirectionTargets
	for i := 0; i < len(noSpaceArgs)-1; i++ {
		val := noSpaceArgs[i]
		switch val {
		case ">":
			fallthrough
		case "1>":
			output.outputRedirect = noSpaceArgs[i+1]
		case ">>":
			fallthrough
		case "1>>":
			output.outputAppend = noSpaceArgs[i+1]
		case "2>":
			output.errRedirect = noSpaceArgs[i+1]
		case "2>>":
			output.errAppend = noSpaceArgs[i+1]
		default:
			continue
		}
	}

	return output

}

func absolutePath(filePath string) (absPath string, err error) {

	if filepath.IsAbs(filePath) {
		absPath = filepath.Join(filePath)
	} else {
		currentDirectory, err := filepath.Abs("./")
		if err != nil {
			return "", err
		}

		absPath = filepath.Join(currentDirectory, filePath)
	}

	return absPath, nil

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

func writeToFile(path string, content string, isAppend bool) {
	if path == "" {
		return
	}

	absPath, err := absolutePath(path)
	if err != nil {
		printErr(fmt.Sprintf("Cannot resolve path: %v\n", err))
		return
	}

	flags := os.O_WRONLY | os.O_CREATE
	if isAppend {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(absPath, flags, 0644)
	if err != nil {
		printErr(fmt.Sprintf("Cannot open file: %v\n", err))
		return
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		printErr(fmt.Sprintf("Unable to write content: %v\n", err))
	}
}
