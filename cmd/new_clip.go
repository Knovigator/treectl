package cmd

import (
	"fmt"
	"mime"
	"os" // added import for os package
	"path/filepath"
	"strings"

	"github.com/Knovigator/knovigator/treectl/api"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newClipCmd = &cobra.Command{
	Use:   "clip <url>",
	Short: "Create a new clip",
	Long:  `Create a new clip from a URL or with an attachment.`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runNewClip,
}

var content string
var attachment string

func init() {
	newClipCmd.Flags().StringVarP(&content, "content", "c", "", "Additional content for the clip")
	newClipCmd.Flags().StringVarP(&attachment, "attachment", "f", "", "Path to the file to attach")
}

func runNewClip(cmd *cobra.Command, args []string) {
	var url string
	if len(args) > 0 {
		url = args[0]
	}

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

	var image, video, file []byte
	var err error

	if attachment != "" {
		fileContent, err := os.ReadFile(attachment)
		if err != nil {
			fmt.Printf("Error reading attachment file: %v\n", err)
			return
		}

		mimeType := getMimeType(attachment)
		switch {
		case strings.HasPrefix(mimeType, "image/"):
			image = fileContent
		case strings.HasPrefix(mimeType, "video/"):
			video = fileContent
		default:
			file = fileContent
		}
	}

	result, err := api.ClipLink(
		backendURL,
		accessToken,
		client,
		uid,
		url,
		image,
		video,
		file,
		"",
		content,
		destination,
	)

	if err != nil {
		fmt.Println("Error creating clip:", err)
		return
	} else {
		fmt.Printf("Clip created successfully. See it at: http://home.treechat.ai/quest/%s\n", result["id"])
	}
}

func getMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// If the MIME type is not found, default to application/octet-stream
		mimeType = "application/octet-stream"
	}
	return mimeType
}
