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
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/temporalio/tctl-kit/pkg/color"
	"github.com/temporalio/tctl-kit/pkg/output"
	"github.com/urfave/cli/v2"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
)

const (
	addSearchAttributesTimeout = 30 * time.Second
)

// ListSearchAttributes lists search attributes
func ListSearchAttributes(c *cli.Context) error {
	namespace := c.String(FlagNamespace)
	client := cFactory.OperatorClient(c)
	ctx, cancel := newContext(c)
	defer cancel()

	resp, err := client.ListSearchAttributes(
		ctx,
		&operatorservice.ListSearchAttributesRequest{Namespace: namespace},
	)
	if err != nil {
		return fmt.Errorf("unable to list search attributes: %w", err)
	}

	var items []interface{}
	type sa struct {
		Name string
		Type string
	}
	for saName, saType := range resp.GetSystemAttributes() {
		items = append(items, sa{Name: saName, Type: saType.String()})
	}
	for saName, saType := range resp.GetCustomAttributes() {
		items = append(items, sa{Name: saName, Type: saType.String()})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].(sa).Name < items[j].(sa).Name
	})

	opts := &output.PrintOptions{
		Fields: []string{"Name", "Type"},
	}
	return output.PrintItems(c, items, opts)
}

// AddSearchAttributes to add search attributes
func AddSearchAttributes(c *cli.Context) error {
	namespace := c.String(FlagNamespace)
	names := c.StringSlice(FlagName)
	typeStrs := c.StringSlice(FlagType)

	if len(names) != len(typeStrs) {
		return fmt.Errorf("number of --%s and --%s options should be the same", FlagName, FlagType)
	}

	client := cFactory.OperatorClient(c)

	ctx, cancel := newContext(c)
	defer cancel()
	listReq := &operatorservice.ListSearchAttributesRequest{}
	existingSearchAttributes, err := client.ListSearchAttributes(ctx, listReq)
	if err != nil {
		return fmt.Errorf("unable to get existing search attributes: %w", err)
	}

	searchAttributes := make(map[string]enumspb.IndexedValueType, len(typeStrs))
	for i := 0; i < len(typeStrs); i++ {
		typeStr := typeStrs[i]

		typeInt, err := stringToEnum(typeStr, enumspb.IndexedValueType_value)
		if err != nil {
			return fmt.Errorf("unable to parse search attribute type %s: %w", typeStr, err)
		}
		existingSearchAttributeType, searchAttributeExists := existingSearchAttributes.CustomAttributes[names[i]]
		if !searchAttributeExists {
			searchAttributes[names[i]] = enumspb.IndexedValueType(typeInt)
			continue
		}
		if existingSearchAttributeType != enumspb.IndexedValueType(typeInt) {
			return fmt.Errorf("search attribute %s already exists and has different type %s: %w", names[i], existingSearchAttributeType, err)
		}
	}

	if len(searchAttributes) == 0 {
		fmt.Println(color.Yellow(c, "Search attributes already exist"))
		return nil
	}

	promptMsg := fmt.Sprintf(
		"You are about to add search attributes %s. Continue? Y/N",
		color.Yellow(c, strings.TrimLeft(fmt.Sprintf("%v", searchAttributes), "map")),
	)
	if !promptYes(promptMsg, c.Bool(FlagYes)) {
		return nil
	}

	request := &operatorservice.AddSearchAttributesRequest{
		SearchAttributes: searchAttributes,
		Namespace:        namespace,
	}

	ctx, cancel = newContextWithTimeout(c, addSearchAttributesTimeout)
	defer cancel()
	_, err = client.AddSearchAttributes(ctx, request)
	if err != nil {
		return fmt.Errorf("unable to add search attributes: %w", err)
	}
	fmt.Println(color.Green(c, "Search attributes have been added"))
	return nil
}

// RemoveSearchAttributes to add search attributes
func RemoveSearchAttributes(c *cli.Context) error {
	namespace := c.String(FlagNamespace)
	names := c.StringSlice(FlagName)

	promptMsg := fmt.Sprintf(
		"You are about to remove search attributes %s. Continue? Y/N",
		color.Yellow(c, "%v", names),
	)
	if !promptYes(promptMsg, c.Bool(FlagYes)) {
		return nil
	}

	client := cFactory.OperatorClient(c)
	ctx, cancel := newContext(c)
	defer cancel()
	request := &operatorservice.RemoveSearchAttributesRequest{
		SearchAttributes: names,
		Namespace:        namespace,
	}

	_, err := client.RemoveSearchAttributes(ctx, request)
	if err != nil {
		return fmt.Errorf("unable to remove search attributes: %w", err)
	}
	fmt.Println(color.Green(c, "Search attributes have been removed"))
	return nil
}
