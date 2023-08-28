// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package main

import (
	"fmt"
	"os"

	"github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/labeler"
)

func main() {
	if err := labeler.DeviceDiscovery(); err != nil {
		fmt.Printf("Device discovery failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Device discovery finished successfully\n")

	os.Exit(0)
}
