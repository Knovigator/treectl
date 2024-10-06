package cmd

import (
	"fmt"

	"github.com/Knovigator/knovigator/treectl/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newClipCmd = &cobra.Command{
	Use:   "clip <url>",
	Short: "Create a new clip",
	Long:  `Create a new clip from a URL.`,
	Args:  cobra.ExactArgs(1),
	Run:   runNewClip,
}

var content string
var title string

func init() {
	newClipCmd.Flags().StringVarP(&content, "content", "c", "", "Additional content for the clip")
	newClipCmd.Flags().StringVarP(&title, "title", "t", "", "Title for the clip")
}

func runNewClip(cmd *cobra.Command, args []string) {
	url := args[0]

	// load credentials from viper config
	accessToken := viper.GetString("access_token")
	client := viper.GetString("client")
	uid := viper.GetString("uid")
	backendURL := viper.GetString("backend_url")

	if accessToken == "" || client == "" || uid == "" || backendURL == "" {
		fmt.Println("Error: Missing credentials. Please login first.")
		return
	}

	// Set the destination to a stream with id 'PSEUDOSTREAM__CLIPS'
	destination := map[string]interface{}{
		"type": "stream",
		"name": "Clips",
		"id":   "PSEUDOSTREAM__CLIPS",
	}

	result, err := api.ClipLink(
		backendURL,
		accessToken,
		client,
		uid,
		url,
		nil, // image
		nil, // video
		nil, // file
		title,
		content,
		destination,
	)

	if err != nil {
		fmt.Println("Error creating clip:", err)
		return
	} else {
		fmt.Printf("Clip created successfully: http://home.treechat.ai/quest/%s\n", result["id"])
		// collect and print keys of result
		// keys := []string{}
		// for key := range result {
		// 	keys = append(keys, key)
		// }
		// fmt.Println("Keys of result:", keys)
	}
}
