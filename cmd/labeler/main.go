// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package main

import (
	"fmt"
	"github.com/otcshare/intel-ethernet-operator/pkg/labeler"
	"os"
)

func main() {
	if err := labeler.DeviceDiscovery(); err != nil {
		fmt.Printf("Device discovery failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Device discovery finished successfully\n")

	os.Exit(0)
}
