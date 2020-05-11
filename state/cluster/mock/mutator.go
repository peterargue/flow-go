// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import cluster "github.com/dapperlabs/flow-go/model/cluster"
import mock "github.com/stretchr/testify/mock"

// Mutator is an autogenerated mock type for the Mutator type
type Mutator struct {
	mock.Mock
}

// Bootstrap provides a mock function with given fields: genesis
func (_m *Mutator) Bootstrap(genesis *cluster.Block) error {
	ret := _m.Called(genesis)

	var r0 error
	if rf, ok := ret.Get(0).(func(*cluster.Block) error); ok {
		r0 = rf(genesis)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Extend provides a mock function with given fields: block
func (_m *Mutator) Extend(block *cluster.Block) error {
	ret := _m.Called(block)

	var r0 error
	if rf, ok := ret.Get(0).(func(*cluster.Block) error); ok {
		r0 = rf(block)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
