// Code generated by mockery v2.43.2. DO NOT EDIT.

package mock

import (
	crypto "github.com/onflow/crypto"
	flow "github.com/onflow/flow-go/model/flow"

	mock "github.com/stretchr/testify/mock"
)

// DKGStateReader is an autogenerated mock type for the DKGStateReader type
type DKGStateReader struct {
	mock.Mock
}

// GetDKGStarted provides a mock function with given fields: epochCounter
func (_m *DKGStateReader) GetDKGStarted(epochCounter uint64) (bool, error) {
	ret := _m.Called(epochCounter)

	if len(ret) == 0 {
		panic("no return value specified for GetDKGStarted")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (bool, error)); ok {
		return rf(epochCounter)
	}
	if rf, ok := ret.Get(0).(func(uint64) bool); ok {
		r0 = rf(epochCounter)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(epochCounter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDKGState provides a mock function with given fields: epochCounter
func (_m *DKGStateReader) GetDKGState(epochCounter uint64) (flow.DKGState, error) {
	ret := _m.Called(epochCounter)

	if len(ret) == 0 {
		panic("no return value specified for GetDKGState")
	}

	var r0 flow.DKGState
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (flow.DKGState, error)); ok {
		return rf(epochCounter)
	}
	if rf, ok := ret.Get(0).(func(uint64) flow.DKGState); ok {
		r0 = rf(epochCounter)
	} else {
		r0 = ret.Get(0).(flow.DKGState)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(epochCounter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveMyBeaconPrivateKey provides a mock function with given fields: epochCounter
func (_m *DKGStateReader) RetrieveMyBeaconPrivateKey(epochCounter uint64) (crypto.PrivateKey, bool, error) {
	ret := _m.Called(epochCounter)

	if len(ret) == 0 {
		panic("no return value specified for RetrieveMyBeaconPrivateKey")
	}

	var r0 crypto.PrivateKey
	var r1 bool
	var r2 error
	if rf, ok := ret.Get(0).(func(uint64) (crypto.PrivateKey, bool, error)); ok {
		return rf(epochCounter)
	}
	if rf, ok := ret.Get(0).(func(uint64) crypto.PrivateKey); ok {
		r0 = rf(epochCounter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(crypto.PrivateKey)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) bool); ok {
		r1 = rf(epochCounter)
	} else {
		r1 = ret.Get(1).(bool)
	}

	if rf, ok := ret.Get(2).(func(uint64) error); ok {
		r2 = rf(epochCounter)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UnsafeRetrieveMyBeaconPrivateKey provides a mock function with given fields: epochCounter
func (_m *DKGStateReader) UnsafeRetrieveMyBeaconPrivateKey(epochCounter uint64) (crypto.PrivateKey, error) {
	ret := _m.Called(epochCounter)

	if len(ret) == 0 {
		panic("no return value specified for UnsafeRetrieveMyBeaconPrivateKey")
	}

	var r0 crypto.PrivateKey
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (crypto.PrivateKey, error)); ok {
		return rf(epochCounter)
	}
	if rf, ok := ret.Get(0).(func(uint64) crypto.PrivateKey); ok {
		r0 = rf(epochCounter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(crypto.PrivateKey)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(epochCounter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDKGStateReader creates a new instance of DKGStateReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDKGStateReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *DKGStateReader {
	mock := &DKGStateReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
