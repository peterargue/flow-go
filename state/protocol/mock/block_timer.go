// Code generated by mockery v2.13.0. DO NOT EDIT.

package mock

import (
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// BlockTimer is an autogenerated mock type for the BlockTimer type
type BlockTimer struct {
	mock.Mock
}

// Build provides a mock function with given fields: parentTimestamp
func (_m *BlockTimer) Build(parentTimestamp time.Time) time.Time {
	ret := _m.Called(parentTimestamp)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(time.Time) time.Time); ok {
		r0 = rf(parentTimestamp)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// Validate provides a mock function with given fields: parentTimestamp, currentTimestamp
func (_m *BlockTimer) Validate(parentTimestamp time.Time, currentTimestamp time.Time) error {
	ret := _m.Called(parentTimestamp, currentTimestamp)

	var r0 error
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) error); ok {
		r0 = rf(parentTimestamp, currentTimestamp)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type NewBlockTimerT interface {
	mock.TestingT
	Cleanup(func())
}

// NewBlockTimer creates a new instance of BlockTimer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBlockTimer(t NewBlockTimerT) *BlockTimer {
	mock := &BlockTimer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
