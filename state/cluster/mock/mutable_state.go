// Code generated by mockery v2.13.0. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	cluster "github.com/onflow/flow-go/state/cluster"

	mock "github.com/stretchr/testify/mock"

	modelcluster "github.com/onflow/flow-go/model/cluster"
)

// MutableState is an autogenerated mock type for the MutableState type
type MutableState struct {
	mock.Mock
}

// AtBlockID provides a mock function with given fields: blockID
func (_m *MutableState) AtBlockID(blockID flow.Identifier) cluster.Snapshot {
	ret := _m.Called(blockID)

	var r0 cluster.Snapshot
	if rf, ok := ret.Get(0).(func(flow.Identifier) cluster.Snapshot); ok {
		r0 = rf(blockID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cluster.Snapshot)
		}
	}

	return r0
}

// Extend provides a mock function with given fields: candidate
func (_m *MutableState) Extend(candidate *modelcluster.Block) error {
	ret := _m.Called(candidate)

	var r0 error
	if rf, ok := ret.Get(0).(func(*modelcluster.Block) error); ok {
		r0 = rf(candidate)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Final provides a mock function with given fields:
func (_m *MutableState) Final() cluster.Snapshot {
	ret := _m.Called()

	var r0 cluster.Snapshot
	if rf, ok := ret.Get(0).(func() cluster.Snapshot); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cluster.Snapshot)
		}
	}

	return r0
}

// Params provides a mock function with given fields:
func (_m *MutableState) Params() cluster.Params {
	ret := _m.Called()

	var r0 cluster.Params
	if rf, ok := ret.Get(0).(func() cluster.Params); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cluster.Params)
		}
	}

	return r0
}

type NewMutableStateT interface {
	mock.TestingT
	Cleanup(func())
}

// NewMutableState creates a new instance of MutableState. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMutableState(t NewMutableStateT) *MutableState {
	mock := &MutableState{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
