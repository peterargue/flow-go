// Code generated by mockery v2.21.4. DO NOT EDIT.

package mockp2p

import (
	p2p "github.com/onflow/flow-go/network/underlay"

	mock "github.com/stretchr/testify/mock"
)

// NetworkConfigOption is an autogenerated mock type for the NetworkConfigOption type
type NetworkConfigOption struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *NetworkConfigOption) Execute(_a0 *p2p.NetworkConfig) {
	_m.Called(_a0)
}

type mockConstructorTestingTNewNetworkConfigOption interface {
	mock.TestingT
	Cleanup(func())
}

// NewNetworkConfigOption creates a new instance of NetworkConfigOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNetworkConfigOption(t mockConstructorTestingTNewNetworkConfigOption) *NetworkConfigOption {
	mock := &NetworkConfigOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
