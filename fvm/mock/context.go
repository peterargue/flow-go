// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import flow "github.com/dapperlabs/flow-go/model/flow"
import fvm "github.com/dapperlabs/flow-go/fvm"
import mock "github.com/stretchr/testify/mock"
import runtime "github.com/onflow/cadence/runtime"

// Context is an autogenerated mock type for the Context type
type Context struct {
	mock.Mock
}

// Environment provides a mock function with given fields: ledger
func (_m *Context) Environment(ledger fvm.Ledger) fvm.Environment {
	ret := _m.Called(ledger)

	var r0 fvm.Environment
	if rf, ok := ret.Get(0).(func(fvm.Ledger) fvm.Environment); ok {
		r0 = rf(ledger)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fvm.Environment)
		}
	}

	return r0
}

// GetAccount provides a mock function with given fields: address, ledger
func (_m *Context) GetAccount(address flow.Address, ledger fvm.Ledger) (*flow.Account, error) {
	ret := _m.Called(address, ledger)

	var r0 *flow.Account
	if rf, ok := ret.Get(0).(func(flow.Address, fvm.Ledger) *flow.Account); ok {
		r0 = rf(address, ledger)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*flow.Account)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Address, fvm.Ledger) error); ok {
		r1 = rf(address, ledger)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Invoke provides a mock function with given fields: i, ledger
func (_m *Context) Invoke(i fvm.Invokable, ledger fvm.Ledger) (*fvm.InvocationResult, error) {
	ret := _m.Called(i, ledger)

	var r0 *fvm.InvocationResult
	if rf, ok := ret.Get(0).(func(fvm.Invokable, fvm.Ledger) *fvm.InvocationResult); ok {
		r0 = rf(i, ledger)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*fvm.InvocationResult)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(fvm.Invokable, fvm.Ledger) error); ok {
		r1 = rf(i, ledger)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewChild provides a mock function with given fields: opts
func (_m *Context) NewChild(opts ...fvm.Option) fvm.Context {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 fvm.Context
	if rf, ok := ret.Get(0).(func(...fvm.Option) fvm.Context); ok {
		r0 = rf(opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fvm.Context)
		}
	}

	return r0
}

// Options provides a mock function with given fields:
func (_m *Context) Options() fvm.Options {
	ret := _m.Called()

	var r0 fvm.Options
	if rf, ok := ret.Get(0).(func() fvm.Options); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(fvm.Options)
	}

	return r0
}

// Parse provides a mock function with given fields: i, ledger
func (_m *Context) Parse(i fvm.Invokable, ledger fvm.Ledger) (fvm.Invokable, error) {
	ret := _m.Called(i, ledger)

	var r0 fvm.Invokable
	if rf, ok := ret.Get(0).(func(fvm.Invokable, fvm.Ledger) fvm.Invokable); ok {
		r0 = rf(i, ledger)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fvm.Invokable)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(fvm.Invokable, fvm.Ledger) error); ok {
		r1 = rf(i, ledger)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Runtime provides a mock function with given fields:
func (_m *Context) Runtime() runtime.Runtime {
	ret := _m.Called()

	var r0 runtime.Runtime
	if rf, ok := ret.Get(0).(func() runtime.Runtime); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(runtime.Runtime)
		}
	}

	return r0
}
