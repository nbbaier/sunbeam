package cmd

import (
	"io"
	"os"

	"github.com/pomdtr/sunbeam/internal"
	"github.com/spf13/cobra"
)

func NewReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <page>",
		Short: "Read page from file, and push it's content",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return Draw(func() ([]byte, error) {
					return io.ReadAll(os.Stdin)
				})
			}
			return Draw(internal.NewFileGenerator(args[0]))
		},
	}

	return cmd
}
