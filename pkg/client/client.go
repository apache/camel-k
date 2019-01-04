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
	"github.com/apache/camel-k/pkg/apis"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/user"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Client is an abstraction for a k8s client
type Client interface {
	controller.Client
	kubernetes.Interface
}

// Injectable identifies objects that can receive a Client
type Injectable interface {
	InjectClient(Client)
}

type defaultClient struct {
	controller.Client
	kubernetes.Interface
}

// NewOutOfClusterClient creates a new k8s client that can be used from outside the cluster
func NewOutOfClusterClient(kubeconfig string, namespace string) (Client, error) {
	initialize(kubeconfig)
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	return FromManager(mgr)
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
	}, nil
}

// init initialize the k8s client for usage outside the cluster
func initialize(kubeconfig string) error {
	if kubeconfig == "" {
		kubeconfig = getDefaultKubeConfigFile()
	}
	os.Setenv(k8sutil.KubeConfigEnvVar, kubeconfig)
	return nil
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
