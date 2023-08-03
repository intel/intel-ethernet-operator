// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package daemon

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/utils"
)

var (
	maxFileSize           = int64(10)              // Maximum update file size in kilobytes
	updateXMLParseTimeout = 100 * time.Millisecond // Update xml parse timeout
)

func isRebootRequired(path string) (bool, error) {
	invf, err := utils.OpenNoLinks(path)
	if err != nil {
		return true, err
	}
	defer invf.Close()

	stat, err := invf.Stat()
	if err != nil {
		return true, err
	}

	kSize := stat.Size() / 1024
	if kSize > maxFileSize {
		return true, errors.New("Update result xml file too large: " + strconv.Itoa(int(kSize)) + "kB")
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateXMLParseTimeout)
	defer cancel()

	rebootRequired := false
	decoder := xml.NewDecoder(invf)
	for {
		select {
		case <-ctx.Done():
			cancel()
			return rebootRequired, ctx.Err()
		default:
			token, err := decoder.Token()
			if token == nil {
				return rebootRequired, nil
			}
			if err != nil {
				if err == io.EOF {
					return rebootRequired, nil
				}
				return false, err
			}

			switch t := token.(type) {
			case xml.StartElement:
				if t.Name.Local == "RebootRequired" {
					var r int
					err := decoder.DecodeElement(&r, &t)
					if err != nil {
						return false, err
					}
					rebootRequired = (r != 0)
				}
			}
		}
	}
}
