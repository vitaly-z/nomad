// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package consulcompat

import (
	"testing"

	consulTestUtil "github.com/hashicorp/consul/sdk/testutil"
	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper/testlog"
	"github.com/hashicorp/nomad/helper/uuid"
	testutil "github.com/hashicorp/nomad/testutil"
	"github.com/shoenig/test/must"
)

// startNomad runs a Nomad agent in dev mode with bootstrapped ACLs
func startNomad(t *testing.T, consulToken string, consulCfg *consulTestUtil.TestServerConfig) (func(), *nomadapi.Client) {
	ts := testutil.NewTestServer(t, func(c *testutil.TestServerConfig) {
		c.Consul = &testutil.Consul{ // TODO: this doesn't match the full config anymore!
			Address: consulCfg.Addresses.HTTP,
			Token:   consulToken,
		}
		c.DevMode = true
		c.ACL.Enabled = true
		c.Client = &testutil.ClientConfig{
			Enabled: true,
		}
		c.LogLevel = testlog.HCLoggerTestLevel().String()
	})

	// TODO: we should run this entire test suite with mTLS everywhere
	nc, err := nomadapi.NewClient(&nomadapi.Config{
		Address:   "http://" + ts.HTTPAddr,
		TLSConfig: &nomadapi.TLSConfig{},
	})
	must.NoError(t, err, must.Sprint("unable to create nomad api client"))

	rootToken := uuid.Generate()
	_, _, err = nc.ACLTokens().BootstrapOpts(rootToken, nil)
	must.NoError(t, err)
	nc.SetSecretID(rootToken)

	return ts.Stop, nc
}
