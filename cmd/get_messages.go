package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Knovigator/knovigator/treectl/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func init() {
	getMessagesCmd.Flags().StringVarP(&outputFormat, "output", "o", "ascii", "Output format: ascii or json")
}

func runGetMessages(cmd *cobra.Command, args []string) {
	messageIDs := args

	// load credentials from viper config
	accessToken := viper.GetString("access_token")
	client := viper.GetString("client")
	uid := viper.GetString("uid")
	backendURL := viper.GetString("backend_url")

	if accessToken == "" || client == "" || uid == "" || backendURL == "" {
		fmt.Println("Error: Missing credentials. Please login first.")
		return
	}

	messagesInfo, err := api.GetMessages(backendURL, accessToken, client, uid, messageIDs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	switch outputFormat {
	case "json":
		// pretty print the messages info
		prettyJSON, err := json.MarshalIndent(messagesInfo, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting JSON: %v\n", err)
			return
		}
		fmt.Println(string(prettyJSON))
	case "ascii":
		if answers, ok := messagesInfo["answers"].([]interface{}); ok {
			for _, answer := range answers {
				if answerMap, ok := answer.(map[string]interface{}); ok {
					message := api.Message{
						ID:      answerMap["id"].(string),
						Content: answerMap["content"].(string),
						Extra:   answerMap,
					}
					fmt.Println(message.ToASCII())
				}
			}
		}
	default:
		fmt.Printf("Invalid output format: %s. Use 'ascii' or 'json'.\n", outputFormat)
	}
}
