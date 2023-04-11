package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/sunbeam/internal"
	"github.com/spf13/cobra"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func NewCmdEval() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval [file]",
		Args:  cobra.MaximumNArgs(1),
		Short: "Evaluate a file or stdin as a page",
		RunE: func(cmd *cobra.Command, args []string) error {
			generator := func() ([]byte, error) {
				buffer := bytes.Buffer{}
				i := interp.New(interp.Options{
					Stdout: &buffer,
				})
				i.Use(stdlib.Symbols)

				if len(args) > 0 {
					if _, err := i.EvalPath(args[0]); err != nil {
						return nil, err
					}
				} else {
					b, err := io.ReadAll(os.Stdin)
					if err != nil {
						return nil, err
					}

					if _, err := i.Eval(string(b)); err != nil {
						return nil, err
					}
				}

				return buffer.Bytes(), nil
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				b, err := generator()
				if err != nil {
					return err
				}
				fmt.Print(string(b))
				return nil
			}

			runner := internal.NewRunner(generator)
			return internal.NewPaginator(runner).Draw()
		},
	}

	return cmd
}