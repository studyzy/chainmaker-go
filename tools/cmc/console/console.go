/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"fmt"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/spf13/cobra"
)

var exitCmd = &cobra.Command{
	Use:   "exit",
	Short: "Exit console",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Bye!")
		os.Exit(0)
	},
}

func NewConsoleCMD(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Open a console to interact with ChainMaker daemon",
		Long:  "Open a console to interact with ChainMaker daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			// remove console cmd, because already in console now.
			rootCmd.RemoveCommand(cmd)
			// add exit console command.
			rootCmd.AddCommand(exitCmd)
			fmt.Printf("Welcome to cmc console!\nPlease use `exit` or `Ctrl-D` to exit this program.\n")
			defer fmt.Println("Bye!")
			console := &CobraPrompt{
				RootCmd:                rootCmd,
				DynamicSuggestionsFunc: handleDynamicSuggestions,
				GoPromptOptions: []prompt.Option{
					prompt.OptionTitle("Interactive ChainMaker Client"),
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
