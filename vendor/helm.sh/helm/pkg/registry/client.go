/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registry

import (
	"io"

	"helm.sh/helm/internal/experimental/registry"
	"github.com/containerd/containerd/remotes"
)

type (
	// ClientOptions is used to construct a new client
	ClientOptions struct {
		Out          io.Writer
		Resolver     remotes.Resolver
	}

	// Client works with OCI-compliant registries and local Helm chart cache
	Client struct {
		registry.Client
	}
)


// NewClient returns a new registry client with config
func NewClient(options *ClientOptions) (*Client, error) {
	client, err := registry.NewClient(
		registry.ClientOptAuthorizer(&registry.Authorizer{}),
		registry.ClientOptDebug(true),
		registry.ClientOptWriter(options.Out),
		registry.ClientOptResolver(&registry.Resolver{Resolver: options.Resolver}),
	)
	if err != nil {
		return nil, err
	}
	return &Client{Client: *client}, nil
}

func ParseReference(s string) (*registry.Reference, error) {
	return registry.ParseReference(s)
}
