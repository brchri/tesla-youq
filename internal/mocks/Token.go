// Code generated by mockery v2.36.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Token is an autogenerated mock type for the Token type
type Token struct {
	mock.Mock
}

type Token_Expecter struct {
	mock *mock.Mock
}

func (_m *Token) EXPECT() *Token_Expecter {
	return &Token_Expecter{mock: &_m.Mock}
}

// Done provides a mock function with given fields:
func (_m *Token) Done() <-chan struct{} {
	ret := _m.Called()

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// Token_Done_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Done'
type Token_Done_Call struct {
	*mock.Call
}

// Done is a helper method to define mock.On call
func (_e *Token_Expecter) Done() *Token_Done_Call {
	return &Token_Done_Call{Call: _e.mock.On("Done")}
}

func (_c *Token_Done_Call) Run(run func()) *Token_Done_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Token_Done_Call) Return(_a0 <-chan struct{}) *Token_Done_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Token_Done_Call) RunAndReturn(run func() <-chan struct{}) *Token_Done_Call {
	_c.Call.Return(run)
	return _c
}

// Error provides a mock function with given fields:
func (_m *Token) Error() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Token_Error_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Error'
type Token_Error_Call struct {
	*mock.Call
}

// Error is a helper method to define mock.On call
func (_e *Token_Expecter) Error() *Token_Error_Call {
	return &Token_Error_Call{Call: _e.mock.On("Error")}
}

func (_c *Token_Error_Call) Run(run func()) *Token_Error_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Token_Error_Call) Return(_a0 error) *Token_Error_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Token_Error_Call) RunAndReturn(run func() error) *Token_Error_Call {
	_c.Call.Return(run)
	return _c
}

// Wait provides a mock function with given fields:
func (_m *Token) Wait() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Token_Wait_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Wait'
type Token_Wait_Call struct {
	*mock.Call
}

// Wait is a helper method to define mock.On call
func (_e *Token_Expecter) Wait() *Token_Wait_Call {
	return &Token_Wait_Call{Call: _e.mock.On("Wait")}
}

func (_c *Token_Wait_Call) Run(run func()) *Token_Wait_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Token_Wait_Call) Return(_a0 bool) *Token_Wait_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Token_Wait_Call) RunAndReturn(run func() bool) *Token_Wait_Call {
	_c.Call.Return(run)
	return _c
}

// WaitTimeout provides a mock function with given fields: _a0
func (_m *Token) WaitTimeout(_a0 time.Duration) bool {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(time.Duration) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Token_WaitTimeout_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WaitTimeout'
type Token_WaitTimeout_Call struct {
	*mock.Call
}

// WaitTimeout is a helper method to define mock.On call
//   - _a0 time.Duration
func (_e *Token_Expecter) WaitTimeout(_a0 interface{}) *Token_WaitTimeout_Call {
	return &Token_WaitTimeout_Call{Call: _e.mock.On("WaitTimeout", _a0)}
}

func (_c *Token_WaitTimeout_Call) Run(run func(_a0 time.Duration)) *Token_WaitTimeout_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(time.Duration))
	})
	return _c
}

func (_c *Token_WaitTimeout_Call) Return(_a0 bool) *Token_WaitTimeout_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Token_WaitTimeout_Call) RunAndReturn(run func(time.Duration) bool) *Token_WaitTimeout_Call {
	_c.Call.Return(run)
	return _c
}

// NewToken creates a new instance of Token. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewToken(t interface {
	mock.TestingT
	Cleanup(func())
}) *Token {
	mock := &Token{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
