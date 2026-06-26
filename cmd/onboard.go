package cmd

import (
	"fmt"
	"os"

	treectlcontent "github.com/Knovigator/treectl/content"
	"github.com/spf13/cobra"
)

var OnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Output agent instructions for treectl",
	Long:  "Output agent-facing onboarding guidance and packaged-skill installation instructions for treectl.",
	RunE:  runOnboard,
}

var onboardShort bool
var onboardLong bool
var onboardAgentsMD bool
var onboardOutputPath string

func init() {
	OnboardCmd.Flags().BoolVar(&onboardShort, "short", false, "Use compact onboarding content")
	OnboardCmd.Flags().BoolVar(&onboardLong, "long", false, "Use full onboarding content (default)")
	OnboardCmd.Flags().BoolVar(&onboardAgentsMD, "agents-md", false, "Emit only the agents.md-ready block")
	OnboardCmd.Flags().StringVarP(&onboardOutputPath, "output", "o", "", "Write to file instead of stdout")
}

func runOnboard(cmd *cobra.Command, args []string) error {
	if onboardShort && onboardLong {
		return fmt.Errorf("use only one of --short or --long")
	}

	mode := "long"
	if onboardShort {
		mode = "short"
	}

	content, err := treectlcontent.BuildOnboardContent(mode)
	if err != nil {
		return err
	}

	if onboardAgentsMD {
		if mode == "short" {
			content, err = treectlcontent.OnboardShort()
		} else {
			content, err = treectlcontent.OnboardLong()
		}
		if err != nil {
			return err
		}
	}

	if onboardOutputPath != "" {
		err = os.WriteFile(onboardOutputPath, []byte(content), 0644)
		if err != nil {
			return err
		}
		fmt.Printf("Written to %s\n", onboardOutputPath)
		return nil
	}

	fmt.Print(content)
	if len(content) == 0 || content[len(content)-1] != '\n' {
		fmt.Println()
	}
	return nil
}
