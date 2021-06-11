package controller

import (
	"sync"
	"testing"
	"time"

	"github.com/k0sproject/k0s/pkg/component"
	"github.com/k0sproject/k0s/pkg/controllercloudprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
)

func EmptyAddressCollector(node *v1.Node) []v1.NodeAddress {
	return []v1.NodeAddress{}
}

// DummyRunner is a simple `controllercloudprovider.Runner` that returns if the
// provided channel is closed, or after a 30s timeout.  The main motivation is
// to have a runner that waits until completed, with notification.
func DummyRunner(wg *sync.WaitGroup, cancelled *bool) (controllercloudprovider.Runner, error) {
	wg.Add(1)

	return func(stopCh <-chan struct{}) error {
		defer wg.Done()

		t := time.NewTimer(30 * time.Second)
		defer t.Stop()

		select {
		case <-stopCh:
			*cancelled = true
		case <-t.C:
		}

		return nil
	}, nil
}

// DummyRunnerBuilder adapts `DummyRunner` to `RunnerBuilder`
func DummyRunnerBuilder(wg *sync.WaitGroup, cancelled *bool) RunnerBuilder {
	return func() (controllercloudprovider.Runner, error) {
		return DummyRunner(wg, cancelled)
	}
}

type ControllerCloudProviderSuite struct {
	suite.Suite
	ccp       component.Component
	cancelled bool
	wg        sync.WaitGroup
}

// SetupTest builds a makeshift controller-cloud-provider configuration, and
// a dummy-runner (no-op) per-test invocation.
func (suite *ControllerCloudProviderSuite) SetupTest() {
	config := controllercloudprovider.Config{
		AddressCollector: EmptyAddressCollector,
		KubeConfig:       "/does/not/exist",
		UpdateFrequency:  1 * time.Second,
	}

	suite.ccp = newControllerCloudProvider(config, DummyRunnerBuilder(&suite.wg, &suite.cancelled))
	assert.NotNil(suite.T(), suite.ccp)
}

// TestInit covers the `Init()` function.
func (suite *ControllerCloudProviderSuite) TestInit() {
	assert.Nil(suite.T(), suite.ccp.Init())
}

// TestRunStop covers the scenario of issuing a `Start()`, and ensuring
// that when `Stop()` is called, the underlying goroutine is cancelled.
// This is effectively testing the close-channel semantics baked into
// `Stop()`, without worrying about what was actually running.
func (suite *ControllerCloudProviderSuite) TestRunStop() {
	assert.Nil(suite.T(), suite.ccp.Init())
	assert.Nil(suite.T(), suite.ccp.Run())

	// Ensures that the stopping mechanism actually closes the stop channel.
	assert.Nil(suite.T(), suite.ccp.Stop())
	suite.wg.Wait()

	assert.Equal(suite.T(), true, suite.cancelled)
}

// TestHealthy covers the `Healthy()` function post-init.
func (suite *ControllerCloudProviderSuite) TestHealthy() {
	assert.Nil(suite.T(), suite.ccp.Init())
	assert.Nil(suite.T(), suite.ccp.Healthy())
}

// TestControllerCloudProviderTestSuite sets up the suite for testing.
func TestControllerCloudProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerCloudProviderSuite))
}
