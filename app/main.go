package main

import (
	"bytes"
	"fmt"
	"github.com/chzyer/readline"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

var redirectionsOperators = []string{">", "1>", ">>", "1>>", "2>", "2>>"}
var builtinTools = []string{"type", "exit", "echo", "pwd"}


func main() {

	var items []readline.PrefixCompleterInterface
	for _, tool := range builtinTools {
		items = append(items, readline.PcItem(tool))
	}
	completer := readline.NewPrefixCompleter(items...)

	// 2. Initialize the Readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "$ ",
		AutoComplete: completer,
	})

	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {

		line, err := rl.Readline()
		if err != nil {
			break
		}

		fmt.Print("\r")

		args := SplitArgs(line)

		if len(args) == 0 {
			printErr("There must be a command\r\n")
			continue
		}

		// remove the space before the first command
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

//	func getCommandAutoComplete(args []string) (command string) {
//		if len(args) == 1 {
//			return args[0]
//		}
//
//		isTab := args[1] == "\t"
//		debug(args)
//		debug(fmt.Sprintf("isTab: %v", isTab))
//		if !isTab {
//			return args[0]
//		}
//
//		isAutoComplete := false
//		prefixCommand := args[0]
//		for i := 0; i < len(builtinTools); i++ {
//			if strings.HasPrefix(builtinTools[i], prefixCommand) {
//				command = builtinTools[i]
//				isAutoComplete = true
//			}
//		}
//
//		return command
//	}

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
				fmt.Sprintf("cd: %s: No such file or directory\r\n", path),
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
			fmt.Sprintf("Error while join file path: %v\r\n", err),
			redirectionTargets,
		)
	}

	err = os.Chdir(absPath)
	if err != nil {
		outputError(
			fmt.Sprintf("Cannot change to specified location although it exists: %v\r\n", err),
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

	if len(noSpaceArgs) <= 1 {
		outputError("lacking agrument: type [tool]\r\n", redirectionTargets)
		return
	}

	toolName := noSpaceArgs[1]

	if slices.Contains(builtinTools, toolName) {
		outputSuccess(
			fmt.Sprintf("%s is a shell builtin\r\n", toolName),
			redirectionTargets,
		)
		return
	}

	toolAbsPath, err := exec.LookPath(toolName)
	if err != nil {
		outputError(
			fmt.Sprintf("%s: not found\r\n", toolName),
			redirectionTargets,
		)
		return
	}

	outputSuccess(
		fmt.Sprintf("%s is %s\r\n", toolName, toolAbsPath),
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

	if len(args) <= 1 {
		outputError("Lacking agrument: echo [something to echo]\r\n", redirectionTargets)
		return
	}

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
		fmt.Sprintf("%s\r\n", strings.Join(output, "")),
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
			fmt.Sprintf("%s: not found\r\n", command),
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
	fmt.Printf("DEBUGGING: |%#v|\r\n", input)
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
