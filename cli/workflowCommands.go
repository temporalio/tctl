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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/pborman/uuid"
	"github.com/urfave/cli/v2"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	filterpb "go.temporal.io/api/filter/v1"
	historypb "go.temporal.io/api/history/v1"
	querypb "go.temporal.io/api/query/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	sdkclient "go.temporal.io/sdk/client"

	"github.com/temporalio/tctl-kit/pkg/color"
	"github.com/temporalio/tctl-kit/pkg/output"
	"github.com/temporalio/tctl/cli/stringify"
	clispb "go.temporal.io/server/api/cli/v1"
	"go.temporal.io/server/common"
	"go.temporal.io/server/common/backoff"
	"go.temporal.io/server/common/clock"
	"go.temporal.io/server/common/collection"
	"go.temporal.io/server/common/convert"
	"go.temporal.io/server/common/primitives/timestamp"
	"go.temporal.io/server/common/searchattribute"
	"go.temporal.io/server/service/history/workflow"
)

// RunWorkflow starts a new workflow execution and print workflow progress and result
func RunWorkflow(c *cli.Context) error {
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	taskQueue := c.String(FlagTaskQueue)
	workflowType := c.String(FlagWorkflowType)
	et := c.Int(FlagWorkflowExecutionTimeout)
	rt := c.Int(FlagWorkflowRunTimeout)
	dt := c.Int(FlagWorkflowTaskTimeout)
	wid := c.String(FlagWorkflowID)
	if len(wid) == 0 {
		wid = uuid.New()
	}
	reusePolicy := defaultWorkflowIDReusePolicy
	if c.IsSet(FlagWorkflowIDReusePolicy) {
		reusePolicyInt, err := stringToEnum(c.String(FlagWorkflowIDReusePolicy), enumspb.WorkflowIdReusePolicy_value)
		if err != nil {
			return fmt.Errorf("unable to parse workflow ID reuse policy: %s", err)
		}
		reusePolicy = enumspb.WorkflowIdReusePolicy(reusePolicyInt)
	}

	inputs, err := unmarshalInputsFromCLI(c)
	if err != nil {
		return err
	}

	wo := sdkclient.StartWorkflowOptions{
		ID:                       wid,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: time.Duration(et) * time.Second,
		WorkflowTaskTimeout:      time.Duration(dt) * time.Second,
		WorkflowRunTimeout:       time.Duration(rt) * time.Second,
		WorkflowIDReusePolicy:    reusePolicy,
	}
	if c.IsSet(FlagCronSchedule) {
		wo.CronSchedule = c.String(FlagCronSchedule)
	}

	wo.Memo, err = unmarshalMemoFromCLI(c)
	if err != nil {
		return err
	}
	wo.SearchAttributes, err = unmarshalSearchAttrFromCLI(c)
	if err != nil {
		return err
	}

	tcCtx, cancel := newContextForLongPoll(c)
	defer cancel()
	resp, err := sdkClient.ExecuteWorkflow(tcCtx, wo, workflowType, inputs...)

	if err != nil {
		return fmt.Errorf("failed to run workflow: %s", err)
	}

	executionDetails := struct {
		WorkflowId string
		RunId      string
		Type       string
		Namespace  string
		TaskQueue  string
		Args       string
	}{

		WorkflowId: wid,
		RunId:      resp.GetRunID(),
		Type:       workflowType,
		Namespace:  namespace,
		TaskQueue:  taskQueue,
		Args:       truncate(formatInputsForDisplay(inputs)),
	}
	data := []interface{}{
		executionDetails,
	}
	fmt.Println(color.Magenta(c, "Running execution:"))
	opts := &output.PrintOptions{
		Fields:      []string{"WorkflowId", "RunId", "Type", "Namespace", "TaskQueue", "Args"},
		IgnoreFlags: true,
		Output:      output.Card,
		Separator:   "",
	}
	output.PrintItems(c, data, opts)

	return printWorkflowProgress(c, wid, resp.GetRunID(), true)
}

func unmarshalInputsFromCLI(c *cli.Context) ([]interface{}, error) {
	jsonsRaw, err := readJSONInputs(c)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	for _, jsonRaw := range jsonsRaw {
		if jsonRaw == nil {
			result = append(result, nil)
		} else {
			var j interface{}
			if err := json.Unmarshal(jsonRaw, &j); err != nil {
				return nil, fmt.Errorf("input is not valid JSON: %s", err)
			}
			result = append(result, j)
		}

	}

	return result, nil
}

func formatInputsForDisplay(inputs []interface{}) string {
	var result []string
	for _, input := range inputs {
		s, _ := json.Marshal(input)
		result = append(result, string(s))
	}
	return fmt.Sprintf("[%s]", strings.Join(result, ","))
}

func unmarshalSearchAttrFromCLI(c *cli.Context) (map[string]interface{}, error) {
	sanitize := func(val string) []string {
		trimmedVal := strings.TrimSpace(val)
		if len(trimmedVal) == 0 {
			return nil
		}
		splitVal := strings.Split(trimmedVal, searchAttrInputSeparator)
		result := make([]string, len(splitVal))
		for i, v := range splitVal {
			result[i] = strings.TrimSpace(v)
		}
		return result
	}

	searchAttrKeys := sanitize(c.String(FlagSearchAttributeKey))
	if len(searchAttrKeys) == 0 {
		return nil, nil
	}
	rawSearchAttrVals := sanitize(c.String(FlagSearchAttributeValue))
	if len(rawSearchAttrVals) == 0 {
		return nil, nil
	}

	if len(searchAttrKeys) != len(rawSearchAttrVals) {
		return nil, fmt.Errorf("uneven number of search attributes keys (%d): %v and values(%d): %v", len(searchAttrKeys), searchAttrKeys, len(rawSearchAttrVals), rawSearchAttrVals)
	}

	fields := make(map[string]interface{}, len(searchAttrKeys))

	for i, v := range rawSearchAttrVals {
		var j interface{}
		if err := json.Unmarshal([]byte(v), &j); err != nil {
			return nil, fmt.Errorf("unable to parse search attribute JSON: %s", err)
		}
		fields[searchAttrKeys[i]] = j
	}

	return fields, nil
}

func unmarshalMemoFromCLI(c *cli.Context) (map[string]interface{}, error) {
	// Memo flags were not passed => Memo is not provided.
	if !c.IsSet(FlagMemoKey) && !c.IsSet(FlagMemo) && !c.IsSet(FlagMemoFile) {
		return nil, nil
	}

	if !c.IsSet(FlagMemoKey) {
		return nil, fmt.Errorf("memo keys must be provided using %s", FlagMemoKey)
	}

	if c.IsSet(FlagMemo) && c.IsSet(FlagMemoFile) {
		return nil, fmt.Errorf("provide only one of %s or %s", FlagMemo, FlagMemoFile)
	}

	if !c.IsSet(FlagMemo) && !c.IsSet(FlagMemoFile) {
		return nil, fmt.Errorf("memo values must be provided using %s or %s", FlagMemo, FlagMemoFile)
	}

	memoKeys := c.StringSlice(FlagMemoKey)

	var memoValues []string
	if c.IsSet(FlagMemoFile) {
		inputFile := c.String(FlagMemoFile)
		// This method is purely used to parse input from the CLI. The input comes from a trusted user
		// #nosec
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read memo file %s", inputFile)
		}
		memoValues = strings.Split(string(data), "\n")
	} else if c.IsSet(FlagMemo) {
		memoValues = c.StringSlice(FlagMemo)
	}

	if len(memoKeys) != len(memoValues) {
		return nil, fmt.Errorf("number of memo keys %d and values %d are not equal", len(memoKeys), len(memoValues))
	}

	fields := make(map[string]interface{}, len(memoKeys))
	for i, key := range memoKeys {
		fields[key] = memoValues[i]
	}
	return fields, nil
}

type historyIterator struct {
	iter interface {
		HasNext() bool
		Next() (*historypb.HistoryEvent, error)
	}
	maxFieldLength int
	lastEvent      *historypb.HistoryEvent
}

func (h *historyIterator) HasNext() bool {
	return h.iter.HasNext()
}

func (h *historyIterator) Next() (interface{}, error) {
	event, err := h.iter.Next()
	if err != nil {
		return nil, err
	}

	reflect.ValueOf(h.lastEvent).Elem().Set(reflect.ValueOf(event).Elem())

	return eventRow{
		ID:      convert.Int64ToString(event.GetEventId()),
		Time:    formatTime(timestamp.TimeValue(event.GetEventTime()), false),
		Type:    ColorEvent(event),
		Details: HistoryEventToString(event, false, h.maxFieldLength),
	}, nil
}

// helper function to print workflow progress with time refresh every second
func printWorkflowProgress(c *cli.Context, wid, rid string, watch bool) error {
	var maxFieldLength int
	if c.IsSet(FlagMaxFieldLength) {
		maxFieldLength = c.Int(FlagMaxFieldLength)
	}
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	tcCtx, cancel := newIndefiniteContext(c)
	defer cancel()

	doneChan := make(chan bool)
	timeElapse := 1
	isTimeElapseExist := false
	ticker := time.NewTicker(time.Second).C
	opts := &output.PrintOptions{
		Fields:     []string{"ID", "Time", "Type"},
		FieldsLong: []string{"Details"},
	}
	fmt.Println(color.Magenta(c, "Progress:"))
	var lastEvent historypb.HistoryEvent // used for print result of this run

	errChan := make(chan error)
	go func() {
		hIter := sdkClient.GetWorkflowHistory(tcCtx, wid, rid, watch, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		iter := &historyIterator{iter: hIter, maxFieldLength: maxFieldLength, lastEvent: &lastEvent}
		err = output.Pager(c, iter, opts)
		if err != nil {
			errChan <- err
			return
		}

		doneChan <- true
	}()

	for {
		select {
		case <-ticker:
			if !watch {
				continue
			}

			if isTimeElapseExist {
				removePrevious2LinesFromTerminal()
			}
			fmt.Printf("\nTime elapse: %ds\n", timeElapse)
			isTimeElapseExist = true
			timeElapse++
		case <-doneChan: // print result of this run
			fmt.Println(color.Magenta(c, "\nResult:"))
			if watch {
				fmt.Printf("  Run Time: %d seconds\n", timeElapse)
			}
			printRunStatus(c, &lastEvent)
			return nil
		case err = <-errChan:
			return err
		}
	}
}

// TerminateWorkflow terminates a workflow execution
func TerminateWorkflow(c *cli.Context) error {
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)
	reason := c.String(FlagReason)

	ctx, cancel := newContext(c)
	defer cancel()
	err = sdkClient.TerminateWorkflow(ctx, wid, rid, reason, nil)
	if err != nil {
		return fmt.Errorf("unable to terminate workflow: %s", err)
	}

	fmt.Println("Terminate workflow succeeded.")

	return nil
}

// CancelWorkflow cancels a workflow execution
func CancelWorkflow(c *cli.Context) error {
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)

	ctx, cancel := newContext(c)
	defer cancel()
	err = sdkClient.CancelWorkflow(ctx, wid, rid)
	if err != nil {
		return fmt.Errorf("unable to cancel workflow: %s", err)
	}
	fmt.Println(color.Green(c, "canceled workflow, workflow id: %s, run id: %s", wid, rid))

	return nil
}

// SignalWorkflow signals a workflow execution
func SignalWorkflow(c *cli.Context) error {
	serviceClient := cFactory.FrontendClient(c)

	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}

	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)
	name := c.String(FlagName)
	input, err := processJSONInput(c)
	if err != nil {
		return err
	}

	tcCtx, cancel := newContext(c)
	defer cancel()
	_, err = serviceClient.SignalWorkflowExecution(tcCtx, &workflowservice.SignalWorkflowExecutionRequest{
		Namespace: namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
		SignalName: name,
		Input:      input,
		Identity:   getCliIdentity(),
	})

	if err != nil {
		return fmt.Errorf("signal workflow failed: %s", err)
	}

	fmt.Println("Signal workflow succeeded.")

	return nil
}

// QueryWorkflow query workflow execution
func QueryWorkflow(c *cli.Context) error {
	queryType := c.String(FlagQueryType)

	if err := queryWorkflowHelper(c, queryType); err != nil {
		return err
	}

	return nil
}

// QueryWorkflowUsingStackTrace query workflow execution using __stack_trace as query type
func QueryWorkflowUsingStackTrace(c *cli.Context) error {
	return queryWorkflowHelper(c, "__stack_trace")
}

func queryWorkflowHelper(c *cli.Context, queryType string) error {
	serviceClient := cFactory.FrontendClient(c)

	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)
	input, err := processJSONInput(c)
	if err != nil {
		return err
	}

	tcCtx, cancel := newContext(c)
	defer cancel()
	queryRequest := &workflowservice.QueryWorkflowRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
		Query: &querypb.WorkflowQuery{
			QueryType: queryType,
		},
	}
	if input != nil {
		queryRequest.Query.QueryArgs = input
	}
	if c.IsSet(FlagQueryRejectCondition) {
		var rejectCondition enumspb.QueryRejectCondition
		switch c.String(FlagQueryRejectCondition) {
		case "not_open":
			rejectCondition = enumspb.QUERY_REJECT_CONDITION_NOT_OPEN
		case "not_completed_cleanly":
			rejectCondition = enumspb.QUERY_REJECT_CONDITION_NOT_COMPLETED_CLEANLY
		default:
			return fmt.Errorf("invalid reject condition %v, valid values are \"not_open\" and \"not_completed_cleanly\"", c.String(FlagQueryRejectCondition))
		}
		queryRequest.QueryRejectCondition = rejectCondition
	}
	queryResponse, err := serviceClient.QueryWorkflow(tcCtx, queryRequest)
	if err != nil {
		return fmt.Errorf("query workflow failed: %s", err)
	}

	if queryResponse.QueryRejected != nil {
		fmt.Printf("Query was rejected, workflow has status: %v\n", queryResponse.QueryRejected.GetStatus())
	} else {
		queryResult := stringify.AnyToString(queryResponse.QueryResult, true, 0, customDataConverter())
		fmt.Printf("Query result:\n%v\n", queryResult)
	}

	return nil
}

// ListWorkflow list workflow executions based on filters
func ListWorkflow(c *cli.Context) error {
	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	queryOpen := c.Bool(FlagOpen)
	workflowID := c.String(FlagWorkflowID)
	workflowType := c.String(FlagWorkflowType)
	earliestTime, err := parseTime(c.String(FlagFrom), time.Time{}, time.Now().UTC())
	if err != nil {
		return err
	}
	latestTime, err := parseTime(c.String(FlagTo), time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return err
	}
	wfStatusInt, err := stringToEnum(c.String(FlagWorkflowStatus), enumspb.WorkflowExecutionStatus_value)
	if err != nil {
		return fmt.Errorf("unable to parse workflow status: %s", err)
	}
	wfStatus := enumspb.WorkflowExecutionStatus(wfStatusInt)
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	paginationFunc := func(npt []byte) ([]interface{}, []byte, error) {
		var items []interface{}
		var err error
		if c.IsSet(FlagListQuery) {
			query := c.String(FlagListQuery)
			items, npt, err = listWorkflows(c, sdkClient, npt, namespace, query)
		} else if queryOpen {
			items, npt, err = listOpenWorkflows(c, sdkClient, npt, namespace, earliestTime, latestTime, workflowID, workflowType)
		} else {
			items, npt, err = listClosedWorkflows(c, sdkClient, npt, namespace, earliestTime, latestTime, workflowID, workflowType, wfStatus)
		}
		if err != nil {
			return nil, nil, err
		}

		return items, npt, nil
	}

	iter := collection.NewPagingIterator(paginationFunc)
	opts := &output.PrintOptions{
		Fields:     []string{"Execution.WorkflowId", "Execution.RunId", "StartTime"},
		FieldsLong: []string{"Type.Name", "TaskQueue", "ExecutionTime", "CloseTime"},
	}
	return output.Pager(c, iter, opts)
}

// ScanAllWorkflow list all workflow executions using Scan API.
func ScanAllWorkflow(c *cli.Context) error {
	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	listQuery := c.String(FlagListQuery)
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	paginationFunc := func(npt []byte) ([]interface{}, []byte, error) {
		req := &workflowservice.ScanWorkflowExecutionsRequest{
			Namespace:     namespace,
			NextPageToken: npt,
			Query:         listQuery,
		}

		var workflows *workflowservice.ScanWorkflowExecutionsResponse
		op := func() error {
			ctx, cancel := newContext(c)
			defer cancel()
			response, err := sdkClient.ScanWorkflow(ctx, req)
			if err != nil {
				return err
			}
			workflows = response
			return nil
		}
		err := backoff.Retry(op, common.CreateFrontendServiceRetryPolicy(), common.IsContextDeadlineExceededErr)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to list workflow executions: %s", err)
		}

		var items []interface{}
		for _, e := range workflows.Executions {
			items = append(items, e)
		}
		if err != nil {
			return nil, nil, err
		}

		return items, workflows.NextPageToken, nil
	}

	iter := collection.NewPagingIterator(paginationFunc)
	opts := &output.PrintOptions{
		Fields:     []string{"Execution.WorkflowId", "Execution.RunId", "StartTime"},
		FieldsLong: []string{"Type.Name", "TaskQueue", "ExecutionTime", "CloseTime"},
	}

	return output.Pager(c, iter, opts)
}

// CountWorkflow count number of workflows
func CountWorkflow(c *cli.Context) error {
	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	query := c.String(FlagListQuery)
	request := &workflowservice.CountWorkflowExecutionsRequest{
		Query: query,
	}

	var count int64
	op := func() error {
		ctx, cancel := newContext(c)
		defer cancel()
		response, err := sdkClient.CountWorkflow(ctx, request)
		if err != nil {
			return err
		}
		count = response.GetCount()
		return nil
	}
	err = backoff.Retry(op, common.CreateFrontendServiceRetryPolicy(), common.IsContextDeadlineExceededErr)
	if err != nil {
		return fmt.Errorf("unable to count workflows: %s", err)
	}
	fmt.Println(count)
	return nil
}

// ListArchivedWorkflow lists archived workflow executions based on filters
func ListArchivedWorkflow(c *cli.Context) error {
	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	query := c.String(FlagListQuery)
	contextTimeout := defaultContextTimeoutForListArchivedWorkflow
	if c.IsSet(FlagContextTimeout) {
		contextTimeout = time.Duration(c.Int(FlagContextTimeout)) * time.Second
	}

	sdkClient, err := getSDKClient(c)
	if err != nil {
		return err
	}

	req := &workflowservice.ListArchivedWorkflowExecutionsRequest{
		Namespace: namespace,
		Query:     query,
	}
	var resp *workflowservice.ListArchivedWorkflowExecutionsResponse

	paginationFunc := func(npt []byte) ([]interface{}, []byte, error) {
		// the executions will be empty if the query is still running before timeout
		// so keep calling the API until some results are returned (query completed)
		req.NextPageToken = npt
		for resp == nil || (len(resp.Executions) == 0 && resp.NextPageToken != nil) {
			ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
			resp, err = sdkClient.ListArchivedWorkflow(ctx, req)
			if err != nil {
				cancel()
				return nil, nil, fmt.Errorf("unable to list archived workflows: %s", err)
			}
			cancel()
		}

		var items []interface{}
		for _, e := range resp.Executions {
			items = append(items, e)
		}

		return items, resp.NextPageToken, nil
	}

	iter := collection.NewPagingIterator(paginationFunc)
	opts := &output.PrintOptions{
		Fields:     []string{"Execution.WorkflowId", "Execution.RunId", "StartTime"},
		FieldsLong: []string{"Type.Name", "TaskQueue", "ExecutionTime", "CloseTime"},
	}
	return output.Pager(c, iter, opts)
}

// DescribeWorkflow show information about the specified workflow execution
func DescribeWorkflow(c *cli.Context) error {
	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)

	frontendClient := cFactory.FrontendClient(c)
	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	printRaw := c.Bool(FlagPrintRaw) // printRaw is false by default,
	// and will show datetime and decoded search attributes instead of raw timestamp and byte arrays
	printResetPointsOnly := c.Bool(FlagResetPointsOnly)

	ctx, cancel := newContext(c)
	defer cancel()

	resp, err := frontendClient.DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
	})
	if err != nil {
		return fmt.Errorf("workflow describe failed: %s", err)
	}

	if printResetPointsOnly {
		printAutoResetPoints(resp)
		return nil
	}

	if printRaw {
		prettyPrintJSONObject(resp)
	} else {
		prettyPrintJSONObject(convertDescribeWorkflowExecutionResponse(c, resp))
	}

	return nil
}

func printAutoResetPoints(resp *workflowservice.DescribeWorkflowExecutionResponse) {
	fmt.Println("Auto Reset Points:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(true)
	table.SetColumnSeparator("|")
	header := []string{"Binary Checksum", "Create Time", "RunId", "EventId"}
	headerColor := []tablewriter.Colors{tableHeaderBlue, tableHeaderBlue, tableHeaderBlue, tableHeaderBlue}
	table.SetHeader(header)
	table.SetHeaderColor(headerColor...)
	if resp.WorkflowExecutionInfo.AutoResetPoints != nil && len(resp.WorkflowExecutionInfo.AutoResetPoints.Points) > 0 {
		for _, pt := range resp.WorkflowExecutionInfo.AutoResetPoints.Points {
			var row []string
			row = append(row, pt.GetBinaryChecksum())
			row = append(row, timestamp.TimeValue(pt.GetCreateTime()).String())
			row = append(row, pt.GetRunId())
			row = append(row, convert.Int64ToString(pt.GetFirstWorkflowTaskCompletedId()))
			table.Append(row)
		}
	}
	table.Render()
}

func convertDescribeWorkflowExecutionResponse(c *cli.Context, resp *workflowservice.DescribeWorkflowExecutionResponse) *clispb.DescribeWorkflowExecutionResponse {

	info := resp.GetWorkflowExecutionInfo()
	executionInfo := &clispb.WorkflowExecutionInfo{
		Execution:            info.GetExecution(),
		Type:                 info.GetType(),
		CloseTime:            info.GetCloseTime(),
		StartTime:            info.GetStartTime(),
		Status:               info.GetStatus(),
		HistoryLength:        info.GetHistoryLength(),
		ParentNamespaceId:    info.GetParentNamespaceId(),
		ParentExecution:      info.GetParentExecution(),
		Memo:                 info.GetMemo(),
		SearchAttributes:     convertSearchAttributes(c, info.GetSearchAttributes()),
		AutoResetPoints:      info.GetAutoResetPoints(),
		StateTransitionCount: info.GetStateTransitionCount(),
	}

	var pendingActivitiesStr []*clispb.PendingActivityInfo
	for _, pendingActivity := range resp.GetPendingActivities() {
		pendingActivityStr := &clispb.PendingActivityInfo{
			ActivityId:         pendingActivity.GetActivityId(),
			ActivityType:       pendingActivity.GetActivityType(),
			State:              pendingActivity.GetState(),
			ScheduledTime:      pendingActivity.GetScheduledTime(),
			LastStartedTime:    pendingActivity.GetLastStartedTime(),
			LastHeartbeatTime:  pendingActivity.GetLastHeartbeatTime(),
			Attempt:            pendingActivity.GetAttempt(),
			MaximumAttempts:    pendingActivity.GetMaximumAttempts(),
			ExpirationTime:     pendingActivity.GetExpirationTime(),
			LastFailure:        convertFailure(pendingActivity.GetLastFailure()),
			LastWorkerIdentity: pendingActivity.GetLastWorkerIdentity(),
		}

		if pendingActivity.GetHeartbeatDetails() != nil {
			pendingActivityStr.HeartbeatDetails = stringify.AnyToString(pendingActivity.GetHeartbeatDetails(), true, 0, customDataConverter())
		}
		pendingActivitiesStr = append(pendingActivitiesStr, pendingActivityStr)
	}

	return &clispb.DescribeWorkflowExecutionResponse{
		ExecutionConfig:       resp.ExecutionConfig,
		WorkflowExecutionInfo: executionInfo,
		PendingActivities:     pendingActivitiesStr,
		PendingChildren:       resp.PendingChildren,
	}
}

func convertSearchAttributes(c *cli.Context, searchAttributes *commonpb.SearchAttributes) *clispb.SearchAttributes {
	if len(searchAttributes.GetIndexedFields()) == 0 {
		return nil
	}

	fields, err := searchattribute.Stringify(searchAttributes, nil)
	if err != nil {
		fmt.Printf("%s: unable to stringify search attribute: %v\n",
			color.Magenta(c, "Warning"),
			err)
	}

	return &clispb.SearchAttributes{IndexedFields: fields}
}

func convertFailure(failure *failurepb.Failure) *clispb.Failure {
	if failure == nil {
		return nil
	}

	fType := reflect.TypeOf(failure.GetFailureInfo()).Elem().Name()
	if failure.GetTimeoutFailureInfo() != nil {
		fType = fmt.Sprintf("%s: %s", fType, failure.GetTimeoutFailureInfo().GetTimeoutType().String())
	}

	f := &clispb.Failure{
		Message:     failure.GetMessage(),
		Source:      failure.GetSource(),
		StackTrace:  failure.GetStackTrace(),
		Cause:       convertFailure(failure.GetCause()),
		FailureType: fType,
	}

	return f
}

func printRunStatus(c *cli.Context, event *historypb.HistoryEvent) {
	switch event.GetEventType() {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		fmt.Printf("  Status: %s\n", color.Green(c, "COMPLETED"))
		result := stringify.AnyToString(event.GetWorkflowExecutionCompletedEventAttributes().GetResult(), true, 0, customDataConverter())
		fmt.Printf("  Output: %s\n", result)
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		fmt.Printf("  Status: %s\n", color.Red(c, "FAILED"))
		fmt.Printf("  Failure: %s\n", convertFailure(event.GetWorkflowExecutionFailedEventAttributes().GetFailure()).String())
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		fmt.Printf("  Status: %s\n", color.Red(c, "TIMEOUT"))
		fmt.Printf("  Retry status: %s\n", event.GetWorkflowExecutionTimedOutEventAttributes().GetRetryState())
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
		fmt.Printf("  Status: %s\n", color.Red(c, "CANCELED"))
		details := stringify.AnyToString(event.GetWorkflowExecutionCanceledEventAttributes().GetDetails(), true, 0, customDataConverter())
		fmt.Printf("  Detail: %s\n", details)
	}
}

func scanWorkflowExecutions(sdkClient sdkclient.Client, pageSize int, nextPageToken []byte, query string, c *cli.Context) ([]*workflowpb.WorkflowExecutionInfo, []byte, error) {
	request := &workflowservice.ScanWorkflowExecutionsRequest{
		PageSize:      int32(pageSize),
		NextPageToken: nextPageToken,
		Query:         query,
	}

	var workflows *workflowservice.ScanWorkflowExecutionsResponse
	op := func() error {
		ctx, cancel := newContext(c)
		defer cancel()
		response, err := sdkClient.ScanWorkflow(ctx, request)
		if err != nil {
			return err
		}
		workflows = response
		return nil
	}
	err := backoff.Retry(op, common.CreateFrontendServiceRetryPolicy(), common.IsContextDeadlineExceededErr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflow: %s", err)
	}

	return workflows.Executions, workflows.NextPageToken, nil
}

// ShowHistory shows the history of given workflow execution based on workflowID and runID.
func ShowHistory(c *cli.Context) error {
	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)

	return printWorkflowProgress(c, wid, rid, false)
}

// ObserveHistory show the process of running workflow
func ObserveHistory(c *cli.Context) error {
	wid := c.String(FlagWorkflowID)
	rid := c.String(FlagRunID)

	return printWorkflowProgress(c, wid, rid, true)
}

// ResetWorkflow reset workflow
func ResetWorkflow(c *cli.Context) error {
	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	wid := c.String(FlagWorkflowID)
	reason := c.String(FlagReason)
	if len(reason) == 0 {
		return fmt.Errorf("reason flag cannot be empty")
	}
	rid := c.String(FlagRunID)
	eventID := c.Int64(FlagEventID)
	resetType := c.String(FlagResetType)
	extraForResetType, ok := resetTypesMap[resetType]
	if !ok && eventID <= 0 {
		return fmt.Errorf("specify either valid event id or reset type (one of %s)", strings.Join(mapKeysToArray(resetTypesMap), ", "))
	}
	if ok && len(extraForResetType.(string)) > 0 {
		value := c.String(extraForResetType.(string))
		if len(value) == 0 {
			return fmt.Errorf("option %s is required", extraForResetType.(string))
		}
	}
	resetReapplyType := c.String(FlagResetReapplyType)
	if _, ok := resetReapplyTypesMap[resetReapplyType]; !ok {
		return fmt.Errorf("must specify valid reset reapply type: %v", strings.Join(mapKeysToArray(resetReapplyTypesMap), ", "))
	}

	ctx, cancel := newContext(c)
	defer cancel()

	frontendClient := cFactory.FrontendClient(c)

	resetBaseRunID := rid
	workflowTaskFinishID := eventID
	if resetType != "" {
		resetBaseRunID, workflowTaskFinishID, err = getResetEventIDByType(ctx, c, resetType, namespace, wid, rid, frontendClient)
		if err != nil {
			return fmt.Errorf("getting reset event ID by type failed: %s", err)
		}
	}
	resp, err := frontendClient.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
		Namespace: namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      resetBaseRunID,
		},
		Reason:                    fmt.Sprintf("%v:%v", getCurrentUserFromEnv(), reason),
		WorkflowTaskFinishEventId: workflowTaskFinishID,
		RequestId:                 uuid.New(),
		ResetReapplyType:          resetReapplyTypesMap[resetReapplyType].(enumspb.ResetReapplyType),
	})
	if err != nil {
		return fmt.Errorf("reset failed: %s", err)
	}
	prettyPrintJSONObject(resp)
	return nil
}

func processResets(c *cli.Context, namespace string, wes chan commonpb.WorkflowExecution, done chan bool, wg *sync.WaitGroup, params batchResetParamsType) {
	for {
		select {
		case we := <-wes:
			fmt.Println("received: ", we.GetWorkflowId(), we.GetRunId())
			wid := we.GetWorkflowId()
			rid := we.GetRunId()
			var err error
			for i := 0; i < 3; i++ {
				err = doReset(c, namespace, wid, rid, params)
				if err == nil {
					break
				}
				if _, ok := err.(*serviceerror.InvalidArgument); ok {
					break
				}
				fmt.Println("failed and retry...: ", wid, rid, err)
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000)))
			}
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
			if err != nil {
				fmt.Println("[ERROR] failed processing: ", wid, rid, err.Error())
			}
		case <-done:
			wg.Done()
			return
		}
	}
}

type batchResetParamsType struct {
	reason               string
	skipOpen             bool
	nonDeterministicOnly bool
	skipBaseNotCurrent   bool
	dryRun               bool
	resetType            string
}

// ResetInBatch resets workflow in batch
func ResetInBatch(c *cli.Context) error {
	namespace, err := getRequiredGlobalOption(c, FlagNamespace)
	if err != nil {
		return err
	}
	resetType := c.String(FlagResetType)

	inFileName := c.String(FlagInputFile)
	query := c.String(FlagListQuery)
	excFileName := c.String(FlagExcludeFile)
	separator := c.String(FlagInputSeparator)
	parallel := c.Int(FlagParallelism)

	extraForResetType, ok := resetTypesMap[resetType]
	if !ok {
		return fmt.Errorf("reset type is not supported: %s", extraForResetType)
	} else if len(extraForResetType.(string)) > 0 {
		value := c.String(extraForResetType.(string))
		if len(value) == 0 {
			return fmt.Errorf("option %s is required", extraForResetType.(string))
		}
	}

	batchResetParams := batchResetParamsType{
		reason:               c.String(FlagReason),
		skipOpen:             c.Bool(FlagSkipCurrentOpen),
		nonDeterministicOnly: c.Bool(FlagNonDeterministic),
		skipBaseNotCurrent:   c.Bool(FlagSkipBaseIsNotCurrent),
		dryRun:               c.Bool(FlagDryRun),
		resetType:            resetType,
	}

	if inFileName == "" && query == "" {
		return fmt.Errorf("must provide input file or list query to get target workflows to reset")
	}

	wg := &sync.WaitGroup{}

	wes := make(chan commonpb.WorkflowExecution)
	done := make(chan bool)
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go processResets(c, namespace, wes, done, wg, batchResetParams)
	}

	// read exclude
	excludes := map[string]string{}
	if len(excFileName) > 0 {
		// This code is only used in the CLI. The input provided is from a trusted user.
		// #nosec
		excFile, err := os.Open(excFileName)
		if err != nil {
			return fmt.Errorf("unable to read exclude rules: %s", err)
		}
		defer excFile.Close()
		scanner := bufio.NewScanner(excFile)
		idx := 0
		for scanner.Scan() {
			idx++
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 {
				fmt.Printf("line %v is empty, skipped\n", idx)
				continue
			}
			cols := strings.Split(line, separator)
			if len(cols) < 1 {
				return fmt.Errorf("exclude file: unable to split, line %v has less than 1 cols separated by comma, only %v", idx, len(cols))
			}
			wid := strings.TrimSpace(cols[0])
			rid := "not-needed"
			excludes[wid] = rid
		}
	}
	fmt.Println("num of excludes:", len(excludes))

	if len(inFileName) > 0 {
		inFile, err := os.Open(inFileName)
		if err != nil {
			return fmt.Errorf("unable to open input file: %s", err)
		}
		defer inFile.Close()
		scanner := bufio.NewScanner(inFile)
		idx := 0
		for scanner.Scan() {
			idx++
			line := strings.TrimSpace(scanner.Text())
			if len(line) == 0 {
				fmt.Printf("line %v is empty, skipped\n", idx)
				continue
			}
			cols := strings.Split(line, separator)
			if len(cols) < 1 {
				return fmt.Errorf("include file: unable to split, line %v has less than 1 cols separated by comma, only %v", idx, len(cols))
			}
			fmt.Printf("Start processing line %v ...\n", idx)
			wid := strings.TrimSpace(cols[0])
			rid := ""
			if len(cols) > 1 {
				rid = strings.TrimSpace(cols[1])
			}

			_, ok := excludes[wid]
			if ok {
				fmt.Println("skip by exclude file: ", wid, rid)
				continue
			}

			wes <- commonpb.WorkflowExecution{
				WorkflowId: wid,
				RunId:      rid,
			}
		}
	} else {
		sdkClient, err := getSDKClient(c)
		if err != nil {
			return err
		}

		pageSize := 1000
		var nextPageToken []byte
		var result []*workflowpb.WorkflowExecutionInfo
		for {
			result, nextPageToken, err = scanWorkflowExecutions(sdkClient, pageSize, nextPageToken, query, c)
			if err != nil {
				return err
			}
			for _, we := range result {
				wid := we.Execution.GetWorkflowId()
				rid := we.Execution.GetRunId()
				_, ok := excludes[wid]
				if ok {
					fmt.Println("skip by exclude file: ", wid, rid)
					continue
				}

				wes <- commonpb.WorkflowExecution{
					WorkflowId: wid,
					RunId:      rid,
				}
			}

			if nextPageToken == nil {
				break
			}
		}
	}

	close(done)
	fmt.Println("wait for all goroutines...")
	wg.Wait()

	return nil
}

func printErrorAndReturn(msg string, err error) error {
	fmt.Println(msg)
	return err
}

func doReset(c *cli.Context, namespace, wid, rid string, params batchResetParamsType) error {
	ctx, cancel := newContext(c)
	defer cancel()

	frontendClient := cFactory.FrontendClient(c)
	resp, err := frontendClient.DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
		},
	})
	if err != nil {
		return printErrorAndReturn("DescribeWorkflowExecution failed", err)
	}

	currentRunID := resp.WorkflowExecutionInfo.Execution.GetRunId()
	if currentRunID != rid && params.skipBaseNotCurrent {
		fmt.Println("skip because base run is different from current run: ", wid, rid, currentRunID)
		return nil
	}
	if rid == "" {
		rid = currentRunID
	}

	if resp.WorkflowExecutionInfo.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING || resp.WorkflowExecutionInfo.CloseTime == nil {
		if params.skipOpen {
			fmt.Println("skip because current run is open: ", wid, rid, currentRunID)
			// skip and not terminate current if open
			return nil
		}
	}

	if params.nonDeterministicOnly {
		isLDN, err := isLastEventWorkflowTaskFailedWithNonDeterminism(ctx, namespace, wid, rid, frontendClient)
		if err != nil {
			return printErrorAndReturn("check isLastEventWorkflowTaskFailedWithNonDeterminism failed", err)
		}
		if !isLDN {
			fmt.Println("skip because last event is not WorkflowTaskFailedWithNonDeterminism")
			return nil
		}
	}

	resetBaseRunID, workflowTaskFinishID, err := getResetEventIDByType(ctx, c, params.resetType, namespace, wid, rid, frontendClient)
	if err != nil {
		return printErrorAndReturn("getResetEventIDByType failed", err)
	}
	fmt.Println("WorkflowTaskFinishEventId for reset:", wid, rid, resetBaseRunID, workflowTaskFinishID)

	if params.dryRun {
		fmt.Printf("dry run to reset wid: %v, rid:%v to baseRunId:%v, eventId:%v \n", wid, rid, resetBaseRunID, workflowTaskFinishID)
	} else {
		resp2, err := frontendClient.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
			Namespace: namespace,
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: wid,
				RunId:      resetBaseRunID,
			},
			WorkflowTaskFinishEventId: workflowTaskFinishID,
			RequestId:                 uuid.New(),
			Reason:                    fmt.Sprintf("%v:%v", getCurrentUserFromEnv(), params.reason),
		})

		if err != nil {
			return printErrorAndReturn("ResetWorkflowExecution failed", err)
		}
		fmt.Println("new runId for wid/rid is ,", wid, rid, resp2.GetRunId())
	}

	return nil
}

func isLastEventWorkflowTaskFailedWithNonDeterminism(ctx context.Context, namespace, wid, rid string, frontendClient workflowservice.WorkflowServiceClient) (bool, error) {
	req := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
		MaximumPageSize: 1000,
		NextPageToken:   nil,
	}

	var firstEvent, workflowTaskFailedEvent *historypb.HistoryEvent
	for {
		resp, err := frontendClient.GetWorkflowExecutionHistory(ctx, req)
		if err != nil {
			return false, printErrorAndReturn("GetWorkflowExecutionHistory failed", err)
		}
		for _, e := range resp.GetHistory().GetEvents() {
			if firstEvent == nil {
				firstEvent = e
			}
			if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED {
				workflowTaskFailedEvent = e
			} else if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
				workflowTaskFailedEvent = nil
			}
		}
		if len(resp.NextPageToken) != 0 {
			req.NextPageToken = resp.NextPageToken
		} else {
			break
		}
	}

	if workflowTaskFailedEvent != nil {
		attr := workflowTaskFailedEvent.GetWorkflowTaskFailedEventAttributes()

		if attr.GetCause() == enumspb.WORKFLOW_TASK_FAILED_CAUSE_WORKFLOW_WORKER_UNHANDLED_FAILURE ||
			strings.Contains(attr.GetFailure().GetMessage(), "nondeterministic") {
			fmt.Printf("found non determnistic workflow wid:%v, rid:%v, orignalStartTime:%v \n", wid, rid, timestamp.TimeValue(firstEvent.GetEventTime()))
			return true, nil
		}
	}

	return false, nil
}

func getResetEventIDByType(ctx context.Context, c *cli.Context, resetType, namespace, wid, rid string, frontendClient workflowservice.WorkflowServiceClient) (resetBaseRunID string, workflowTaskFinishID int64, err error) {
	fmt.Println("resetType:", resetType)
	switch resetType {
	case "LastWorkflowTask":
		resetBaseRunID, workflowTaskFinishID, err = getLastWorkflowTaskEventID(ctx, namespace, wid, rid, frontendClient)
		if err != nil {
			return
		}
	case "LastContinuedAsNew":
		resetBaseRunID, workflowTaskFinishID, err = getLastContinueAsNewID(ctx, namespace, wid, rid, frontendClient)
		if err != nil {
			return
		}
	case "FirstWorkflowTask":
		resetBaseRunID, workflowTaskFinishID, err = getFirstWorkflowTaskEventID(ctx, namespace, wid, rid, frontendClient)
		if err != nil {
			return
		}
	case "BadBinary":
		binCheckSum := c.String(FlagResetBadBinaryChecksum)
		resetBaseRunID, workflowTaskFinishID, err = getBadWorkflowTaskCompletedID(ctx, namespace, wid, rid, binCheckSum, frontendClient)
		if err != nil {
			return
		}
	default:
		panic("not supported resetType")
	}
	return
}

// Returns event id of the last completed task or id of the next event after scheduled task.
func getLastWorkflowTaskEventID(ctx context.Context, namespace, wid, rid string, frontendClient workflowservice.WorkflowServiceClient) (resetBaseRunID string, workflowTaskEventID int64, err error) {
	resetBaseRunID = rid
	req := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
		MaximumPageSize: 1000,
		NextPageToken:   nil,
	}

	for {
		resp, err := frontendClient.GetWorkflowExecutionHistory(ctx, req)
		if err != nil {
			return "", 0, printErrorAndReturn("GetWorkflowExecutionHistory failed", err)
		}
		for _, e := range resp.GetHistory().GetEvents() {
			if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
				workflowTaskEventID = e.GetEventId()
			} else if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED {
				workflowTaskEventID = e.GetEventId() + 1
			}
		}
		if len(resp.NextPageToken) != 0 {
			req.NextPageToken = resp.NextPageToken
		} else {
			break
		}
	}
	if workflowTaskEventID == 0 {
		return "", 0, printErrorAndReturn("Get LastWorkflowTaskID failed", fmt.Errorf("unable to find any scheduled or completed task"))
	}
	return
}

func badChecksum(bad string) func(string) error {
	return func(maybeBad string) error {
		if maybeBad == bad {
			return fmt.Errorf("Bad checksum %q", bad)
		}
		return nil
	}
}

func getBadWorkflowTaskCompletedID(ctx context.Context, namespace, wid, rid, binChecksum string, frontendClient workflowservice.WorkflowServiceClient) (resetBaseRunID string, workflowTaskCompletedID int64, err error) {
	resetBaseRunID = rid
	resp, err := frontendClient.DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
	})
	if err != nil {
		return "", 0, printErrorAndReturn("DescribeWorkflowExecution failed", err)
	}

	_, p := workflow.FindAutoResetPoint(clock.NewRealTimeSource(), badChecksum(binChecksum), resp.WorkflowExecutionInfo.AutoResetPoints)
	if p != nil {
		workflowTaskCompletedID = p.GetFirstWorkflowTaskCompletedId()
	}

	if workflowTaskCompletedID == 0 {
		return "", 0, printErrorAndReturn("Get BadWorkflowTaskCompletedID failed", serviceerror.NewInvalidArgument("no WorkflowTaskCompletedID"))
	}
	return
}

// Returns id of the first workflow task completed event or if it doesn't exist then id of the event after task scheduled event.
func getFirstWorkflowTaskEventID(ctx context.Context, namespace, wid, rid string, frontendClient workflowservice.WorkflowServiceClient) (resetBaseRunID string, workflowTaskEventID int64, err error) {
	resetBaseRunID = rid
	req := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
		MaximumPageSize: 1000,
		NextPageToken:   nil,
	}
	for {
		resp, err := frontendClient.GetWorkflowExecutionHistory(ctx, req)
		if err != nil {
			return "", 0, printErrorAndReturn("GetWorkflowExecutionHistory failed", err)
		}
		for _, e := range resp.GetHistory().GetEvents() {
			if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
				workflowTaskEventID = e.GetEventId()
				return resetBaseRunID, workflowTaskEventID, nil
			}
			if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED {
				if workflowTaskEventID == 0 {
					workflowTaskEventID = e.GetEventId() + 1
				}
			}
		}
		if len(resp.NextPageToken) != 0 {
			req.NextPageToken = resp.NextPageToken
		} else {
			break
		}
	}
	if workflowTaskEventID == 0 {
		return "", 0, printErrorAndReturn("Get FirstWorkflowTaskID failed", fmt.Errorf("unable to find any scheduled or completed task"))
	}
	return
}

func getLastContinueAsNewID(ctx context.Context, namespace, wid, rid string, frontendClient workflowservice.WorkflowServiceClient) (resetBaseRunID string, workflowTaskCompletedID int64, err error) {
	// get first event
	req := &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      rid,
		},
		MaximumPageSize: 1,
		NextPageToken:   nil,
	}
	resp, err := frontendClient.GetWorkflowExecutionHistory(ctx, req)
	if err != nil {
		return "", 0, printErrorAndReturn("GetWorkflowExecutionHistory failed", err)
	}
	firstEvent := resp.History.Events[0]
	resetBaseRunID = firstEvent.GetWorkflowExecutionStartedEventAttributes().GetContinuedExecutionRunId()
	if resetBaseRunID == "" {
		return "", 0, printErrorAndReturn("GetWorkflowExecutionHistory failed", fmt.Errorf("cannot get resetBaseRunId"))
	}

	req = &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: wid,
			RunId:      resetBaseRunID,
		},
		MaximumPageSize: 1000,
		NextPageToken:   nil,
	}
	for {
		resp, err := frontendClient.GetWorkflowExecutionHistory(ctx, req)
		if err != nil {
			return "", 0, printErrorAndReturn("GetWorkflowExecutionHistory failed", err)
		}
		for _, e := range resp.GetHistory().GetEvents() {
			if e.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
				workflowTaskCompletedID = e.GetEventId()
			}
		}
		if len(resp.NextPageToken) != 0 {
			req.NextPageToken = resp.NextPageToken
		} else {
			break
		}
	}
	if workflowTaskCompletedID == 0 {
		return "", 0, printErrorAndReturn("Get LastContinueAsNewID failed", fmt.Errorf("no WorkflowTaskCompletedID"))
	}
	return
}

func listWorkflows(c *cli.Context, sdkClient sdkclient.Client, npt []byte, namespace string, query string) ([]interface{}, []byte, error) {
	req := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace:     namespace,
		NextPageToken: npt,
		Query:         query,
	}

	var workflows *workflowservice.ListWorkflowExecutionsResponse
	op := func() error {
		ctx, cancel := newContext(c)
		defer cancel()
		resp, err := sdkClient.ListWorkflow(ctx, req)
		if err != nil {
			return err
		}
		workflows = resp
		return nil
	}
	err := backoff.Retry(op, common.CreateFrontendServiceRetryPolicy(), common.IsContextDeadlineExceededErr)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to list workflow executions: %s", err)
	}

	var items []interface{}
	for _, e := range workflows.Executions {
		items = append(items, e)
	}

	return items, workflows.NextPageToken, nil
}

func listOpenWorkflows(c *cli.Context, sdkClient sdkclient.Client, npt []byte, namespace string, earliestTime, latestTime time.Time, wfID, wfType string) ([]interface{}, []byte, error) {
	req := &workflowservice.ListOpenWorkflowExecutionsRequest{
		Namespace:     namespace,
		NextPageToken: npt,
		StartTimeFilter: &filterpb.StartTimeFilter{
			EarliestTime: &earliestTime,
			LatestTime:   &latestTime,
		},
	}
	if len(wfID) > 0 {
		req.Filters = &workflowservice.ListOpenWorkflowExecutionsRequest_ExecutionFilter{ExecutionFilter: &filterpb.WorkflowExecutionFilter{WorkflowId: wfID}}
	}
	if len(wfType) > 0 {
		req.Filters = &workflowservice.ListOpenWorkflowExecutionsRequest_TypeFilter{TypeFilter: &filterpb.WorkflowTypeFilter{Name: wfType}}
	}
	var workflows *workflowservice.ListOpenWorkflowExecutionsResponse
	op := func() error {
		ctx, cancel := newContext(c)
		defer cancel()
		resp, err := sdkClient.ListOpenWorkflow(ctx, req)
		if err != nil {
			return err
		}
		workflows = resp
		return nil
	}
	err := backoff.Retry(op, common.CreateFrontendServiceRetryPolicy(), common.IsContextDeadlineExceededErr)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to list open workflow executions: %s", err)
	}

	var items []interface{}
	for _, e := range workflows.Executions {
		items = append(items, e)
	}

	return items, workflows.NextPageToken, nil
}

func listClosedWorkflows(c *cli.Context, sdkClient sdkclient.Client, npt []byte, namespace string, earliestTime, latestTime time.Time, wfID, wfType string,
	wfStatus enumspb.WorkflowExecutionStatus) ([]interface{}, []byte, error) {
	req := &workflowservice.ListClosedWorkflowExecutionsRequest{
		Namespace:     namespace,
		NextPageToken: npt,
		StartTimeFilter: &filterpb.StartTimeFilter{
			EarliestTime: &earliestTime,
			LatestTime:   &latestTime,
		},
	}
	if len(wfID) > 0 {
		req.Filters = &workflowservice.ListClosedWorkflowExecutionsRequest_ExecutionFilter{ExecutionFilter: &filterpb.WorkflowExecutionFilter{WorkflowId: wfID}}
	}
	if len(wfType) > 0 {
		req.Filters = &workflowservice.ListClosedWorkflowExecutionsRequest_TypeFilter{TypeFilter: &filterpb.WorkflowTypeFilter{Name: wfType}}
	}
	if wfStatus != enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED {
		req.Filters = &workflowservice.ListClosedWorkflowExecutionsRequest_StatusFilter{StatusFilter: &filterpb.StatusFilter{Status: wfStatus}}
	}
	var workflows *workflowservice.ListClosedWorkflowExecutionsResponse
	op := func() error {
		ctx, cancel := newContext(c)
		defer cancel()
		resp, err := sdkClient.ListClosedWorkflow(ctx, req)
		if err != nil {
			return err
		}
		workflows = resp
		return nil
	}
	err := backoff.Retry(op, common.CreateFrontendServiceRetryPolicy(), common.IsContextDeadlineExceededErr)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to list closed workflow executions: %s", err)
	}
	var items []interface{}
	for _, e := range workflows.Executions {
		items = append(items, e)
	}

	return items, workflows.NextPageToken, nil
}

type eventRow struct {
	ID      string
	Time    string
	Type    string
	Details string
}
