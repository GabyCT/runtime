// Copyright (c) 2014,2015,2016 Docker, Inc.
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
	"encoding/json"
	"fmt"
	"os"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/urfave/cli"
)

var stateCommand = cli.Command{
	Name:  "state",
	Usage: "output the state of a container",
	ArgsUsage: `<container-id>

   <container-id> is your name for the instance of the container`,
	Description: `The state command outputs current state information for the
instance of a container.`,
	Action: func(context *cli.Context) error {
		args := context.Args()
		if len(args) != 1 {
			return fmt.Errorf("Expecting only one container ID, got %d: %v", len(args), []string(args))
		}

		return state(args.First())
	},
}

func state(containerID string) error {
	// Checks the MUST and MUST NOT from OCI runtime specification
	fullID, err := expandContainerID(containerID)
	if err != nil {
		return err
	}

	containerID = fullID

	podStatus, err := vc.StatusPod(containerID)
	if err != nil {
		return err
	}

	// Convert the status to the expected State structure
	state, err := oci.StatusToOCIState(podStatus)
	if err != nil {
		return err
	}

	// Update status of process
	running, err := processRunning(state.Pid)
	if err != nil {
		return err
	}
	if running == false {
		if err := stopContainer(podStatus); err != nil {
			return err
		}

		state.Status = "stopped"
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	// Print stateJSON to stdout
	fmt.Fprintf(os.Stdout, "%s", stateJSON)

	return nil
}
