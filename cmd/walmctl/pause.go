package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"k8s.io/klog"
)

type pauseCmd struct {
	out io.Writer
	releaseName string
	async bool
}

func newPauseCmd(out io.Writer) *cobra.Command {
	pc := pauseCmd{out: out}

	cmd := &cobra.Command{
		Use:   "pause",
		Short: "暂停Release服务",
		RunE: func(cmd *cobra.Command, args []string) error {

			if walmserver == "" {
				return errServerRequired
			}
			if len(args) != 2{
				return errors.New("arguments error, pause command receive two arguments, eg: pause release releaseName")
			}
			if args[0] != "release" {
				return errors.New("arguments error, only support pause release")
			}
			pc.releaseName = args[1]
			return pc.run()
		},
	}
	cmd.PersistentFlags().BoolVar(&pc.async, "async", false, "whether asynchronous")
	return cmd
}

func (pc *pauseCmd) run() error {
	client, err := walmctlclient.CreateNewClient(walmserver, enableTLS, rootCA)
	if err != nil {
		klog.Errorf("failed to create walmctl client: %s", err.Error())
		return err
	}
	if err = client.ValidateHostConnect(walmserver); err != nil {
		return err
	}

	_, err = client.PauseRelease(namespace, pc.releaseName, pc.async)
	if err != nil {
		klog.Errorf(err.Error())
		return err
	}
	return nil
}