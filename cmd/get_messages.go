package cmd

import (
	"fmt"

	"github.com/Knovigator/knovigator/treectl/api"
	"github.com/spf13/cobra"
)

var getMessagesCmd = &cobra.Command{
	Use:     "messages [message_id1] [message_id2] ...",
	Aliases: []string{"answers"},
	Short:   "Get information about specific messages",
	Long:    `Fetch and display information about one or more messages using their IDs.`,
	Args:    cobra.MinimumNArgs(1),
	Run:     runGetMessages,
}

var outputFormat string
var getJSONOutput bool

func init() {
	getMessagesCmd.Flags().StringVarP(&outputFormat, "output", "o", "ascii", "Output format: ascii or json")
	getMessagesCmd.Flags().BoolVar(&getJSONOutput, "json", false, "Output JSON instead of human-readable text")
}

func runGetMessages(cmd *cobra.Command, args []string) {
	messageIDs := args

	profile, err := requireAuthenticatedProfile()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	messagesInfo, err := api.GetMessages(profile.BackendURL, profile.AccessToken, profile.Client, profile.UID, messageIDs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	switch resolveOutputFormat(outputFormat, getJSONOutput) {
	case "json":
		prettyJSON, err := api.PrettyJSON(messagesInfo.Raw)
		if err != nil {
			fmt.Printf("Error formatting JSON: %v\n", err)
			return
		}
		fmt.Println(prettyJSON)
	case "ascii":
		for _, answer := range messagesInfo.Answers {
			fmt.Println(answer.ToASCII())
		}
	default:
		fmt.Printf("Invalid output format: %s. Use 'ascii' or 'json'.\n", outputFormat)
	}
}
