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
	"strings"

	"github.com/urfave/cli/v2"

	"go.temporal.io/server/service/worker/batcher"
)

var allBatchTypes = []string{batcher.BatchTypeTerminate, batcher.BatchTypeCancel, batcher.BatchTypeSignal}

func newBatchCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "describe",
			Usage: "Describe a batch operation job",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagJobID,
					Aliases:  FlagJobIDAlias,
					Usage:    "Batch Job Id",
					Required: true,
				},
			},
			Action: func(c *cli.Context) error {
				return DescribeBatchJob(c)
			},
		},
		{
			Name:  "list",
			Usage: "List batch operation jobs",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:  FlagPageSize,
					Value: 30,
					Usage: "Result page size",
				},
			},
			Action: func(c *cli.Context) error {
				return ListBatchJobs(c)
			},
		},
		{
			Name:  "start",
			Usage: "Start a batch operation job",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagListQuery,
					Aliases:  FlagListQueryAlias,
					Usage:    "Query to get workflows for being executed this batch operation",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Aliases:  FlagReasonAlias,
					Usage:    "Reason to run this batch job",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagBatchType,
					Usage:    "Types supported: " + strings.Join(allBatchTypes, ","),
					Required: true,
				},
				&cli.StringFlag{
					Name:  FlagSignalName,
					Usage: "Required for batch signal",
				},
				&cli.StringFlag{
					Name:    FlagInput,
					Aliases: FlagInputAlias,
					Usage:   "Optional input of signal",
				},
				&cli.IntFlag{
					Name:  FlagRPS,
					Value: batcher.DefaultRPS,
					Usage: "RPS of processing",
				},
				&cli.BoolFlag{
					Name:  FlagYes,
					Usage: "Optional flag to disable confirmation prompt",
				},
			},
			Action: func(c *cli.Context) error {
				return StartBatchJob(c)
			},
		},
		{
			Name:  "terminate",
			Usage: "terminate a batch operation job",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagJobID,
					Aliases:  FlagJobIDAlias,
					Usage:    "Batch Job Id",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Aliases:  FlagReasonAlias,
					Usage:    "Reason to stop this batch job",
					Required: true,
				},
			},
			Action: func(c *cli.Context) error {
				return TerminateBatchJob(c)
			},
		},
	}
}
