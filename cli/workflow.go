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

	"github.com/temporalio/tctl-kit/pkg/flags"
	"github.com/urfave/cli/v2"
)

func newWorkflowCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "run",
			Usage: "Start a new workflow execution and show progress",
			Flags: append(flagsForRunWorkflow, flags.FlagsForPaginationAndRendering...),
			Action: func(c *cli.Context) error {
				return RunWorkflow(c)
			},
		},
		{
			Name:    "describe",
			Aliases: []string{"d"},
			Usage:   "Show information of workflow execution",
			Flags: append(flagsForExecution, []cli.Flag{
				&cli.BoolFlag{
					Name:  FlagResetPointsOnly,
					Usage: "Only show auto-reset points",
				},
			}...),
			Action: func(c *cli.Context) error {
				return DescribeWorkflow(c)
			},
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List workflow executions based on query",
			Flags:   append(flagsForWorkflowFiltering, flags.FlagsForPaginationAndRendering...),
			Action: func(c *cli.Context) error {
				return ListWorkflow(c)
			},
		},
		{
			Name:  "show",
			Usage: "Show workflow history",
			Flags: append(append(flagsForExecution, flagsForShowWorkflow...), flags.FlagsForPaginationAndRendering...),
			Action: func(c *cli.Context) error {
				return ShowHistory(c)
			},
		},
		{
			Name:  "query",
			Usage: "Query workflow execution",
			Flags: append(flagsForStackTraceQuery,
				&cli.StringFlag{
					Name:     FlagQueryType,
					Aliases:  FlagQueryTypeAlias,
					Usage:    "The query type you want to run",
					Required: true,
				}),
			Action: func(c *cli.Context) error {
				return QueryWorkflow(c)

			},
		},
		{
			Name:  "stack",
			Usage: "Query workflow execution with __stack_trace as query type",
			Flags: flagsForStackTraceQuery,
			Action: func(c *cli.Context) error {
				return QueryWorkflowUsingStackTrace(c)
			},
		},
		{
			Name:    "signal",
			Aliases: []string{"s"},
			Usage:   "Signal a workflow execution",
			Flags: append(flagsForExecution, []cli.Flag{
				&cli.StringFlag{
					Name:     FlagName,
					Aliases:  FlagNameAlias,
					Usage:    "Signal Name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    FlagInput,
					Aliases: FlagInputAlias,
					Usage:   "Input for the signal (JSON)",
				},
				&cli.StringFlag{
					Name:    FlagInputFile,
					Aliases: FlagInputFileAlias,
					Usage:   "Input for the signal from file (JSON)",
				},
			}...),
			Action: func(c *cli.Context) error {
				return SignalWorkflow(c)
			},
		},
		{
			Name:  "scan",
			Usage: "List workflow executions. Faster and unsorted (requires Elasticsearch to be enabled)",
			Flags: append(flagsForScan, flags.FlagsForPaginationAndRendering...),
			Action: func(c *cli.Context) error {
				return ScanAllWorkflow(c)
			},
		},
		{
			Name:  "count",
			Usage: "Count workflow executions (requires ElasticSearch to be enabled)",
			Flags: getFlagsForCount(),
			Action: func(c *cli.Context) error {
				return CountWorkflow(c)
			},
		},
		{
			Name:  "cancel",
			Usage: "Cancel a workflow execution",
			Flags: flagsForExecution,
			Action: func(c *cli.Context) error {
				return CancelWorkflow(c)
			},
		},
		{
			Name:  "terminate",
			Usage: "Terminate a workflow execution",
			Flags: append(flagsForExecution, []cli.Flag{
				&cli.StringFlag{
					Name:    FlagReason,
					Aliases: FlagReasonAlias,
					Usage:   "Reason for terminating the Workflow Execution",
				},
			}...),
			Action: func(c *cli.Context) error {
				return TerminateWorkflow(c)
			},
		},
		{
			Name:  "reset",
			Usage: "Reset the workflow, by either eventId or resetType",
			Flags: append(flagsForExecution, []cli.Flag{
				&cli.StringFlag{
					Name:  FlagEventID,
					Usage: "The eventId of any event after WorkflowTaskStarted you want to reset to (exclusive). It can be WorkflowTaskCompleted, WorkflowTaskFailed or others",
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Usage:    "Reason for resetting the workflow execution",
					Required: true,
				},
				&cli.StringFlag{
					Name: FlagResetType,
					Usage: "Event type to which you want to reset: " +
						strings.Join(mapKeysToArray(resetTypesMap), ","),
				},
				&cli.StringFlag{
					Name: FlagResetReapplyType,
					Usage: "Event types to reapply after the reset point: " +
						strings.Join(mapKeysToArray(resetReapplyTypesMap), ",") + ". (default: All)",
				},
				&cli.StringFlag{
					Name:  FlagResetBadBinaryChecksum,
					Usage: "Binary checksum for resetType of BadBinary",
				},
			}...),
			Action: func(c *cli.Context) error {
				return ResetWorkflow(c)
			},
		},
		{
			Name:  "reset-batch",
			Usage: " Resets a batch of Workflow Executions by reset type: " + strings.Join(mapKeysToArray(resetTypesMap), ","),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    FlagListQuery,
					Aliases: FlagListQueryAlias,
					Usage:   "Visibility query of Search Attributes describing the Workflow Executions to reset",
				}, &cli.StringFlag{
					Name:    FlagInputFile,
					Aliases: FlagInputFileAlias,
					Usage:   "Input file that specifies Workflow Executions to reset. Each line contains one Workflow Id as the base Run and, optionally, a Run Id",
				},
				&cli.StringFlag{
					Name:  FlagExcludeFile,
					Value: "",
					Usage: "Input file that specifies Workflow Executions to exclude from resetting",
				},
				&cli.StringFlag{
					Name:  FlagInputSeparator,
					Value: "\t",
					Usage: "Separator for the input file. The default is a tab (\t)",
				},
				&cli.StringFlag{
					Name:     FlagReason,
					Usage:    "Reason for resetting the Workflow Executions",
					Required: true,
				},
				&cli.IntFlag{
					Name:  FlagParallelism,
					Value: 1,
					Usage: "Number of goroutines to run in parallel. Each goroutine processes one line for every second",
				},
				&cli.BoolFlag{
					Name:  FlagSkipCurrentOpen,
					Usage: "Skip a Workflow Execution if the current Run is open for the same Workflow Id as the base Run",
				},
				&cli.BoolFlag{
					Name: FlagSkipBaseIsNotCurrent,
					// TODO https://github.com/uber/cadence/issues/2930
					// The right way to prevent needs server side implementation .
					// This client side is only best effort
					Usage: "Skip a Workflow Execution if the base Run is not the current Run",
				},
				&cli.BoolFlag{
					Name:  FlagNonDeterministic,
					Usage: "Reset workflow execution only if its last event is WorkflowTaskFailed with a nondeterministic error",
				},
				&cli.StringFlag{
					Name:     FlagResetType,
					Usage:    "Where to reset: " + strings.Join(mapKeysToArray(resetTypesMap), ","),
					Required: true,
				},
				&cli.StringFlag{
					Name:  FlagResetBadBinaryChecksum,
					Usage: "Binary checksum for resetType of BadBinary",
				},
				&cli.BoolFlag{
					Name:  FlagDryRun,
					Usage: "Simulate reset without resetting any Workflow Executions",
				},
			},
			Action: func(c *cli.Context) error {
				return ResetInBatch(c)
			},
		},
	}
}
