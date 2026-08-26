package extractor

// This file owns the input boundary for selector command text.

import (
	"bufio"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Command represents a single extraction command.
type Command struct {
	Type    string // FILE, PREFIX, NEAR, SYMBOL, REFERENCES, TESTS, SEARCH
	Path    string // file path for file/span selectors; literal for discovery selectors
	Pattern string
}

// CommandParseResult preserves valid literal selectors while reporting
// malformed non-empty input lines separately.
type CommandParseResult struct {
	Commands []Command
	Failures []string
}

func (e *Extractor) ParseCommands(input string) []Command {
	return e.parseCommands(input, false).Commands
}

func (e *Extractor) ParseCommandsDetailed(input string) CommandParseResult {
	return e.parseCommands(input, true)
}

// ParseStrictCommandsDetailed accepts only one complete selector per non-empty
// line. Review continuation uses this boundary so ordinary findings are never
// mistaken for extraction requests merely because they mention selector text.
func (e *Extractor) ParseStrictCommandsDetailed(input string) CommandParseResult {
	var result CommandParseResult
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Buffer(make([]byte, 64*1024), len(input)+1)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		cmd, ok := parseCommandLine(line)
		if !ok {
			result.Failures = append(result.Failures, fmt.Sprintf("line %d: not a complete supported selector", lineNumber))
			continue
		}
		if err := validateCommand(cmd); err != nil {
			result.Failures = append(result.Failures, fmt.Sprintf("line %d: %v", lineNumber, err))
			continue
		}
		result.Commands = append(result.Commands, cmd)
	}
	if err := scanner.Err(); err != nil {
		result.Failures = append(result.Failures, fmt.Sprintf("reading selector response: %v", err))
	}
	return result
}

func (e *Extractor) parseCommands(input string, reportMalformed bool) CommandParseResult {
	var result CommandParseResult
	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if shouldRecoverEmbeddedFiles(line) {
			result.Commands = append(result.Commands, parseEmbeddedFileCommands(line)...)
			continue
		}
		if cmd, ok := parseCommandLine(line); ok {
			if err := validateCommand(cmd); err != nil {
				if reportMalformed {
					result.Failures = append(result.Failures, fmt.Sprintf("line %d: %v", lineNumber, err))
				}
				continue
			}
			result.Commands = append(result.Commands, cmd)
			continue
		}
		embedded := parseEmbeddedFileCommands(line)
		if len(embedded) > 0 {
			result.Commands = append(result.Commands, embedded...)
			continue
		}
		if reportMalformed {
			result.Failures = append(result.Failures, fmt.Sprintf("line %d: invalid or unsupported selector: %s", lineNumber, strings.TrimSpace(line)))
		}
	}
	return result
}

func validateCommand(cmd Command) error {
	switch cmd.Type {
	case "PREFIX", "NEAR", "SYMBOL":
		if cmd.Pattern == "" {
			return fmt.Errorf("%s requires path#pattern", cmd.Type)
		}
	case "REFERENCES", "TESTS", "SEARCH":
		if strings.TrimSpace(cmd.Path) == "" {
			return fmt.Errorf("%s requires a non-empty literal", cmd.Type)
		}
	}
	return nil
}

func parseCommandLine(line string) (Command, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Command{}, false
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return Command{}, false
	}
	cmdType := strings.ToUpper(strings.TrimSpace(parts[0]))
	if !isSupportedCommandType(cmdType) {
		return Command{}, false
	}
	value := strings.TrimSpace(parts[1])
	if value == "" {
		return Command{}, false
	}
	cmd := Command{Type: cmdType}
	if cmdType == "REFERENCES" || cmdType == "TESTS" || cmdType == "SEARCH" {
		cmd.Path = strings.TrimSpace(strings.Trim(value, "\""))
		return cmd, cmd.Path != ""
	}
	pathAndPattern := strings.SplitN(value, "#", 2)
	cmd.Path = strings.TrimSpace(pathAndPattern[0])
	if len(pathAndPattern) > 1 {
		cmd.Pattern = strings.TrimSpace(pathAndPattern[1])
	}
	if cmd.Path == "" {
		return Command{}, false
	}
	return cmd, true
}

func isSupportedCommandType(cmdType string) bool {
	switch cmdType {
	case "FILE", "PREFIX", "NEAR", "SYMBOL", "REFERENCES", "TESTS", "SEARCH":
		return true
	default:
		return false
	}
}

func shouldRecoverEmbeddedFiles(line string) bool {
	trimmed := strings.TrimSpace(line)
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) < 2 {
		return false
	}
	cmdType := strings.ToUpper(parts[0])
	if cmdType == "PREFIX" || cmdType == "NEAR" || cmdType == "SYMBOL" || cmdType == "REFERENCES" || cmdType == "TESTS" || cmdType == "SEARCH" {
		return false
	}
	return len(fileTokenIndexes(line)) > 1
}

func parseEmbeddedFileCommands(line string) []Command {
	indexes := fileTokenIndexes(line)
	commands := make([]Command, 0, len(indexes))
	for i, idx := range indexes {
		start := idx + len("FILE:")
		end := len(line)
		if i+1 < len(indexes) {
			end = indexes[i+1]
		}
		path := strings.TrimSpace(line[start:end])
		path = strings.TrimRight(path, " \t\r\n.,;:)]}")
		if path != "" {
			commands = append(commands, Command{Type: "FILE", Path: path})
		}
	}
	return commands
}

func fileTokenIndexes(line string) []int {
	upper := strings.ToUpper(line)
	var indexes []int
	for searchFrom := 0; searchFrom < len(upper); {
		idx := strings.Index(upper[searchFrom:], "FILE:")
		if idx < 0 {
			break
		}
		idx += searchFrom
		if hasFileTokenBoundary(line, idx) {
			indexes = append(indexes, idx)
		}
		searchFrom = idx + len("FILE:")
	}
	return indexes
}

func hasFileTokenBoundary(line string, idx int) bool {
	if idx == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(line[:idx])
	return !(unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '_')
}
