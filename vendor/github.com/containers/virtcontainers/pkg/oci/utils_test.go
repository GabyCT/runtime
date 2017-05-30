//
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
//

package oci

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	vc "github.com/containers/virtcontainers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const tempBundlePath = "/tmp/virtc/ocibundle/"
const containerID = "virtc-oci-test"
const consolePath = "/tmp/virtc/console"
const fileMode = os.FileMode(0640)
const dirMode = os.FileMode(0750)

func createConfig(fileName string, fileData string) (string, error) {
	configPath := path.Join(tempBundlePath, fileName)

	err := ioutil.WriteFile(configPath, []byte(fileData), fileMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create config file %s %v\n", configPath, err)
		return "", err
	}

	return configPath, nil
}

func TestMinimalPodConfig(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	runtimeConfig := RuntimeConfig{
		HypervisorType: vc.QemuHypervisor,
		AgentType:      vc.HyperstartAgent,
		ProxyType:      vc.CCProxyType,
		ShimType:       vc.CCShimType,
		Console:        consolePath,
	}

	expectedCmd := vc.Cmd{
		Args: []string{"sh"},
		Envs: []vc.EnvVar{
			{
				Var:   "PATH",
				Value: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			},
			{
				Var:   "TERM",
				Value: "xterm",
			},
		},
		WorkDir:             "/",
		User:                "0",
		PrimaryGroup:        "0",
		SupplementaryGroups: []string{"10", "29"},
		Interactive:         true,
		Console:             consolePath,
	}

	expectedContainerConfig := vc.ContainerConfig{
		ID:             containerID,
		RootFs:         path.Join(tempBundlePath, "rootfs"),
		ReadonlyRootfs: true,
		Cmd:            expectedCmd,
		Annotations: map[string]string{
			ConfigPathKey:    configPath,
			BundlePathKey:    tempBundlePath,
			ContainerTypeKey: string(vc.PodSandbox),
		},
	}

	expectedNetworkConfig := vc.NetworkConfig{
		NumInterfaces: 1,
	}

	expectedPodConfig := vc.PodConfig{
		ID: fmt.Sprintf("%s%s", PodIDPrefix, containerID),

		HypervisorType: vc.QemuHypervisor,
		AgentType:      vc.HyperstartAgent,
		ProxyType:      vc.CCProxyType,
		ShimType:       vc.CCShimType,

		NetworkModel:  vc.CNMNetworkModel,
		NetworkConfig: expectedNetworkConfig,

		Containers: []vc.ContainerConfig{expectedContainerConfig},

		Annotations: map[string]string{
			ConfigPathKey: configPath,
			BundlePathKey: tempBundlePath,
		},
	}

	ociSpec, err := ParseConfigJSON(tempBundlePath)
	if err != nil {
		t.Fatalf("Could not parse config.json: %v", err)
	}

	podConfig, err := PodConfig(ociSpec, runtimeConfig, tempBundlePath, containerID, consolePath)
	if err != nil {
		t.Fatalf("Could not create Pod configuration %v", err)
	}

	if reflect.DeepEqual(podConfig, expectedPodConfig) == false {
		t.Fatalf("Got %v\n expecting %v", podConfig, expectedPodConfig)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func testStatusToOCIStateSuccessful(t *testing.T, cStatus vc.ContainerStatus, expected specs.State) {
	ociState, err := StatusToOCIState(cStatus)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(ociState, expected) == false {
		t.Fatalf("Got %v\n expecting %v", ociState, expected)
	}
}

func TestStatusToOCIStateSuccessfulWithReadyState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	state := vc.State{
		State: vc.StateReady,
	}

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		State:       state,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "created",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStatusToOCIStateSuccessfulWithRunningState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	state := vc.State{
		State: vc.StateRunning,
	}

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		State:       state,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "running",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStatusToOCIStateSuccessfulWithStoppedState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	state := vc.State{
		State: vc.StateStopped,
	}

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		State:       state,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "stopped",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStatusToOCIStateSuccessfulWithNoState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStateToOCIState(t *testing.T) {
	var state vc.State

	if ociState := stateToOCIState(state); ociState != "" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}

	state.State = vc.StateReady
	if ociState := stateToOCIState(state); ociState != "created" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}

	state.State = vc.StateRunning
	if ociState := stateToOCIState(state); ociState != "running" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}

	state.State = vc.StateStopped
	if ociState := stateToOCIState(state); ociState != "stopped" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}
}

func TestEnvVars(t *testing.T) {
	envVars := []string{"foo=bar", "TERM=xterm", "HOME=/home/foo", "TERM=\"bar\"", "foo=\"\""}
	expectecVcEnvVars := []vc.EnvVar{
		{
			Var:   "foo",
			Value: "bar",
		},
		{
			Var:   "TERM",
			Value: "xterm",
		},
		{
			Var:   "HOME",
			Value: "/home/foo",
		},
		{
			Var:   "TERM",
			Value: "\"bar\"",
		},
		{
			Var:   "foo",
			Value: "\"\"",
		},
	}

	vcEnvVars, err := EnvVars(envVars)
	if err != nil {
		t.Fatalf("Could not create environment variable slice %v", err)
	}

	if reflect.DeepEqual(vcEnvVars, expectecVcEnvVars) == false {
		t.Fatalf("Got %v\n expecting %v", vcEnvVars, expectecVcEnvVars)
	}
}

func TestMalformedEnvVars(t *testing.T) {
	envVars := []string{"foo"}
	r, err := EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}

	envVars = []string{"TERM="}
	r, err = EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}

	envVars = []string{"=foo"}
	r, err = EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}

	envVars = []string{"=foo="}
	r, err = EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}
}

func TestGetConfigPath(t *testing.T) {
	expected := filepath.Join(tempBundlePath, "config.json")

	configPath := getConfigPath(tempBundlePath)

	if configPath != expected {
		t.Fatalf("Got %s, Expecting %s", configPath, expected)
	}
}

func testGetContainerTypeSuccessful(t *testing.T, annotations map[string]string, expected vc.ContainerType) {
	containerType, err := GetContainerType(annotations)
	if err != nil {
		t.Fatal(err)
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}
}

func TestGetContainerTypePodSandbox(t *testing.T) {
	annotations := map[string]string{
		ContainerTypeKey: string(vc.PodSandbox),
	}

	testGetContainerTypeSuccessful(t, annotations, vc.PodSandbox)
}

func TestGetContainerTypePodContainer(t *testing.T) {
	annotations := map[string]string{
		ContainerTypeKey: string(vc.PodContainer),
	}

	testGetContainerTypeSuccessful(t, annotations, vc.PodContainer)
}

func TestGetContainerTypeFailure(t *testing.T) {
	expected := vc.UnknownContainerType

	containerType, err := GetContainerType(map[string]string{})
	if err == nil {
		t.Fatalf("This test should fail because annotations is empty")
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}
}

func testContainerTypeSuccessful(t *testing.T, ociSpec CompatOCISpec, expected vc.ContainerType, expectedAnnotationFound bool) {
	containerType, cTypeAnnotationFound, err := ociSpec.ContainerType()
	if err != nil {
		t.Fatal(err)
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}

	if cTypeAnnotationFound != expectedAnnotationFound {
		t.Fatalf("Got %t, Expecting %t", cTypeAnnotationFound, expectedAnnotationFound)
	}
}

func TestContainerTypePodSandbox(t *testing.T) {
	var ociSpec CompatOCISpec

	ociSpec.Annotations = map[string]string{
		CRIOContainerTypeKey: ContainerTypePod,
	}

	testContainerTypeSuccessful(t, ociSpec, vc.PodSandbox, true)
}

func TestContainerTypePodContainer(t *testing.T) {
	var ociSpec CompatOCISpec

	ociSpec.Annotations = map[string]string{
		CRIOContainerTypeKey: ContainerTypeContainer,
	}

	testContainerTypeSuccessful(t, ociSpec, vc.PodContainer, true)
}

func TestContainerTypePodSandboxEmptyAnnotation(t *testing.T) {
	testContainerTypeSuccessful(t, CompatOCISpec{}, vc.PodSandbox, false)
}

func TestContainerTypeFailure(t *testing.T) {
	var ociSpec CompatOCISpec
	expected := vc.UnknownContainerType
	unknownType := "unknown_type"

	ociSpec.Annotations = map[string]string{
		CRIOContainerTypeKey: unknownType,
	}

	containerType, cTypeAnnotationFound, err := ociSpec.ContainerType()
	if err == nil {
		t.Fatalf("This test should fail because the container type is %s", unknownType)
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}

	if cTypeAnnotationFound != true {
		t.Fatalf("Got %t, Expecting %t", cTypeAnnotationFound, true)
	}
}

func TestPodIDSuccessful(t *testing.T) {
	var ociSpec CompatOCISpec
	testPodID := "testPodID"

	ociSpec.Annotations = map[string]string{
		CRIOSandboxNameKey: testPodID,
	}

	podID, err := ociSpec.PodID()
	if err != nil {
		t.Fatal(err)
	}

	if podID != testPodID {
		t.Fatalf("Got %s, Expecting %s", podID, testPodID)
	}
}

func TestPodIDFailure(t *testing.T) {
	var ociSpec CompatOCISpec

	podID, err := ociSpec.PodID()
	if err == nil {
		t.Fatalf("This test should fail because annotations is empty")
	}

	if podID != "" {
		t.Fatalf("Got %s, Expecting empty pod ID", podID)
	}
}

func TestMain(m *testing.M) {
	/* Create temp bundle directory if necessary */
	err := os.MkdirAll(tempBundlePath, dirMode)
	if err != nil {
		fmt.Printf("Unable to create %s %v\n", tempBundlePath, err)
		os.Exit(1)
	}

	defer os.RemoveAll(tempBundlePath)

	os.Exit(m.Run())
}
