package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/chzyer/readline"
)

var redirectionsOperators = []string{">", "1>", ">>", "1>>", "2>", "2>>"}
var builtinTools = []string{"type", "exit", "echo", "pwd"}

func main() {

	completer := NewCommandCompleter()

	statefulComplter := CustomCompleter{
		inner: completer,
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "$ ",
		AutoComplete: &statefulComplter,
		Listener:     &BellListener{completer: &statefulComplter},
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

		// goes to the next line
		fmt.Print("\r")

		args := SplitArgs(line)

		if len(args) == 0 {
			printErr("There must be a command\n")
			continue
		}

		// remove the space before the first command
		if strings.TrimSpace(args[0]) == "" {
			args = args[1:]
		}

		noSpaceArgs := filterEmptyArgs(args)

		command := args[0]

		pipeIndex := slices.Index(args, "|")
		if pipeIndex != -1 {
			handlePipe(args, pipeIndex)
			continue
		}

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
func handlePipe(args []string, pipeIndex int) {
	noSpaceArgs := filterEmptyArgs(args)
	redirectionTargets := findRedirectionTargets(noSpaceArgs)
	initializeRedirections(redirectionTargets)

	firstCommandSection := args[:pipeIndex]
	secondCommandSection := args[pipeIndex+1:]

	firstCommand := firstCommandSection[0]
	_, err := exec.LookPath(firstCommand)
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("%s: not found\r\n", firstCommand)),
			redirectionTargets,
			true,
		)
		return
	}

	if (strings.TrimSpace(secondCommandSection[0])) == "" {
		secondCommandSection = secondCommandSection[1:]
	}

	secondCommand := secondCommandSection[0]
	_, err = exec.LookPath(secondCommand)
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("%s: not found\r\n", firstCommand)),
			redirectionTargets,
			true,
		)
		return
	}

	firstCommandArgs := filterAndJoinArgs(firstCommandSection[1:])
	secondCommandArgs := filterAndJoinArgs(secondCommandSection[1:])

	firstCmd := exec.Command(firstCommand, firstCommandArgs...)
	secondCmd := exec.Command(secondCommand, secondCommandArgs...)

	pipe, _ := firstCmd.StdoutPipe()
	secondCmd.Stdin = pipe
	secondCmd.Stderr = os.Stderr
	secondCmd.Stdout = os.Stdout

	// Capture second command output
	// secondStdout, _ := secondCmd.StdoutPipe()
	// secondStderr, _ := secondCmd.StderrPipe()

	// Start both
	firstCmd.Start()
	secondCmd.Start()

	// var wg sync.WaitGroup
	// wg.Add(3)
	//
	// go func() {
	// 	defer wg.Done()
	// 	outputStream(firstStderr, redirectionTargets, true)
	// }()
	// go func() {
	// 	defer wg.Done()
	// 	outputStream(secondStdout, redirectionTargets, false)
	// }()
	// go func() {
	// 	defer wg.Done()
	// 	outputStream(secondStderr, redirectionTargets, true)
	// }()

	// wg.Wait()
	firstCmd.Wait()
	secondCmd.Wait()

}

func handleCD(noSpaceArgs []string) {
	redirectionTargets := findRedirectionTargets(noSpaceArgs)
	initializeRedirections(redirectionTargets)

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
			outputStream(
				strings.NewReader(fmt.Sprintf("Cannot change to home directory: %v", err)),
				redirectionTargets,
				true,
			)
		}
		return
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			outputStream(
				strings.NewReader(fmt.Sprintf("cd: %s: No such file or directory\r\n", path)),
				redirectionTargets,
				true,
			)
		} else {
			outputStream(
				strings.NewReader(fmt.Sprintf("Unexpected error when checking path status: %v", err)),
				redirectionTargets,
				true,
			)
		}
		return
	}

	absPath, err := absolutePath(path)
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("Error while join file path: %v\r\n", err)),
			redirectionTargets,
			true,
		)
	}

	err = os.Chdir(absPath)
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("Cannot change to specified location although it exists: %v\r\n", err)),
			redirectionTargets,
			true,
		)
	}
}

func handlePWD(noSpaceArgs []string) {
	redirectTargets := findRedirectionTargets(noSpaceArgs)
	initializeRedirections(redirectTargets)

	currentDir, err := filepath.Abs("./")
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("Cannot find current directory path: %v", err)),
			redirectTargets,
			true,
		)
		return
	}

	outputStream(
		strings.NewReader(fmt.Sprintln(currentDir)),
		redirectTargets,
		false,
	)

}
func handleType(noSpaceArgs []string) {
	redirectionTargets := findRedirectionTargets(noSpaceArgs)
	initializeRedirections(redirectionTargets)

	if len(noSpaceArgs) <= 1 {
		outputStream(
			strings.NewReader("lacking agrument: type [tool]\n"),
			redirectionTargets,
			true,
		)
		return
	}

	toolName := noSpaceArgs[1]

	if slices.Contains(builtinTools, toolName) {
		outputStream(
			strings.NewReader(fmt.Sprintf("%s is a shell builtin\r\n", toolName)),
			redirectionTargets,
			false,
		)
		return
	}

	toolAbsPath, err := exec.LookPath(toolName)
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("%s: not found\r\n", toolName)),
			redirectionTargets,
			true,
		)
		return
	}

	outputStream(
		strings.NewReader(fmt.Sprintf("%s is %s\r\n", toolName, toolAbsPath)),
		redirectionTargets,
		false,
	)
}

func handleExit() {
	os.Exit(0)
}

func handleEcho(args []string) {
	noSpaceArgs := filterEmptyArgs(args)
	redirectionTargets := findRedirectionTargets(noSpaceArgs)
	initializeRedirections(redirectionTargets)

	var output []string

	if len(args) <= 1 {
		outputStream(
			strings.NewReader("Lacking agrument: echo [something to echo]\n"),
			redirectionTargets,
			true,
		)
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

	outputStream(
		strings.NewReader(fmt.Sprintf("%s\r\n", strings.Join(output, ""))),
		redirectionTargets,
		false,
	)

}

func handleDefault(args []string) {
	command := strings.TrimSpace(args[0])
	cleanedArgs := filterAndJoinArgs(args[1:])

	noSpaceArgs := filterEmptyArgs(args)
	redirectionTargets := findRedirectionTargets(noSpaceArgs)

	initializeRedirections(redirectionTargets)

	_, err := exec.LookPath(command)
	if err != nil {
		outputStream(
			strings.NewReader(fmt.Sprintf("%s: not found\n", command)),
			redirectionTargets,
			true,
		)
		return
	}

	cmd := exec.Command(command, cleanedArgs...)

	outPipe, _ := cmd.StdoutPipe()
	errPipe, _ := cmd.StderrPipe()

	defer func() {
		outPipe.Close()
		errPipe.Close()
	}()

	_ = cmd.Start()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		outputStream(outPipe, redirectionTargets, false)
	}()

	go func() {
		defer wg.Done()
		outputStream(errPipe, redirectionTargets, true)
	}()

	wg.Wait()

	_ = cmd.Wait()

}

func outputStream(src io.Reader, targets redirectionTargets, isError bool) {
	var fallback io.Writer
	var appendPath, redirectPath string

	if isError {
		fallback = os.Stderr
		appendPath = targets.errAppend
		redirectPath = targets.errRedirect
	} else {
		fallback = os.Stdout
		appendPath = targets.outputAppend
		redirectPath = targets.outputRedirect
	}

	writers, closers, err := getWriters(redirectPath, appendPath, fallback)
	if err != nil {
		printErr(fmt.Sprintf("Output error: %v\n", err))
		return
	}
	// Ensure every file we opened gets closed
	defer func() {
		for _, closeFn := range closers {
			closeFn()
		}
	}()

	// Bundle all writers into one "super writer"
	multiDest := io.MultiWriter(writers...)

	// Stream the data to EVERY destination at the same time
	io.Copy(multiDest, src)
}

func getWriters(redirectPath string, appendPath string, fallback io.Writer) ([]io.Writer, []func(), error) {
	var writers []io.Writer
	var closers []func()

	// 1. Check for Truncated Redirection (>)
	if redirectPath != "" {
		f, err := os.OpenFile(redirectPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, nil, err
		}
		writers = append(writers, f)
		closers = append(closers, func() { f.Close() })
	}

	// 2. Check for Append Redirection (>>)
	// Note: Per your logic, we allow both!
	if appendPath != "" {
		f, err := os.OpenFile(appendPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			// Clean up already opened files if this one fails
			for _, c := range closers {
				c()
			}
			return nil, nil, err
		}
		writers = append(writers, f)
		closers = append(closers, func() { f.Close() })
	}

	// 3. If no files were opened, use the fallback (Stdout/Stderr)
	if len(writers) == 0 {
		writers = append(writers, fallback)
	}

	return writers, closers, nil
}

func initializeRedirections(input redirectionTargets) {

	if input.errAppend != "" {
		writeToFile(input.errAppend, "", true)
	}

	if input.errRedirect != "" {
		writeToFile(input.errRedirect, "", false)
	}

	if input.outputAppend != "" {
		writeToFile(input.outputAppend, "", true)
	}

	if input.outputRedirect != "" {
		writeToFile(input.outputRedirect, "", false)
	}

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

		// encounter a character outside of quote
		case !unicode.IsSpace(char):

			if buffer.Len() > 0 && isSpaceOnly {
				output = append(output, buffer.String())
				buffer.Reset()
			}

			if char == '|' {
				output = append(output, "|")
				continue
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

func findLCP(matches [][]rune) []rune {
	if len(matches) == 0 {
		return nil
	}
	// we assume the first word is the LCP (Longest Common Prefix)
	firstWord := matches[0]

	// loop through every char of the first word
	for i, charToMatch := range firstWord {
		// check for other matches to see if their ith char matches the ith char of the first word
		for j := 1; j < len(matches); j++ {

			// if not, then we know the LCP stop here
			if i >= len(matches[j]) || matches[j][i] != charToMatch {
				return firstWord[:i]
			}
		}
	}

	return firstWord
}

type CustomCompleter struct {
	inner    *readline.PrefixCompleter
	tabCount int
}

func (c *CustomCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	matches, length := c.inner.Do(line, pos)

	if len(matches) == 0 {
		fmt.Print("\x07")
		c.tabCount = 0
		return nil, 0
	}

	if len(matches) == 1 {
		c.tabCount = 0
		return [][]rune{matches[0]}, length
	}

	lcp := findLCP(matches)

	if len(lcp) > 0 {
		c.tabCount = 0
		return [][]rune{lcp}, length
	}

	if len(matches) > 1 {
		c.tabCount++

		if c.tabCount == 1 {
			fmt.Print("\x07")
			return nil, 0
		}

		prefix := string(line[:pos])
		var suggestions []string
		for _, m := range matches {
			suggestions = append(suggestions, prefix+string(m))
		}

		sort.Strings(suggestions)

		fmt.Printf("\n%s\n", strings.Join(suggestions, " "))
		fmt.Printf("$ %s", string(line))

		c.tabCount = 0
		return nil, 0
	}

	return nil, 0
}

type BellListener struct {
	completer *CustomCompleter
}

func (b *BellListener) OnChange(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
	if key == '\t' {
		lineSoFar := line[:pos]

		matches, _ := b.completer.Do(lineSoFar, len(lineSoFar))

		if len(matches) == 0 {
			fmt.Print("\x07")
		}
	} else {
		b.completer.tabCount = 0
	}
	return nil, 0, false
}

func NewCommandCompleter() (completer *readline.PrefixCompleter) {
	commandSet := make(map[string]struct{})

	for _, tool := range builtinTools {
		commandSet[tool] = struct{}{}
	}

	pathEnv := os.Getenv("PATH")
	paths := filepath.SplitList(pathEnv)

	for _, path := range paths {
		dir, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		for _, file := range dir {
			if file.IsDir() {
				continue
			}

			_, err := exec.LookPath(file.Name())
			if err != nil {
				continue
			}

			commandSet[file.Name()] = struct{}{}

		}
	}

	var items []readline.PrefixCompleterInterface
	for command := range commandSet {
		items = append(items, readline.PcItem(command))
	}

	completer = readline.NewPrefixCompleter(items...)

	return completer

}
