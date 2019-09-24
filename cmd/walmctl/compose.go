package main

import (
	"github.com/spf13/cobra"
)

const composeDesc = `
Compose a Walm Compose file
`

type composeCmd struct {
	file string
}

func newComposeCmd() *cobra.Command {
	compose := composeCmd{}
	cmd := &cobra.Command{
		Use:   "compose [file]",
		Short: "Compose a Walm Compose file",
		Long:  composeDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return compose.run()
		},
	}
	cmd.PersistentFlags().StringVar(&compose.file, "file", "walmcompose.yaml", "walm compose file")
	cmd.MarkFlagRequired("file")

	return cmd
}

func (compose *composeCmd) run() error {
	return nil
}
