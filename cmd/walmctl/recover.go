package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/klog"
)

type recoverCmd struct {
	out io.Writer
	releaseName string
	async bool
}

func newRecoverCmd(out io.Writer) *cobra.Command {
	rc := recoverCmd{out: out}

	cmd := &cobra.Command{
		Use:   "recover",
		Short: "恢复Release服务",
		RunE: func(cmd *cobra.Command, args []string) error {

			if walmserver == "" {
				return errServerRequired
			}
			if len(args) != 2{
				return errors.New("arguments error, recover command receive two arguments, eg: recover release releaseName")
			}
			if args[0] != "release" {
				return errors.New("arguments error, only support recover release")
			}
			rc.releaseName = args[1]
			return rc.run()
		},
	}
	cmd.PersistentFlags().BoolVar(&rc.async, "async", false, "whether asynchronous")
	return cmd
}

func (rc *recoverCmd) run() error {
	client, err := walmctlclient.CreateNewClient(walmserver, enableTLS, rootCA)
	if err != nil {
		klog.Errorf("failed to create walmctl client: %s", err.Error())
		return err
	}
	if err = client.ValidateHostConnect(walmserver); err != nil {
		return err
	}

	_, err = client.RecoverRelease(namespace, rc.releaseName, rc.async)
	if err != nil {
		klog.Errorf(err.Error())
		return err
	}
	return nil
}