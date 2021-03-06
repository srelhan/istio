// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"istio.io/istio/galley/pkg/kube"
	"istio.io/istio/galley/pkg/kube/converter"
	"istio.io/istio/galley/pkg/meshconfig"
	kube_meta "istio.io/istio/galley/pkg/metadata/kube"
	"istio.io/istio/galley/pkg/runtime"
	"istio.io/istio/galley/pkg/testing/mock"
	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/mcp/server"
	mcptestmon "istio.io/istio/pkg/mcp/testing/monitoring"
)

func TestNewServer_Errors(t *testing.T) {

loop:
	for i := 0; ; i++ {
		p := defaultPatchTable()
		mk := mock.NewKube()
		p.newKubeFromConfigFile = func(string) (kube.Interfaces, error) { return mk, nil }
		p.newSource = func(kube.Interfaces, time.Duration, *kube.Schema, *converter.Config) (runtime.Source, error) {
			return runtime.NewInMemorySource(), nil
		}
		p.newMeshConfigCache = func(path string) (meshconfig.Cache, error) { return meshconfig.NewInMemory(), nil }
		p.fsNew = func(string, *kube.Schema, *converter.Config) (runtime.Source, error) {
			return runtime.NewInMemorySource(), nil
		}
		p.mcpMetricReporter = func(string) server.Reporter {
			return nil
		}

		e := errors.New("err")

		args := DefaultArgs()
		args.APIAddress = "tcp://0.0.0.0:0"
		args.Insecure = true

		switch i {
		case 0:
			p.logConfigure = func(*log.Options) error { return e }
		case 1:
			p.newKubeFromConfigFile = func(string) (kube.Interfaces, error) { return nil, e }
		case 2:
			p.newSource = func(kube.Interfaces, time.Duration, *kube.Schema, *converter.Config) (runtime.Source, error) {
				return nil, e
			}
		case 3:
			p.netListen = func(network, address string) (net.Listener, error) { return nil, e }
		case 4:
			p.newMeshConfigCache = func(path string) (meshconfig.Cache, error) { return nil, e }
		case 5:
			args.ConfigPath = "aaa"
			p.fsNew = func(string, *kube.Schema, *converter.Config) (runtime.Source, error) { return nil, e }
		default:
			break loop
		}

		_, err := newServer(args, p, false)
		if err == nil {
			t.Fatalf("Expected error not found for i=%d", i)
		}
	}
}

func TestNewServer(t *testing.T) {
	p := defaultPatchTable()
	mk := mock.NewKube()
	p.newKubeFromConfigFile = func(string) (kube.Interfaces, error) { return mk, nil }
	p.newSource = func(kube.Interfaces, time.Duration, *kube.Schema, *converter.Config) (runtime.Source, error) {
		return runtime.NewInMemorySource(), nil
	}
	p.mcpMetricReporter = func(s string) server.Reporter {
		return mcptestmon.NewInMemoryServerStatsContext()
	}
	p.newMeshConfigCache = func(path string) (meshconfig.Cache, error) { return meshconfig.NewInMemory(), nil }
	p.fsNew = func(string, *kube.Schema, *converter.Config) (runtime.Source, error) {
		return runtime.NewInMemorySource(), nil
	}
	p.verifyResourceTypesPresence = func(kube.Interfaces) error {
		return nil
	}

	args := DefaultArgs()
	args.APIAddress = "tcp://0.0.0.0:0"
	args.Insecure = true

	typeCount := len(kube_meta.Types.All())
	tests := []struct {
		name              string
		convertK8SService bool
		wantListeners     int
	}{
		{
			name:          "Simple",
			wantListeners: typeCount - 1,
		},
		{
			name:              "ConvertK8SService",
			convertK8SService: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := newServer(args, p, test.convertK8SService)
			if err != nil {
				t.Fatalf("Unexpected error creating service: %v", err)
			}
			_ = s.Close()
		})
	}
}

func TestServer_Basic(t *testing.T) {
	p := defaultPatchTable()
	mk := mock.NewKube()
	p.newKubeFromConfigFile = func(string) (kube.Interfaces, error) { return mk, nil }
	p.newSource = func(kube.Interfaces, time.Duration, *kube.Schema, *converter.Config) (runtime.Source, error) {
		return runtime.NewInMemorySource(), nil
	}
	p.mcpMetricReporter = func(s string) server.Reporter {
		return mcptestmon.NewInMemoryServerStatsContext()
	}
	p.newMeshConfigCache = func(path string) (meshconfig.Cache, error) { return meshconfig.NewInMemory(), nil }
	p.verifyResourceTypesPresence = func(kube.Interfaces) error {
		return nil
	}

	args := DefaultArgs()
	args.APIAddress = "tcp://0.0.0.0:0"
	args.Insecure = true
	s, err := newServer(args, p, false)
	if err != nil {
		t.Fatalf("Unexpected error creating service: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		s.Run()
	}()

	wg.Wait()
	_ = s.Close()
}
