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

package cli

import (
	"github.com/temporalio/tctl-kit/pkg/flags"
	"github.com/urfave/cli/v2"
)

func newBatchCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "describe",
			Usage: "Describe a batch operation job",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:     FlagJobID,
					Usage:    "Batch Job Id",
					Required: true,
				},
			}, flags.FlagsForRendering...),
			Action: func(c *cli.Context) error {
				return DescribeBatchJob(c)
			},
		},
		{
			Name:      "list",
			Usage:     "List batch operation jobs",
			Flags:     flags.FlagsForPaginationAndRendering,
			ArgsUsage: " ",
			Action: func(c *cli.Context) error {
				return ListBatchJobs(c)
			},
		},
		{
			Name:  "signal",
			Usage: "Signal a batch of Workflow Executions",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagQuery,
					Aliases:  FlagQueryAlias,
					Usage:    "Specify the Workflow Executions to operate on. See https://docs.temporal.io/docs/tctl/workflow/list#--query for details",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagName,
					Usage:    "Signal name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    FlagInput,
					Aliases: FlagInputAlias,
					Usage:   "Input for the signal (JSON)",
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Usage:    "Reason to signal",
					Required: true,
				},
				&cli.BoolFlag{
					Name:    FlagYes,
					Aliases: FlagYesAlias,
					Usage:   "Confirm all prompts",
				},
			},
			Action: func(c *cli.Context) error {
				return BatchTerminate(c)
			},
		},
		{
			Name:  "terminate",
			Usage: "Terminate a batch of Workflow Executions",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagQuery,
					Aliases:  FlagQueryAlias,
					Usage:    "Specify the Workflow Executions to operate on. See https://docs.temporal.io/docs/tctl/workflow/list#--query for details",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Usage:    "Reason to terminate",
					Required: true,
				},
				&cli.BoolFlag{
					Name:    FlagYes,
					Aliases: FlagYesAlias,
					Usage:   "Confirm all prompts",
				},
			},
			Action: func(c *cli.Context) error {
				return BatchTerminate(c)
			},
		},
		{
			Name:  "cancel",
			Usage: "Cancel a batch of Workflow Executions",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagQuery,
					Aliases:  FlagQueryAlias,
					Usage:    "Specify the Workflow Executions to operate on. See https://docs.temporal.io/docs/tctl/workflow/list#--query for details",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Usage:    "Reason to cancel",
					Required: true,
				},
				&cli.BoolFlag{
					Name:    FlagYes,
					Aliases: FlagYesAlias,
					Usage:   "Confirm all prompts",
				},
			},
			Action: func(c *cli.Context) error {
				return BatchCancel(c)
			},
		},
		{
			Name:  "stop",
			Usage: "Stop a batch operation job",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagJobID,
					Usage:    "Batch Job Id",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Usage:    "Reason to stop the batch job",
					Required: true,
				},
			},
			Action: func(c *cli.Context) error {
				return StopBatchJob(c)
			},
		},
	}
}
