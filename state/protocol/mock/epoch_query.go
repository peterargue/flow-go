// Code generated by mockery v2.43.2. DO NOT EDIT.

package mock

import (
	protocol "github.com/onflow/flow-go/state/protocol"
	mock "github.com/stretchr/testify/mock"
)

// EpochQuery is an autogenerated mock type for the EpochQuery type
type EpochQuery struct {
	mock.Mock
}

// Current provides a mock function with given fields:
func (_m *EpochQuery) Current() protocol.Epoch {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Current")
	}

	var r0 protocol.Epoch
	if rf, ok := ret.Get(0).(func() protocol.Epoch); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.Epoch)
		}
	}

	return r0
}

// NextCommitted provides a mock function with given fields:
func (_m *EpochQuery) NextCommitted() protocol.Epoch {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NextCommitted")
	}

	var r0 protocol.Epoch
	if rf, ok := ret.Get(0).(func() protocol.Epoch); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.Epoch)
		}
	}

	return r0
}

// NextUnsafe provides a mock function with given fields:
func (_m *EpochQuery) NextUnsafe() protocol.TentativeEpoch {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NextUnsafe")
	}

	var r0 protocol.TentativeEpoch
	if rf, ok := ret.Get(0).(func() protocol.TentativeEpoch); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.TentativeEpoch)
		}
	}

	return r0
}

// Previous provides a mock function with given fields:
func (_m *EpochQuery) Previous() protocol.Epoch {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Previous")
	}

	var r0 protocol.Epoch
	if rf, ok := ret.Get(0).(func() protocol.Epoch); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(protocol.Epoch)
		}
	}

	return r0
}

// NewEpochQuery creates a new instance of EpochQuery. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewEpochQuery(t interface {
	mock.TestingT
	Cleanup(func())
}) *EpochQuery {
	mock := &EpochQuery{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
