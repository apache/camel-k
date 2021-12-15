/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	user "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/apache/camel-k/pkg/apis"
	camel "github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
	camelv1 "github.com/apache/camel-k/pkg/client/camel/clientset/versioned/typed/camel/v1"
	camelv1alpha1 "github.com/apache/camel-k/pkg/client/camel/clientset/versioned/typed/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
)

const (
	inContainerNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	kubeConfigEnvVar         = "KUBECONFIG"
)

// Client is an abstraction for a k8s client.
type Client interface {
	ctrl.Client
	kubernetes.Interface
	CamelV1() camelv1.CamelV1Interface
	CamelV1alpha1() camelv1alpha1.CamelV1alpha1Interface
	GetScheme() *runtime.Scheme
	GetConfig() *rest.Config
	GetCurrentNamespace(kubeConfig string) (string, error)
	ServerOrClientSideApplier() ServerOrClientSideApplier
}

// Injectable identifies objects that can receive a Client.
type Injectable interface {
	InjectClient(Client)
}

// Provider is used to provide a new instance of the Client each time it's required.
type Provider struct {
	Get func() (Client, error)
}

type defaultClient struct {
	ctrl.Client
	kubernetes.Interface
	camel  camel.Interface
	scheme *runtime.Scheme
	config *rest.Config
}

// Check interface compliance.
var _ Client = &defaultClient{}

func (c *defaultClient) CamelV1() camelv1.CamelV1Interface {
	return c.camel.CamelV1()
}

func (c *defaultClient) CamelV1alpha1() camelv1alpha1.CamelV1alpha1Interface {
	return c.camel.CamelV1alpha1()
}

func (c *defaultClient) GetScheme() *runtime.Scheme {
	return c.scheme
}

func (c *defaultClient) GetConfig() *rest.Config {
	return c.config
}

func (c *defaultClient) GetCurrentNamespace(kubeConfig string) (string, error) {
	return GetCurrentNamespace(kubeConfig)
}

// NewOutOfClusterClient creates a new k8s client that can be used from outside the cluster.
func NewOutOfClusterClient(kubeconfig string) (Client, error) {
	initialize(kubeconfig)
	// using fast discovery from outside the cluster
	return NewClient(true)
}

// NewClient creates a new k8s client that can be used from outside or in the cluster.
func NewClient(fastDiscovery bool) (Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return NewClientWithConfig(fastDiscovery, cfg)
}

// NewClientWithConfig creates a new k8s client that can be used from outside or in the cluster.
func NewClientWithConfig(fastDiscovery bool, cfg *rest.Config) (Client, error) {
	clientScheme := scheme.Scheme

	// Setup Scheme for all resources
	err := apis.AddToScheme(clientScheme)
	if err != nil {
		return nil, err
	}

	var clientset kubernetes.Interface
	if clientset, err = kubernetes.NewForConfig(cfg); err != nil {
		return nil, err
	}

	var camelClientset camel.Interface
	if camelClientset, err = camel.NewForConfig(cfg); err != nil {
		return nil, err
	}

	var mapper meta.RESTMapper
	if fastDiscovery {
		mapper = newFastDiscoveryRESTMapper(cfg)
	}

	// Create a new client to avoid using cache (enabled by default with controller-runtime client)
	clientOptions := ctrl.Options{
		Scheme: clientScheme,
		Mapper: mapper,
	}
	dynClient, err := ctrl.New(cfg, clientOptions)
	if err != nil {
		return nil, err
	}

	return &defaultClient{
		Client:    dynClient,
		Interface: clientset,
		camel:     camelClientset,
		scheme:    clientOptions.Scheme,
		config:    cfg,
	}, nil
}

// FromManager creates a new k8s client from a manager object.
func FromManager(manager manager.Manager) (Client, error) {
	var err error
	var clientset kubernetes.Interface
	if clientset, err = kubernetes.NewForConfig(manager.GetConfig()); err != nil {
		return nil, err
	}
	var camelClientset camel.Interface
	if camelClientset, err = camel.NewForConfig(manager.GetConfig()); err != nil {
		return nil, err
	}

	return &defaultClient{
		Client:    manager.GetClient(),
		Interface: clientset,
		camel:     camelClientset,
		scheme:    manager.GetScheme(),
		config:    manager.GetConfig(),
	}, nil
}

// init initialize the k8s client for usage outside the cluster.
func initialize(kubeconfig string) {
	if kubeconfig == "" {
		// skip out-of-cluster initialization if inside the container
		if kc, err := shouldUseContainerMode(); kc && err == nil {
			return
		} else if err != nil {
			logrus.Errorf("could not determine if running in a container: %v", err)
		}
		var err error
		kubeconfig, err = getDefaultKubeConfigFile()
		if err != nil {
			panic(err)
		}
	}

	if err := os.Setenv(kubeConfigEnvVar, kubeconfig); err != nil {
		panic(err)
	}
}

func getDefaultKubeConfigFile() (string, error) {
	dir, err := user.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".kube", "config"), nil
}

// GetCurrentNamespace --.
func GetCurrentNamespace(kubeconfig string) (string, error) {
	if kubeconfig == "" {
		kubeContainer, err := shouldUseContainerMode()
		if err != nil {
			return "", err
		}
		if kubeContainer {
			return getNamespaceFromKubernetesContainer()
		}
	}
	if kubeconfig == "" {
		var err error
		kubeconfig, err = getDefaultKubeConfigFile()
		if err != nil {
			logrus.Errorf("Cannot get information about current user: %v", err)
		}
	}
	if kubeconfig == "" {
		return "default", nil
	}

	data, err := util.ReadFile(kubeconfig)
	if err != nil {
		return "", err
	}
	conf := clientcmdapi.NewConfig()
	if len(data) == 0 {
		return "", errors.New("kubernetes config file is empty")
	}

	decoded, _, err := clientcmdlatest.Codec.Decode(data, &schema.GroupVersionKind{Version: clientcmdlatest.Version, Kind: "Config"}, conf)
	if err != nil {
		return "", err
	}

	clientcmdconfig, ok := decoded.(*clientcmdapi.Config)
	if !ok {
		return "", fmt.Errorf("type assertion failed: %v", decoded)
	}

	cc := clientcmd.NewDefaultClientConfig(*clientcmdconfig, &clientcmd.ConfigOverrides{})
	ns, _, err := cc.Namespace()
	return ns, err
}

func shouldUseContainerMode() (bool, error) {
	// When kube config is set, container mode is not used
	if os.Getenv(kubeConfigEnvVar) != "" {
		return false, nil
	}
	// Use container mode only when the kubeConfigFile does not exist and the container namespace file is present
	configFile, err := getDefaultKubeConfigFile()
	if err != nil {
		return false, err
	}
	configFilePresent := true
	_, err = os.Stat(configFile)
	if err != nil && os.IsNotExist(err) {
		configFilePresent = false
	} else if err != nil {
		return false, err
	}
	if !configFilePresent {
		_, err := os.Stat(inContainerNamespaceFile)
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}
	return false, nil
}

func getNamespaceFromKubernetesContainer() (string, error) {
	var nsba []byte
	var err error
	if nsba, err = ioutil.ReadFile(inContainerNamespaceFile); err != nil {
		return "", err
	}
	return string(nsba), nil
}
