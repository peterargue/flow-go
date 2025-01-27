// Code generated by mockery v2.21.4. DO NOT EDIT.

package mocks

import (
	crypto "github.com/onflow/flow-go/crypto"
	flow "github.com/onflow/flow-go/model/flow"

	hotstuff "github.com/onflow/flow-go/consensus/hotstuff"

	mock "github.com/stretchr/testify/mock"
)

// TimeoutSignatureAggregator is an autogenerated mock type for the TimeoutSignatureAggregator type
type TimeoutSignatureAggregator struct {
	mock.Mock
}

// Aggregate provides a mock function with given fields:
func (_m *TimeoutSignatureAggregator) Aggregate() ([]hotstuff.TimeoutSignerInfo, crypto.Signature, error) {
	ret := _m.Called()

	var r0 []hotstuff.TimeoutSignerInfo
	var r1 crypto.Signature
	var r2 error
	if rf, ok := ret.Get(0).(func() ([]hotstuff.TimeoutSignerInfo, crypto.Signature, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []hotstuff.TimeoutSignerInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]hotstuff.TimeoutSignerInfo)
		}
	}

	if rf, ok := ret.Get(1).(func() crypto.Signature); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(crypto.Signature)
		}
	}

	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// TotalWeight provides a mock function with given fields:
func (_m *TimeoutSignatureAggregator) TotalWeight() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// VerifyAndAdd provides a mock function with given fields: signerID, sig, newestQCView
func (_m *TimeoutSignatureAggregator) VerifyAndAdd(signerID flow.Identifier, sig crypto.Signature, newestQCView uint64) (uint64, error) {
	ret := _m.Called(signerID, sig, newestQCView)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(flow.Identifier, crypto.Signature, uint64) (uint64, error)); ok {
		return rf(signerID, sig, newestQCView)
	}
	if rf, ok := ret.Get(0).(func(flow.Identifier, crypto.Signature, uint64) uint64); ok {
		r0 = rf(signerID, sig, newestQCView)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(flow.Identifier, crypto.Signature, uint64) error); ok {
		r1 = rf(signerID, sig, newestQCView)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// View provides a mock function with given fields:
func (_m *TimeoutSignatureAggregator) View() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

type mockConstructorTestingTNewTimeoutSignatureAggregator interface {
	mock.TestingT
	Cleanup(func())
}

// NewTimeoutSignatureAggregator creates a new instance of TimeoutSignatureAggregator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewTimeoutSignatureAggregator(t mockConstructorTestingTNewTimeoutSignatureAggregator) *TimeoutSignatureAggregator {
	mock := &TimeoutSignatureAggregator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
