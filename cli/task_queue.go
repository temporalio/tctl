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
	"github.com/temporalio/tctl-kit/pkg/output"
	"github.com/urfave/cli/v2"
)

func newTaskQueueCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:    "describe",
			Aliases: []string{"d"},
			Usage:   "Describe the Task Queue and its pollers",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:     FlagTaskQueue,
					Aliases:  FlagTaskQueueAlias,
					Usage:    "Task Queue name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    FlagTaskQueueType,
					Aliases: FlagTaskQueueTypeAlias,
					Value:   "workflow",
					Usage:   "Task Queue type [workflow|activity]",
				},
			}, flags.FlagsForRendering...),
			Action: func(c *cli.Context) error {
				return DescribeTaskQueue(c)
			},
		},
		{
			Name:    "list-partition",
			Aliases: []string{"lp"},
			Usage:   "List all the Task Queue partitions and the hostname for partitions",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagTaskQueue,
					Aliases:  FlagTaskQueueAlias,
					Usage:    "Task Queue name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    output.FlagOutput,
					Aliases: []string{"o"},
					Usage:   output.UsageText,
					Value:   string(output.Table),
				},
			},
			Action: func(c *cli.Context) error {
				return ListTaskQueuePartitions(c)
			},
		},
		{
			Name:    "update-versions",
			Aliases: []string{"uv"},
			Usage:   "Update the Task Queue's worker build id version graph",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagTaskQueue,
					Aliases:  FlagTaskQueueAlias,
					Usage:    "Task Queue name",
					Required: true,
				},
				&cli.StringFlag{
					Name:     FlagWorkerBuildId,
					Aliases:  FlagWorkerBuildIdAlias,
					Usage:    "Worker build id to add or modify",
					Required: true,
				},
				&cli.BoolFlag{
					Name:  FlagWorkerBuildIdMakeDefault,
					Usage: "If set, make this build id the default version for new workflows started on the task queue",
				},
				&cli.StringFlag{
					Name: FlagWorkerBuildIdPreviousCompat,
					Usage: "If set, indicates that this version should be considered compatible with the version" +
						" provided by this flag.",
				},
			},
			Action: func(c *cli.Context) error {
				return UpdateWorkerBuildIDVersionGraph(c)
			},
		},
		{
			Name:    "get-versions",
			Aliases: []string{"gv"},
			Usage:   "Fetches the Task Queue's worker build id version graph",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     FlagTaskQueue,
					Aliases:  FlagTaskQueueAlias,
					Usage:    "Task Queue name",
					Required: true,
				},
				&cli.UintFlag{
					Name: FlagGetBuildIDGraphMaxDepth,
					Usage: "If set nonzero, limit the depth of the fetched graph. The limit applies to each branch in " +
						"the DAG, so a value of 1 will return the current default, and one node of each compatible " +
						"branch, if any.",
				},
			},
			Action: func(c *cli.Context) error {
				return GetWorkerBuildIDVersionGraph(c)
			},
		},
	}
}
