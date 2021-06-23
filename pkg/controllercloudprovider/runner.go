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
package controllercloudprovider

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cloud-provider/app"
	"k8s.io/cloud-provider/options"
)

type Runner func(stopCh <-chan struct{}) error

type Config struct {
	AddressCollector AddressCollector
	KubeConfig       string
	BindPort         int
	UpdateFrequency  time.Duration
}

// NewRunner creates a new controller-cloud-provider based on a configuration.
// The runner itself is a specialization of the sample code available from
// `k8s.io/cloud-provider/app`
func NewRunner(c Config) (Runner, error) {
	ccmo, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize command options: %w", err)
	}

	ccmo.KubeCloudShared.CloudProvider.Name = Name
	ccmo.Kubeconfig = c.KubeConfig

	if c.BindPort != 0 {
		ccmo.SecureServing.BindPort = c.BindPort
	}

	if c.UpdateFrequency != 0 {
		ccmo.NodeStatusUpdateFrequency = metav1.Duration{Duration: c.UpdateFrequency}
	}

	controllerList := []string{"cloud-node", "cloud-node-lifecycle", "service", "route"}
	disabledControllerList := []string{"service", "route"}

	ccmc, err := ccmo.Config(controllerList, disabledControllerList)
	if err != nil {
		return nil, fmt.Errorf("unable to create controller-cloud-provider configuration: %w", err)
	}

	// Builds the provider using the specified `AddressCollector`

	cloud := NewProvider(c.AddressCollector)
	if err != nil {
		return nil, fmt.Errorf("unable to create controller-cloud-provider: %w", err)
	}

	return func(stopCh <-chan struct{}) error {
		cloud.Initialize(ccmc.ClientBuilder, stopCh)

		controllerInitializers := app.DefaultControllerInitializers(ccmc.Complete(), cloud)
		for _, disabledController := range disabledControllerList {
			delete(controllerInitializers, disabledController)
		}

		// Override the commands arguments to avoid it by default using `os.Args[]`
		command := app.NewCloudControllerManagerCommand(ccmo, ccmc, controllerInitializers)
		command.SetArgs([]string{})

		if err := command.Execute(); err != nil {
			return fmt.Errorf("unable to execute command: %w", err)
		}

		return nil
	}, nil
}
