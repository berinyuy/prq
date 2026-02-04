package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var configPath string

	root := &cobra.Command{
		Use:           "prq",
		Short:         "Pull request review queue",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			app, err := initApp(configPath)
			if err != nil {
				return err
			}
			cmd.SetContext(withApp(context.Background(), app))
			return nil
		},
	}

	root.PersistentFlags().StringVar(&configPath, "config", "", "Override config path")

	root.AddCommand(NewDoctorCmd())
	root.AddCommand(NewQueueCmd())
	root.AddCommand(NewPickCmd())
	root.AddCommand(NewReviewCmd())
	root.AddCommand(NewDraftCmd())
	root.AddCommand(NewSubmitCmd())
	root.AddCommand(NewFollowupCmd())
	root.AddCommand(NewConfigCmd())

	return root
}
