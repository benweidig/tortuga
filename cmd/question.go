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

	if len(answer) > 2 {
		msg := fmt.Sprintf("Invalid option: %s", answer)
		return false, errors.New(msg)
	}

	if strings.ToLower(answer) == "n\n" {
		return false, nil
	}

	return true, nil
}
