package cli

import (
	"encoding/json"
	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print merged configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd.Context())
			if err != nil {
				return err
			}
			payload := map[string]any{
				"user": app.Config,
				"repo": app.RepoConfig,
			}
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte("\n"))
			return err
		},
	}
	return cmd
}
