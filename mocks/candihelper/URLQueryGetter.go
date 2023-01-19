// Code generated by mockery v2.13.1. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// URLQueryGetter is an autogenerated mock type for the URLQueryGetter type
type URLQueryGetter struct {
	mock.Mock
}

// Get provides a mock function with given fields: key
func (_m *URLQueryGetter) Get(key string) string {
	ret := _m.Called(key)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type mockConstructorTestingTNewURLQueryGetter interface {
	mock.TestingT
	Cleanup(func())
}

// NewURLQueryGetter creates a new instance of URLQueryGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewURLQueryGetter(t mockConstructorTestingTNewURLQueryGetter) *URLQueryGetter {
	mock := &URLQueryGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
