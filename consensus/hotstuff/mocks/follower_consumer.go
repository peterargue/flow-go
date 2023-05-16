// Code generated by mockery v2.21.4. DO NOT EDIT.

package mocks

import (
	model "github.com/onflow/flow-go/consensus/hotstuff/model"
	mock "github.com/stretchr/testify/mock"
)

// FollowerConsumer is an autogenerated mock type for the FollowerConsumer type
type FollowerConsumer struct {
	mock.Mock
}

// OnBlockIncorporated provides a mock function with given fields: _a0
func (_m *FollowerConsumer) OnBlockIncorporated(_a0 *model.Block) {
	_m.Called(_a0)
}

// OnDoubleProposeDetected provides a mock function with given fields: _a0, _a1
func (_m *FollowerConsumer) OnDoubleProposeDetected(_a0 *model.Block, _a1 *model.Block) {
	_m.Called(_a0, _a1)
}

// OnFinalizedBlock provides a mock function with given fields: _a0
func (_m *FollowerConsumer) OnFinalizedBlock(_a0 *model.Block) {
	_m.Called(_a0)
}

// OnInvalidBlockDetected provides a mock function with given fields: err
func (_m *FollowerConsumer) OnInvalidBlockDetected(err model.InvalidProposalError) {
	_m.Called(err)
}

type mockConstructorTestingTNewFollowerConsumer interface {
	mock.TestingT
	Cleanup(func())
}

// NewFollowerConsumer creates a new instance of FollowerConsumer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFollowerConsumer(t mockConstructorTestingTNewFollowerConsumer) *FollowerConsumer {
	mock := &FollowerConsumer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
