package cmd

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/Knovigator/knovigator/treectl/api"
)

func clipLink(url, content, attachment string, isClip bool) {
	profile, err := requireAuthenticatedProfile()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// set the destination to a stream with id 'PSEUDOSTREAM__CLIPS' for clips, or 'PSEUDOSTREAM__POSTS' for posts
	destinationName := "Clips"
	destinationId := "PSEUDOSTREAM__CLIPS"
	if !isClip {
		destinationName = "Posts"
		destinationId = "PSEUDOSTREAM__POSTS"
	}

	destination := map[string]interface{}{
		"type": "stream",
		"name": destinationName,
		"id":   destinationId,
	}

	var image, video, file []byte

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
		profile.BackendURL,
		profile.AccessToken,
		profile.Client,
		profile.UID,
		url,
		image,
		video,
		file,
		"",
		content,
		destination,
	)

	if err != nil {
		fmt.Println("Error creating post:", err)
		return
	} else {
		linkBase := profile.AppHost
		if linkBase == "" {
			linkBase = profile.BackendURL
		}

		fmt.Printf("Post created successfully. See it at: %s/quest/%s\n", strings.TrimRight(linkBase, "/"), result["id"])
	}
}

func getMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// if the MIME type is not found, default to application/octet-stream
		mimeType = "application/octet-stream"
	}
	return mimeType
}
