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

package format

import (
	"fmt"
	"time"

	"github.com/temporalio/tctl/terminal/timeformat"
	"github.com/urfave/cli/v2"
)

const (
	FlagFormat = "format"
)

type FormatOption string

const (
	Table FormatOption = "table"
	JSON  FormatOption = "json"
	Card  FormatOption = "card"
)

type PrintOptions struct {
	Fields    []string
	Header    bool
	Separator string
}

func formatField(c *cli.Context, i interface{}) string {
	switch v := i.(type) {
	case time.Time:
		return timeformat.FormatTime(c, &v)
	case *time.Time:
		return timeformat.FormatTime(c, v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
