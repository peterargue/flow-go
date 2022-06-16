// Code generated by mockery v2.13.0. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"
)

// OnViewCallback is an autogenerated mock type for the OnViewCallback type
type OnViewCallback struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *OnViewCallback) Execute(_a0 *flow.Header) {
	_m.Called(_a0)
}

type NewOnViewCallbackT interface {
	mock.TestingT
	Cleanup(func())
}

// NewOnViewCallback creates a new instance of OnViewCallback. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewOnViewCallback(t NewOnViewCallbackT) *OnViewCallback {
	mock := &OnViewCallback{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}