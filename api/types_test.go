package api

import "testing"

func TestAnswerMediaURLsIncludesAIRunOutputs(t *testing.T) {
	answer := Answer{
		AIRuns: []AIRun{
			{
				ID:     "run-1",
				Status: "succeeded",
				OutputURLs: []AIRunOutput{
					{
						URL: "//example.com/original.png",
						URLs: map[string]string{
							"medium": "//example.com/medium.png",
						},
						Variants: map[string]string{
							"large": "//example.com/large.png",
						},
					},
				},
			},
		},
	}

	if got := answer.GenerationStatus(); got != "completed" {
		t.Fatalf("expected completed status, got %q", got)
	}

	mediaURLs := answer.MediaURLs()
	for _, expectedURL := range []string{
		"//example.com/original.png",
		"//example.com/medium.png",
		"//example.com/large.png",
	} {
		if !containsString(mediaURLs, expectedURL) {
			t.Fatalf("expected media URLs to include %q, got %#v", expectedURL, mediaURLs)
		}
	}
}

func TestAnswerGenerationStatusPendingWithoutMedia(t *testing.T) {
	answer := Answer{
		AIRuns: []AIRun{
			{
				ID:     "run-1",
				Status: "pending",
			},
		},
	}

	if got := answer.GenerationStatus(); got != "pending" {
		t.Fatalf("expected pending status, got %q", got)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
