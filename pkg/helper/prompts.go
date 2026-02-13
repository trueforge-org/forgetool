package helper

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

var (
	promptNewReaderFn  = func() *bufio.Reader { return bufio.NewReader(os.Stdin) }
	promptReadStringFn = func(reader *bufio.Reader) (string, error) { return reader.ReadString('\n') }
)

// getYesOrNo prompts the user with a question and returns true for yes and false for no
func GetYesOrNo(prompt string) bool {
	reader := promptNewReaderFn()
	for {
		fmt.Print(prompt)
		input, err := promptReadStringFn(reader)
		if err != nil {
			log.Info().Msgf("An error occurred: %v", err)
			continue
		}

		input = strings.TrimSpace(input)
		input = strings.ToLower(input)

		switch input {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		default:
			log.Info().Msg("Invalid input. Please enter yes/no or y/n.")
		}
	}
}
