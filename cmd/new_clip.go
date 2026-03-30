package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var newClipCmd = &cobra.Command{
	Use:   "clip <url>",
	Short: "Create a new clip",
	Long:  `Create a new clip from a URL or with an attachment.`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runNewClip,
}

var clipContent string
var clipAttachment string
var clipStream string

func init() {
	newClipCmd.Flags().StringVarP(&clipContent, "content", "c", "", "Additional content for the clip")
	newClipCmd.Flags().StringVarP(&clipAttachment, "attachment", "f", "", "Path to the file to attach")
	newClipCmd.Flags().StringVar(&clipStream, "stream", "", "Target stream name or UUID. Defaults to clips.")
}

func runNewClip(cmd *cobra.Command, args []string) {
	var url string
	if len(args) > 0 {
		url = args[0]
	}

	profile, err := requireAuthenticatedProfile()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	target, err := resolveStreamTarget(profile, clipStream, streamTarget{
		Kind: "clips",
		ID:   "PSEUDOSTREAM__CLIPS",
		Name: "Clips",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	clipLink(url, clipContent, clipAttachment, target)
}
