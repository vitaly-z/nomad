// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package consulcompat

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	consulTestUtil "github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/shoenig/test/must"
)

const (
	consulDataDir = "consul-data"
)

// startConsul runs a Consul agent with bootstrapped ACLs
func startConsul(t *testing.T, b build, baseDir string) (func() error, *consulTestUtil.TestServerConfig, string, *consulapi.Client) {

	path := filepath.Join(baseDir, binDir, b.Version)
	os.Chdir(path) // so that we can launch Consul from the current directory
	os.Setenv("PATH", path+":"+os.Getenv("PATH"))

	consulDC1 := "dc1"
	rootToken := uuid.Generate()
	agentToken := uuid.Generate()

	testconsul, err := consulTestUtil.NewTestServerConfigT(t,
		func(c *consulTestUtil.TestServerConfig) {
			c.ACL.Enabled = true
			c.ACL.DefaultPolicy = "deny"
			c.ACL.Tokens = consulTestUtil.TestTokens{
				Agent:             agentToken,
				InitialManagement: rootToken,
			}
			c.Datacenter = consulDC1
			c.DataDir = filepath.Join(baseDir, binDir, b.Version, consulDataDir)
			c.LogLevel = "info"
			c.Connect = map[string]any{"enabled": true}
			c.Server = true

			if !testing.Verbose() {
				c.Stdout = io.Discard
				c.Stderr = io.Discard
			}
		})
	must.NoError(t, err, must.Sprint("error starting test consul server"))

	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(baseDir, binDir, b.Version, consulDataDir))
	})

	testconsul.WaitForLeader(t)
	testconsul.WaitForActiveCARoot(t)

	// TODO: we should run this entire test suite with mTLS everywhere
	consulClient, err := consulapi.NewClient(&consulapi.Config{
		Address:    testconsul.HTTPAddr,
		Scheme:     "http",
		Datacenter: consulDC1,
		HttpClient: consulapi.DefaultConfig().HttpClient,
		Token:      rootToken,
		Namespace:  "default",
		Partition:  "default",
		TLSConfig:  consulapi.TLSConfig{},
	})
	must.NoError(t, err)

	return testconsul.Stop, testconsul.Config, agentToken, consulClient
}

func setupConsul(t *testing.T, consulAPI *consulapi.Client) {
	// TODO: install auth method, binding rules, and ACL policies
}
