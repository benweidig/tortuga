package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

func askQuestionYN(question string) (bool, error) {
	color.New(color.Bold).Print(">>> ")
	fmt.Printf("%s [Y/n] ", question)

	r := bufio.NewReader(os.Stdin)
	answer, err := r.ReadString('\n')
	if err != nil {
		return false, err
	}

	sanitizedAnswer := strings.TrimSpace(strings.ToLower(answer))
	if len(sanitizedAnswer) > 1 {
		msg := fmt.Sprintf("Invalid option: '%s'", sanitizedAnswer)
		return false, errors.New(msg)
	}

	if sanitizedAnswer == "y" || sanitizedAnswer == "" {
		return true, nil
	}

	return false, nil
}
