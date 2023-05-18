package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jwalton/gchalk"
)

// PromptYesNo asks the question and defaults to Yes
func PromptYesNo(question string) (bool, error) {
	fmt.Printf("%s %s [Y/n] ", gchalk.WithWhite().Bold(">>>"), question)

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
