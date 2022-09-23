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
	"github.com/golang/mock/gomock"
	"go.temporal.io/api/workflowservice/v1"
)

func (s *cliAppSuite) TestListBatchJobs() {
	s.frontendClient.EXPECT().ListBatchOperations(gomock.Any(), gomock.Any()).Return(&workflowservice.ListBatchOperationsResponse{}, nil).Times(2)
	err := s.app.Run([]string{"", "batch", "list"})
	s.Nil(err)
	s.sdkClient.AssertExpectations(s.T())

	err = s.app.Run([]string{"", "batch", "list", "--fields", "long"})
	s.Nil(err)
	s.sdkClient.AssertExpectations(s.T())
}

func (s *cliAppSuite) TestStopBatchJob() {
	s.sdkClient.On("WorkflowService().StopWorkflowExecution", mock.Anything, mock.Anything).Return(&workflowservice.StopBatchOperationResponse{}, nil).Once()
	err := s.app.Run([]string{"", "batch", "stop", "--job-id", "test", "--reason", "test"})
	s.Nil(err)
	s.sdkClient.AssertExpectations(s.T())
}

func (s *cliAppSuite) TestDescribeBatchJob() {
	s.sdkClient.On("WorkflowService().DescribeBatchJob", mock.Anything, mock.Anything).Return(&workflowservice.DescribeBatchOperationResponse{}, nil).Once()
	err := s.app.Run([]string{"", "batch", "describe", "--job-id", "test"})
	s.Nil(err)
	s.sdkClient.AssertExpectations(s.T())
}
