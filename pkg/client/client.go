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
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/apache/camel-k/pkg/apis"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Client is an abstraction for a k8s client
type Client interface {
	controller.Client
	kubernetes.Interface
	GetScheme() *runtime.Scheme
}

// Injectable identifies objects that can receive a Client
type Injectable interface {
	InjectClient(Client)
}

// Provider is used to provide a new instance of the Client each time it's required
type Provider struct {
	Get func() (Client, error)
}

type defaultClient struct {
	controller.Client
	kubernetes.Interface
	scheme *runtime.Scheme
}

func (c *defaultClient) GetScheme() *runtime.Scheme {
	return c.scheme
}

// NewOutOfClusterClient creates a new k8s client that can be used from outside the cluster
func NewOutOfClusterClient(kubeconfig string) (Client, error) {
	initialize(kubeconfig)
	// using fast discovery from outside the cluster
	return NewClient(true)
}

// NewClient creates a new k8s client that can be used from outside or in the cluster
func NewClient(fastDiscovery bool) (Client, error) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	scheme := clientscheme.Scheme

	// Setup Scheme for all resources
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, err
	}

	var clientset kubernetes.Interface
	if clientset, err = kubernetes.NewForConfig(cfg); err != nil {
		return nil, err
	}

	var mapper meta.RESTMapper
	if fastDiscovery {
		mapper = newFastDiscoveryRESTMapper(cfg)
	}

	// Create a new client to avoid using cache (enabled by default on operator-sdk client)
	clientOptions := controller.Options{
		Scheme: scheme,
		Mapper: mapper,
	}
	dynClient, err := controller.New(cfg, clientOptions)
	if err != nil {
		return nil, err
	}

	return &defaultClient{
		Client:    dynClient,
		Interface: clientset,
		scheme:    clientOptions.Scheme,
	}, nil
}

// FromManager creates a new k8s client from a manager object
func FromManager(manager manager.Manager) (Client, error) {
	var err error
	var clientset kubernetes.Interface
	if clientset, err = kubernetes.NewForConfig(manager.GetConfig()); err != nil {
		return nil, err
	}
	return &defaultClient{
		Client:    manager.GetClient(),
		Interface: clientset,
		scheme:    manager.GetScheme(),
	}, nil
}

// init initialize the k8s client for usage outside the cluster
func initialize(kubeconfig string) {
	if kubeconfig == "" {
		kubeconfig = getDefaultKubeConfigFile()
	}
	os.Setenv(k8sutil.KubeConfigEnvVar, kubeconfig)
}

func getDefaultKubeConfigFile() string {
	usr, err := user.Current()
	if err != nil {
		panic(err) // TODO handle error
	}
	return filepath.Join(usr.HomeDir, ".kube", "config")
}

// GetCurrentNamespace --
func GetCurrentNamespace(kubeconfig string) (string, error) {
	if kubeconfig == "" {
		kubeconfig = getDefaultKubeConfigFile()
	}
	if kubeconfig == "" {
		return "default", nil
	}

	data, err := ioutil.ReadFile(kubeconfig)
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

	clientcmdconfig := decoded.(*clientcmdapi.Config)

	cc := clientcmd.NewDefaultClientConfig(*clientcmdconfig, &clientcmd.ConfigOverrides{})
	ns, _, err := cc.Namespace()
	return ns, err
}
