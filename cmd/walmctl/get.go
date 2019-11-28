package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/go-resty/resty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
)

const getDesc = `
Get a walm release or project detail info.
Options:
use --output/-o to print with json/yaml format.

You must specify the type of resource to get. Valid resource types include:
  * release
  * project
  * migration

[release]
walmctl get release xxx -n/--namespace xxx

[project]
walmctl get project xxx -n/--namespace xxx

[migration]
walmctl get migration pod xxx -n/--namespace xxx
walmctl get migration node xxx
`

type getCmd struct {
	sourceType  string
	sourceName  string
	subType     string
	output 		string
	out    		io.Writer
}


func newGetCmd(out io.Writer) *cobra.Command {
	gc := getCmd{out:out}

	cmd := &cobra.Command{
		Use: "get",
		DisableFlagsInUseLine: true,
		Short: "get [release | project | migration]",
		Long: getDesc,
		RunE: func(cmd *cobra.Command, args []string) error {

			if walmserver == "" {
				return errServerRequired
			}

			if err := checkResourceType(args[0]); err != nil {
				return err
			}
			gc.sourceType = args[0]
			if gc.sourceType == "migration" {
				if len(args) != 3 {
					return errors.Errorf("arguments error, get migration pod/node xxx")
				}
				if args[1] != "pod" && args[1] != "node" {
					return errors.Errorf("arguments error, invalid migration type: %s", args[1])
				}
				gc.subType = args[1]
				gc.sourceName = args[2]
			} else {
				if len(args) != 2 {
					return errors.Errorf("arguments error, get release/project xxx")
				}
				gc.sourceName = args[1]
			}

			if namespace == "" && gc.subType != "node" {
				return errNamespaceRequired
			}

			return gc.run()
		},
	}

	cmd.Flags().StringVarP(&gc.output, "output", "o", "json", "-o, --output='': Output format for detail description. Support: json, yaml")
	return cmd
}

func (gc *getCmd) run() error {

	var resp *resty.Response
	var err error

	client := walmctlclient.CreateNewClient(walmserver)
	if err = client.ValidateHostConnect(); err != nil {
		return err
	}
	if gc.sourceType == "release" {
		resp, err = client.GetRelease(namespace, gc.sourceName)
	} else if gc.sourceType == "project"{
		resp, err =client.GetProject(namespace, gc.sourceName)
	} else if gc.sourceType == "migration"{
		if gc.subType == "node" {
			resp, err = client.GetNodeMigration(gc.sourceName)
		} else {
			resp, err = client.GetPodMigration(namespace, gc.sourceName)
		}
	}

	if err != nil {
		return err
	}

	if gc.output == "yaml" {
		respByte, err := yaml.JSONToYAML(resp.Body())
		if err != nil {
			return errors.New(err.Error())
		}
		fmt.Printf(string(respByte))

	} else if gc.output == "json" {
		fmt.Println(resp)
	} else {
		return errors.Errorf("output format %s not recognized, only support yaml, json", gc.output)
	}

	return nil
}