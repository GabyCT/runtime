// Copyright (c) 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/BurntSushi/toml"
	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
)

const (
	defaultRuntimeLib           = "/var/lib/clear-containers"
	defaultRuntimeRun           = "/var/run/clear-containers"
	defaultRuntimeConfiguration = "/etc/clear-containers/configuration.toml"
	defaultHypervisor           = vc.QemuHypervisor
	defaultProxy                = vc.CCProxyType
	defaultShim                 = vc.CCShimType
	defaultAgent                = vc.HyperstartAgent
	defaultKernelPath           = "/usr/share/clear-containers/vmlinux.container"
	defaultImagePath            = "/usr/share/clear-containers/clear-containers.img"
	defaultHypervisorPath       = "/usr/bin/qemu-lite-system-x86_64"
	defaultProxyURL             = "unix:///run/cc-oci-runtime/proxy.sock"
	defaultPauseRootPath        = "/var/lib/clear-containers/runtime/bundles/pause_bundle"
	pauseBinRelativePath        = "bin/pause"
)

var defaultShimPath = "/usr/local/libexec/cc-shim"

const (
	qemuLite = "qemu-lite"
	qemu     = "qemu"
)

const (
	ccProxy = "cc"
)

const (
	ccShim = "cc"
)

const (
	hyperstartAgent = "hyperstart"
)

var (
	errUnknownHypervisor  = errors.New("unknown hypervisor")
	errUnknownAgent       = errors.New("unknown agent")
	errTooManyHypervisors = errors.New("too many hypervisor sections")
	errTooManyProxies     = errors.New("too many proxy sections")
	errTooManyShims       = errors.New("too many shim sections")
	errTooManyAgents      = errors.New("too many agent sections")
)

type tomlConfig struct {
	Hypervisor map[string]hypervisor
	Proxy      map[string]proxy
	Shim       map[string]shim
	Agent      map[string]agent
	Runtime    runtime
}

type hypervisor struct {
	Path   string
	Kernel string
	Image  string
}

type proxy struct {
	URL string `toml:"url"`
}

type runtime struct {
	GlobalLogPath string `toml:"global_log_path"`
}

type shim struct {
	Path string `toml:"path"`
}

type agent struct {
	PauseRootPath string `toml:"pause_root_path"`
}

func (h hypervisor) path() string {
	if h.Path == "" {
		return defaultHypervisorPath
	}

	return h.Path
}

func (h hypervisor) kernel() string {
	if h.Kernel == "" {
		return defaultKernelPath
	}

	return h.Kernel
}

func (h hypervisor) image() string {
	if h.Image == "" {
		return defaultImagePath
	}

	return h.Image
}

func (p proxy) url() string {
	if p.URL == "" {
		return defaultProxyURL
	}

	return p.URL
}

func (s shim) path() string {
	if s.Path == "" {
		return defaultShimPath
	}

	return s.Path
}

func (a agent) pauseRootPath() string {
	if a.PauseRootPath == "" {
		return defaultPauseRootPath
	}

	return a.PauseRootPath
}

func checkConfigParams(tomlConf tomlConfig) error {
	if len(tomlConf.Hypervisor) > 1 {
		return errTooManyHypervisors
	}

	if len(tomlConf.Proxy) > 1 {
		return errTooManyProxies
	}

	if len(tomlConf.Shim) > 1 {
		return errTooManyShims
	}

	if len(tomlConf.Agent) > 1 {
		return errTooManyAgents
	}

	return nil
}

func newQemuHypervisorConfig(h hypervisor) (vc.HypervisorConfig, error) {
	hypervisor := h.path()
	kernel := h.kernel()
	image := h.image()

	for _, file := range []string{hypervisor, kernel, image} {
		if !fileExists(file) {
			return vc.HypervisorConfig{},
				fmt.Errorf("File does not exist: %v", file)
		}
	}

	return vc.HypervisorConfig{
		HypervisorPath: hypervisor,
		KernelPath:     kernel,
		ImagePath:      image,
	}, nil
}

func newHyperstartAgentConfig(a agent) (vc.HyperConfig, error) {
	dir := a.pauseRootPath()

	if !fileExists(dir) {
		return vc.HyperConfig{}, fmt.Errorf("Directory does not exist: %v", dir)
	}

	path := filepath.Join(dir, pauseBinRelativePath)

	if !fileExists(path) {
		return vc.HyperConfig{}, fmt.Errorf("File does not exist: %v", path)
	}

	return vc.HyperConfig{
		PauseBinPath: path,
	}, nil
}

func newCCShimConfig(s shim) (vc.CCShimConfig, error) {
	path := s.path()

	if !fileExists(path) {
		return vc.CCShimConfig{}, fmt.Errorf("File does not exist: %v", path)
	}

	return vc.CCShimConfig{
		Path: path,
	}, nil
}

func updateRuntimeConfig(configPath string, tomlConf tomlConfig, config *oci.RuntimeConfig) error {
	for k, hypervisor := range tomlConf.Hypervisor {
		switch k {
		case qemu:
			fallthrough
		case qemuLite:
			hConfig, err := newQemuHypervisorConfig(hypervisor)
			if err != nil {
				return fmt.Errorf("%v: %v", configPath, err)
			}

			config.HypervisorConfig = hConfig

			break
		}
	}

	for k, proxy := range tomlConf.Proxy {
		switch k {
		case ccProxy:
			pConfig := vc.CCProxyConfig{
				URL: proxy.url(),
			}

			config.ProxyType = vc.CCProxyType
			config.ProxyConfig = pConfig

			break
		}
	}

	for k, agent := range tomlConf.Agent {
		switch k {
		case hyperstartAgent:
			agentConfig, err := newHyperstartAgentConfig(agent)
			if err != nil {
				return fmt.Errorf("%v: %v", configPath, err)
			}

			config.AgentConfig = agentConfig

			break
		}
	}

	for k, shim := range tomlConf.Shim {
		switch k {
		case ccShim:
			shConfig, err := newCCShimConfig(shim)
			if err != nil {
				return fmt.Errorf("%v: %v", configPath, err)
			}

			config.ShimType = vc.CCShimType
			config.ShimConfig = shConfig

			break
		}
	}

	return nil
}

func loadConfiguration(configPath string) (oci.RuntimeConfig, error) {
	defaultHypervisorConfig := vc.HypervisorConfig{
		HypervisorPath: defaultHypervisorPath,
		KernelPath:     defaultKernelPath,
		ImagePath:      defaultImagePath,
	}

	defaultAgentConfig := vc.HyperConfig{
		PauseBinPath: filepath.Join(defaultPauseRootPath,
			pauseBinRelativePath),
	}

	config := oci.RuntimeConfig{
		HypervisorType:   defaultHypervisor,
		HypervisorConfig: defaultHypervisorConfig,
		AgentType:        defaultAgent,
		AgentConfig:      defaultAgentConfig,
		ProxyType:        defaultProxy,
		ShimType:         defaultShim,
	}

	if configPath == "" {
		configPath = defaultRuntimeConfiguration
	}

	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	var tomlConf tomlConfig
	_, err = toml.Decode(string(configData), &tomlConf)
	if err != nil {
		return config, err
	}

	// The configuration file may have enabled global logging,
	// so handle that before any log calls.
	err = handleGlobalLog(tomlConf.Runtime.GlobalLogPath)
	if err != nil {
		return config, err
	}

	ccLog.Debugf("TOML configuration: %v", tomlConf)

	if err := checkConfigParams(tomlConf); err != nil {
		return config, err
	}

	if err := updateRuntimeConfig(configPath, tomlConf, &config); err != nil {
		return config, err
	}

	return config, nil
}
