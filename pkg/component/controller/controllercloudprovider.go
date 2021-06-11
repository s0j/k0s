/*
Copyright 2021 k0s authors

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
package controller

import (
	"time"

	"github.com/k0sproject/k0s/pkg/component"
	"github.com/k0sproject/k0s/pkg/controllercloudprovider"
)

type controllerCloudProvider struct {
	config        controllercloudprovider.Config
	stopCh        chan struct{}
	runnerBuilder RunnerBuilder
}

var _ component.Component = (*controllerCloudProvider)(nil)

// RunnerBuilder allows for defining arbitrary functions that can
// create `Runner` instances.
type RunnerBuilder func() (controllercloudprovider.Runner, error)

// NewControllerCloudProvider creates a new controller cloud-provider using the default
// address collector and runner.
func NewControllerCloudProvider(kubeConfigPath string, frequency time.Duration, port int) component.Component {
	config := controllercloudprovider.Config{
		AddressCollector: controllercloudprovider.DefaultAddressCollector(),
		KubeConfig:       kubeConfigPath,
		UpdateFrequency:  frequency,
		BindPort:         port,
	}

	return newControllerCloudProvider(config, func() (controllercloudprovider.Runner, error) {
		return controllercloudprovider.NewRunner(config)
	})
}

// newControllerCloudProvider is a helper for creating specialized controller-cloud-provider
// instances that can be used for testing.
func newControllerCloudProvider(config controllercloudprovider.Config, rb RunnerBuilder) component.Component {
	return &controllerCloudProvider{
		config:        config,
		stopCh:        make(chan struct{}),
		runnerBuilder: rb,
	}
}

// Init in the controller-cloud-provider intentionally does nothing.
func (c *controllerCloudProvider) Init() error {
	return nil
}

// Run will create a controller-cloud-provider runner, and run it on a goroutine.
// Failures to create this runner will be returned as an error.
func (c *controllerCloudProvider) Run() error {
	runner, err := c.runnerBuilder()
	if err != nil {
		return err
	}

	go runner(c.stopCh)

	return nil
}

// Stop will stop the controller-cloud-provider runner goroutine (if running)
func (c *controllerCloudProvider) Stop() error {
	close(c.stopCh)

	return nil
}

// Healthy in the controller-cloud-provider intentionally does nothing.
func (c *controllerCloudProvider) Healthy() error {
	return nil
}
