// Code generated by mockery v2.43.2. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"
)

// LinkGenerator is an autogenerated mock type for the LinkGenerator type
type LinkGenerator struct {
	mock.Mock
}

// AccountLink provides a mock function with given fields: address
func (_m *LinkGenerator) AccountLink(address string) (string, error) {
	ret := _m.Called(address)

	if len(ret) == 0 {
		panic("no return value specified for AccountLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(address)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(address)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BlockLink provides a mock function with given fields: id
func (_m *LinkGenerator) BlockLink(id flow.Identifier) (string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for BlockLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) string); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CollectionLink provides a mock function with given fields: id
func (_m *LinkGenerator) CollectionLink(id flow.Identifier) (string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for CollectionLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) string); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExecutionResultLink provides a mock function with given fields: id
func (_m *LinkGenerator) ExecutionResultLink(id flow.Identifier) (string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for ExecutionResultLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) string); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PayloadLink provides a mock function with given fields: id
func (_m *LinkGenerator) PayloadLink(id flow.Identifier) (string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for PayloadLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) string); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TransactionLink provides a mock function with given fields: id
func (_m *LinkGenerator) TransactionLink(id flow.Identifier) (string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for TransactionLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) string); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TransactionResultLink provides a mock function with given fields: id
func (_m *LinkGenerator) TransactionResultLink(id flow.Identifier) (string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for TransactionResultLink")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier) (string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier) string); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewLinkGenerator creates a new instance of LinkGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewLinkGenerator(t interface {
	mock.TestingT
	Cleanup(func())
}) *LinkGenerator {
	mock := &LinkGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
