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

package maven

import (
	"net/url"
	"os"
	"strings"
)

var ProxyFromEnvironment = proxyFromEnvironment{}

type proxyFromEnvironment struct{}

func (proxyFromEnvironment) apply(settings *Settings) error {
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		proxy, err := parseProxyFromEnvVar(httpProxy)
		if err != nil {
			return err
		}
		proxy.ID = "http-proxy"
		settings.Proxies = append(settings.Proxies, proxy)
	}

	if httpsProxy := os.Getenv("HTTPS_PROXY"); httpsProxy != "" {
		proxy, err := parseProxyFromEnvVar(httpsProxy)
		if err != nil {
			return err
		}
		proxy.ID = "https-proxy"
		settings.Proxies = append(settings.Proxies, proxy)
	}

	return nil
}

func parseProxyFromEnvVar(proxyEnvVar string) (Proxy, error) {
	u, err := url.Parse(proxyEnvVar)
	if err != nil {
		return Proxy{}, err
	}
	proxy := Proxy{
		Active:   true,
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Port:     u.Port(),
	}
	if proxy.Port == "" {
		switch proxy.Protocol {
		case "http":
			proxy.Port = "80"
		case "https":
			proxy.Port = "443"
		}
	}
	if user := u.User; user != nil {
		proxy.Username = user.Username()
		if password, set := user.Password(); set {
			proxy.Password = password
		}
	}
	if noProxy := os.Getenv("NO_PROXY"); noProxy != "" {
		// Convert to the format expected by the JVM http.nonProxyHosts system property
		hosts := strings.Split(strings.ReplaceAll(noProxy, " ", ""), ",")
		for i, host := range hosts {
			if strings.HasPrefix(host, ".") {
				hosts[i] = strings.Replace(host, ".", "*.", 1)
			}
		}
		proxy.NonProxyHosts = strings.Join(hosts, "|")
	}

	return proxy, nil
}
