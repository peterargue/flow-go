// Code generated by mockery v2.21.4. DO NOT EDIT.

package mock

import (
	context "context"

	flow "github.com/onflow/flow-go/model/flow"
	execution_data "github.com/onflow/flow-go/module/executiondatasync/execution_data"

	mock "github.com/stretchr/testify/mock"

	state_stream "github.com/onflow/flow-go/engine/access/state_stream"
)

// API is an autogenerated mock type for the API type
type API struct {
	mock.Mock
}

// GetExecutionDataByBlockID provides a mock function with given fields: ctx, blockID
func (_m *API) GetExecutionDataByBlockID(ctx context.Context, blockID flow.Identifier) (*execution_data.BlockExecutionData, error) {
	ret := _m.Called(ctx, blockID)

	var r0 *execution_data.BlockExecutionData
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, flow.Identifier) (*execution_data.BlockExecutionData, error)); ok {
		return rf(ctx, blockID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, flow.Identifier) *execution_data.BlockExecutionData); ok {
		r0 = rf(ctx, blockID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*execution_data.BlockExecutionData)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, flow.Identifier) error); ok {
		r1 = rf(ctx, blockID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubscribeEvents provides a mock function with given fields: ctx, startBlockID, startHeight, filter
func (_m *API) SubscribeEvents(ctx context.Context, startBlockID flow.Identifier, startHeight uint64, filter state_stream.EventFilter) state_stream.Subscription {
	ret := _m.Called(ctx, startBlockID, startHeight, filter)

	var r0 state_stream.Subscription
	if rf, ok := ret.Get(0).(func(context.Context, flow.Identifier, uint64, state_stream.EventFilter) state_stream.Subscription); ok {
		r0 = rf(ctx, startBlockID, startHeight, filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(state_stream.Subscription)
		}
	}

	return r0
}

// SubscribeExecutionData provides a mock function with given fields: ctx, startBlockID, startBlockHeight
func (_m *API) SubscribeExecutionData(ctx context.Context, startBlockID flow.Identifier, startBlockHeight uint64) state_stream.Subscription {
	ret := _m.Called(ctx, startBlockID, startBlockHeight)

	var r0 state_stream.Subscription
	if rf, ok := ret.Get(0).(func(context.Context, flow.Identifier, uint64) state_stream.Subscription); ok {
		r0 = rf(ctx, startBlockID, startBlockHeight)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(state_stream.Subscription)
		}
	}

	return r0
}

type mockConstructorTestingTNewAPI interface {
	mock.TestingT
	Cleanup(func())
}

// NewAPI creates a new instance of API. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAPI(t mockConstructorTestingTNewAPI) *API {
	mock := &API{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
