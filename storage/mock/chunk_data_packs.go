// Code generated by mockery v2.13.1. DO NOT EDIT.

package mock

import (
	flow "github.com/onflow/flow-go/model/flow"
	mock "github.com/stretchr/testify/mock"

	storage "github.com/onflow/flow-go/storage"
)

// ChunkDataPacks is an autogenerated mock type for the ChunkDataPacks type
type ChunkDataPacks struct {
	mock.Mock
}

// BatchRemove provides a mock function with given fields: chunkID, batch
func (_m *ChunkDataPacks) BatchRemove(chunkID flow.Identifier, batch storage.BatchStorage) error {
	ret := _m.Called(chunkID, batch)

	var r0 error
	if rf, ok := ret.Get(0).(func(flow.Identifier, storage.BatchStorage) error); ok {
		r0 = rf(chunkID, batch)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BatchStore provides a mock function with given fields: c, batch
func (_m *ChunkDataPacks) BatchStore(c *flow.ChunkDataPack, batch storage.BatchStorage) error {
	ret := _m.Called(c, batch)

	var r0 error
	if rf, ok := ret.Get(0).(func(*flow.ChunkDataPack, storage.BatchStorage) error); ok {
		r0 = rf(c, batch)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ByChunkID provides a mock function with given fields: chunkID
func (_m *ChunkDataPacks) ByChunkID(chunkID flow.Identifier) (*flow.ChunkDataPack, error) {
	ret := _m.Called(chunkID)

	var r0 *flow.ChunkDataPack
	if rf, ok := ret.Get(0).(func(flow.Identifier) *flow.ChunkDataPack); ok {
		r0 = rf(chunkID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*flow.ChunkDataPack)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Identifier) error); ok {
		r1 = rf(chunkID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Store provides a mock function with given fields: c
func (_m *ChunkDataPacks) Store(c *flow.ChunkDataPack) error {
	ret := _m.Called(c)

	var r0 error
	if rf, ok := ret.Get(0).(func(*flow.ChunkDataPack) error); ok {
		r0 = rf(c)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewChunkDataPacks interface {
	mock.TestingT
	Cleanup(func())
}

// NewChunkDataPacks creates a new instance of ChunkDataPacks. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewChunkDataPacks(t mockConstructorTestingTNewChunkDataPacks) *ChunkDataPacks {
	mock := &ChunkDataPacks{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
