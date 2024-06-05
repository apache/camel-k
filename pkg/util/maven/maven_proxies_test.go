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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHTTPEnvVar(t *testing.T) {

	testcases := []struct {
		name                string
		proxyHTTPEnvVar     string
		proxyHostResult     string
		proxyPortResult     string
		proxyProtocolResult string
		proxyUsernameResult string
		proxyPasswordResult string
	}{
		{
			name:                "HTTP_PROXY_Port",
			proxyHTTPEnvVar:     "http://proxy.namespace.svc:8080",
			proxyHostResult:     "proxy.namespace.svc",
			proxyPortResult:     "8080",
			proxyProtocolResult: "http",
			proxyUsernameResult: "",
			proxyPasswordResult: "",
		},
		{
			name:                "HTTP_PROXY_No_Port",
			proxyHTTPEnvVar:     "http://proxy.namespace.svc",
			proxyHostResult:     "proxy.namespace.svc",
			proxyPortResult:     "80",
			proxyProtocolResult: "http",
			proxyUsernameResult: "",
			proxyPasswordResult: "",
		},
		{
			name:                "HTTP_PROXY_full",
			proxyHTTPEnvVar:     "http://user:password@proxy.namespace.svc:8081",
			proxyHostResult:     "proxy.namespace.svc",
			proxyPortResult:     "8081",
			proxyProtocolResult: "http",
			proxyUsernameResult: "user",
			proxyPasswordResult: "password",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := NewSettings(DefaultRepositories, ProxyFromEnvironment)
			require.NoError(t, err)

			t.Setenv("HTTP_PROXY", tc.proxyHTTPEnvVar)
			err = ProxyFromEnvironment.apply(&settings)

			require.NoError(t, err)
			assert.Equal(t, "http-proxy", settings.Proxies[0].ID)
			assert.Equal(t, true, settings.Proxies[0].Active)
			assert.Equal(t, tc.proxyHostResult, settings.Proxies[0].Host)
			assert.Equal(t, tc.proxyPortResult, settings.Proxies[0].Port)
			assert.Equal(t, tc.proxyProtocolResult, settings.Proxies[0].Protocol)
			assert.Equal(t, tc.proxyUsernameResult, settings.Proxies[0].Username)
			assert.Equal(t, tc.proxyPasswordResult, settings.Proxies[0].Password)
		})
	}
}

func TestProxyHTTPSEnvVar(t *testing.T) {

	testcases := []struct {
		name                string
		proxyHTTPSEnvVar    string
		proxyHostResult     string
		proxyPortResult     string
		proxyProtocolResult string
		proxyUsernameResult string
		proxyPasswordResult string
	}{
		{
			name:                "HTTPS_PROXY_Port",
			proxyHTTPSEnvVar:    "https://proxy.namespace.svc:8443",
			proxyHostResult:     "proxy.namespace.svc",
			proxyPortResult:     "8443",
			proxyProtocolResult: "https",
			proxyUsernameResult: "",
			proxyPasswordResult: "",
		},
		{
			name:                "HTTPS_PROXY_No_Port",
			proxyHTTPSEnvVar:    "https://proxy.namespace.svc",
			proxyHostResult:     "proxy.namespace.svc",
			proxyPortResult:     "443",
			proxyProtocolResult: "https",
			proxyUsernameResult: "",
			proxyPasswordResult: "",
		},
		{
			name:                "HTTP_PROXY_full",
			proxyHTTPSEnvVar:    "https://user:password@proxy.namespace.svc:8444",
			proxyHostResult:     "proxy.namespace.svc",
			proxyPortResult:     "8444",
			proxyProtocolResult: "https",
			proxyUsernameResult: "user",
			proxyPasswordResult: "password",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := NewSettings(DefaultRepositories, ProxyFromEnvironment)
			require.NoError(t, err)

			t.Setenv("HTTPS_PROXY", tc.proxyHTTPSEnvVar)
			err = ProxyFromEnvironment.apply(&settings)

			require.NoError(t, err)
			assert.Equal(t, "https-proxy", settings.Proxies[0].ID)
			assert.Equal(t, true, settings.Proxies[0].Active)
			assert.Equal(t, tc.proxyHostResult, settings.Proxies[0].Host)
			assert.Equal(t, tc.proxyPortResult, settings.Proxies[0].Port)
			assert.Equal(t, tc.proxyProtocolResult, settings.Proxies[0].Protocol)
			assert.Equal(t, tc.proxyUsernameResult, settings.Proxies[0].Username)
			assert.Equal(t, tc.proxyPasswordResult, settings.Proxies[0].Password)
		})
	}
}

func TestNoPROXYEnvVar(t *testing.T) {
	testcases := []struct {
		name             string
		proxyHTTPEnvVar  string
		proxyHTTPSEnvVar string
		noPROXYEnvVar    string
		noProxyResult    string
	}{
		{
			name:             "Valid_NOPROXY_simple",
			proxyHTTPEnvVar:  "http://www.proxy.com",
			proxyHTTPSEnvVar: "",
			noPROXYEnvVar:    "www.no.proxy.com",
			noProxyResult:    "www.no.proxy.com",
		},
		{
			name:             "Valid_NOPROXY_IPS",
			proxyHTTPEnvVar:  "http://www.proxy.com",
			proxyHTTPSEnvVar: "",
			noPROXYEnvVar:    "10.96.0.1,10.96.0.4",
			noProxyResult:    "10.96.0.1|10.96.0.4",
		},
		{
			name:             "Valid_NOPROXY_Complexe",
			proxyHTTPEnvVar:  "",
			proxyHTTPSEnvVar: "https://www.proxy.com",
			noPROXYEnvVar:    "localhost, 127.0.0.1, *.local, .my-co.com",
			noProxyResult:    "localhost|127.0.0.1|*.local|*.my-co.com",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := NewSettings(DefaultRepositories, ProxyFromEnvironment)
			require.NoError(t, err)

			if tc.proxyHTTPEnvVar != "" {
				t.Setenv("HTTP_PROXY", tc.proxyHTTPEnvVar)
			}

			if tc.proxyHTTPSEnvVar != "" {
				t.Setenv("HTTPS_PROXY", tc.proxyHTTPSEnvVar)
			}

			t.Setenv("NO_PROXY", tc.noPROXYEnvVar)

			err = ProxyFromEnvironment.apply(&settings)

			require.NoError(t, err)
			assert.Equal(t, true, settings.Proxies[0].Active)
			assert.Equal(t, tc.noProxyResult, settings.Proxies[0].NonProxyHosts)
		})
	}
}

func TestAllProxyEnvVar(t *testing.T) {
	t.Run("All proxy env vars", func(t *testing.T) {

		settings, err := NewSettings(DefaultRepositories, ProxyFromEnvironment)
		require.NoError(t, err)

		t.Setenv("HTTP_PROXY", "http://www.unsercure-proxy.com")
		t.Setenv("HTTPS_PROXY", "https://www.sercure-proxy.com")
		t.Setenv("NO_PROXY", "localhost, 10.96.0.1, *.local")

		err = ProxyFromEnvironment.apply(&settings)

		require.NoError(t, err)
		assert.Equal(t, 2, len(settings.Proxies))
		assert.Equal(t, "http-proxy", settings.Proxies[0].ID)
		assert.Equal(t, "https-proxy", settings.Proxies[1].ID)
		assert.Equal(t, "localhost|10.96.0.1|*.local", settings.Proxies[0].NonProxyHosts)
		assert.Equal(t, "localhost|10.96.0.1|*.local", settings.Proxies[1].NonProxyHosts)
	})
}

func TestAddAnotherProxyEnvVar(t *testing.T) {
	t.Run("Add a proxy from env vars", func(t *testing.T) {

		settings, err := NewSettings(DefaultRepositories, ProxyFromEnvironment)
		require.NoError(t, err)

		settings.Proxies = append(settings.Proxies, Proxy{
			ID:       "other",
			Active:   false,
			Protocol: "http",
			Host:     "otherproxy.com",
			Port:     "8088",
		})

		t.Setenv("HTTP_PROXY", "http://www.unsercure-proxy.com")

		err = ProxyFromEnvironment.apply(&settings)

		require.NoError(t, err)
		assert.Equal(t, 2, len(settings.Proxies))
		assert.Equal(t, "other", settings.Proxies[0].ID)
		assert.Equal(t, "http-proxy", settings.Proxies[1].ID)
	})
}
