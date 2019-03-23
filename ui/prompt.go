package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// PromptYesNo asks the question and defaults to Yes
func PromptYesNo(question string) (bool, error) {
	color.New(color.Bold).Print(">>> ")
	fmt.Printf("%s [Y/n] ", question)

	r := bufio.NewReader(os.Stdin)
	answer, err := r.ReadString('\n')
	if err != nil {
		return false, err
	}

	// Sanitize
	answer = strings.TrimSpace(strings.ToLower(answer))
	if len(answer) > 1 {
		msg := fmt.Sprintf("Invalid option: '%s'", answer)
		return false, errors.New(msg)
	}

	if answer == "y" || answer == "" {
		return true, nil
	}

	return false, nil
}
