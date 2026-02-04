package cli

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/brianndofor/prq/internal/prompt"
	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check dependencies and configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd.Context())
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			fmt.Fprintln(cmd.OutOrStdout(), "prq doctor")
			if err := app.GH.CheckInstalled(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "- gh: ok")
			if err := app.GH.AuthStatus(ctx); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "- gh auth: ok")

			if _, err := exec.LookPath(app.Config.Provider.Command); err != nil {
				return fmt.Errorf("provider not found: %s", app.Config.Provider.Command)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "- provider: ok")

			schema := prompt.DefaultSchemaPath()
			if err := app.Provider.HealthCheck(ctx, schema); err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "- provider schema: failed\n%v\n", err)
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "- provider schema: ok")
			fmt.Fprintln(cmd.OutOrStdout(), "doctor checks passed")
			return nil
		},
	}
	return cmd
}
