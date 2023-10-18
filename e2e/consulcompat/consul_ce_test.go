// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package consulcompat

import (
	"fmt"
	"os"
	"testing"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/wait"
)

const (
	envGate    = "NOMAD_E2E_CONSULCOMPAT"
	envTempDir = "NOMAD_E2E_CONSULCOMPAT_BASEDIR"
)

func TestConsulCompat(t *testing.T) {
	if os.Getenv(envGate) != "1" {
		t.Skip(envGate + " is not set; skipping")
	}
	t.Run("testConsulVersions", testConsulVersions)
}

func testConsulVersions(t *testing.T) {

	baseDir := os.Getenv(envTempDir)
	if baseDir == "" {
		baseDir = t.TempDir()
	}

	versions := scanConsulVersions(t, getMinimumVersion(t))
	versions.ForEach(func(b build) bool {
		downloadConsulBuild(t, b, baseDir)
		testConsulBuild(t, b, baseDir)
		return true
	})
}

func testConsulBuild(t *testing.T, b build, baseDir string) {
	t.Run("consul("+b.Version+")", func(t *testing.T) {
		cStop, cCfg, agentToken, consulAPI := startConsul(t, b, baseDir)
		defer cStop()

		// smoke test before we continue
		self, err := consulAPI.Agent().Self()
		must.NoError(t, err)
		vers := self["Config"]["Version"].(string)
		must.Eq(t, b.Version, vers)

		// note: Nomad needs to be live before we can setupConsul because we
		// need it to serve the JWKS endpoint
		nStop, nc := startNomad(t, agentToken, cCfg)
		defer nStop()

		setupConsul(t, consulAPI)

		runConnectJob(t, nc)

		// give nomad and consul time to stop
		defer func() { time.Sleep(5 * time.Second) }()
	})
}

func runConnectJob(t *testing.T, nc *nomadapi.Client) {

	// TODO: how much of this can we lift from e2e/v3?
	b, err := os.ReadFile("input/connect.hcl")
	must.NoError(t, err)

	jobs := nc.Jobs()
	job, err := jobs.ParseHCL(string(b), true)
	must.NoError(t, err, must.Sprint("failed to parse job HCL"))

	_, _, err = jobs.Register(job, nil)
	must.NoError(t, err, must.Sprint("failed to register job"))

	t.Cleanup(func() {
		_, _, err = jobs.Deregister(*job.Name, true, nil)
		must.NoError(t, err, must.Sprint("faild to deregister job"))
	})

	must.Wait(t, wait.InitialSuccess(
		wait.ErrorFunc(func() error {
			allocs, _, err := jobs.Allocations(*job.ID, false, nil)
			if err != nil {
				return err
			}
			if n := len(allocs); n != 1 {
				return fmt.Errorf("expected 2 alloc, got %d", n)
			}
			if s := allocs[0].ClientStatus; s != "running" {
				return fmt.Errorf("expected alloc status running, got %s", s)
			}
			return nil
		}),
		wait.Timeout(20*time.Second),
		wait.Gap(1*time.Second),
	))

	// TODO: add test of the application so we can verify connectivity between
	// the two allocs
}
