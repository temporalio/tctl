// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/temporalio/tctl/cli"
	"github.com/temporalio/tctl/cli_curr"
	"github.com/temporalio/tctl/config"
)

const deprecationNotice = ("DEPRECATION NOTICE: tctl will enter End of Support September 30, 2025. " +
	"Please transition to Temporal CLI (https://docs.temporal.io/cli).")

// See https://docs.temporal.io/tctl/ for usage
func main() {
	tctlConfig, _ := config.NewTctlConfig()
	version := tctlConfig.Version

	os.Stderr.WriteString(color.RedString(deprecationNotice) + "\n\n")

	var err error
	if version == "next" || version == "2" {
		appNext := cli.NewCliApp()
		err = appNext.Run(os.Args)
	} else {
		app := cli_curr.NewCliApp()
		err = app.Run(os.Args)
	}

	if err != nil {
		log.Fatal(err)
	}
}
