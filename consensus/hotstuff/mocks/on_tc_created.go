// Code generated by mockery v2.12.3. DO NOT EDIT.

package mocks

import (
	flow "github.com/onflow/flow-go/model/flow"

	mock "github.com/stretchr/testify/mock"
)

// OnTCCreated is an autogenerated mock type for the OnTCCreated type
type OnTCCreated struct {
	mock.Mock
}

// Execute provides a mock function with given fields: tc
func (_m *OnTCCreated) Execute(tc *flow.TimeoutCertificate) {
	_m.Called(tc)
}

type NewOnTCCreatedT interface {
	mock.TestingT
	Cleanup(func())
}

// NewOnTCCreated creates a new instance of OnTCCreated. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewOnTCCreated(t NewOnTCCreatedT) *OnTCCreated {
	mock := &OnTCCreated{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
