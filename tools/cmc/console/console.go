package console

import (
	"fmt"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
)

var exitCmd = &cobra.Command{
	Use:   "exit",
	Short: "Exit prompt",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(0)
	},
}

func NewConsoleCMD(rootCmd *cobra.Command) *cobra.Command {
	rootCmd.AddCommand(exitCmd)
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Open a console to interact with ChainMaker daemon",
		Long:  "Open a console to interact with ChainMaker daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Welcome to cmc console!\n")
			fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")
			defer fmt.Println("Bye!")
			console := &CobraPrompt{
				RootCmd:                rootCmd,
				DynamicSuggestionsFunc: handleDynamicSuggestions,
				PersistFlagValues:      true,
				GoPromptOptions: []prompt.Option{
					prompt.OptionTitle("cmc console: interactive ChainMaker client"),
					prompt.OptionPrefix(">>> "),
					prompt.OptionInputTextColor(prompt.Yellow),
					prompt.OptionMaxSuggestion(10),
				},
			}
			console.Run()
			return nil
		},
	}

	return cmd
}

func handleDynamicSuggestions(annotation string, _ *prompt.Document) []prompt.Suggest {
	switch annotation {
	default:
		return []prompt.Suggest{}
	}
}
