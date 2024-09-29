// Code generated by mockery v2.20.2. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/alexandernizov/grpcmessanger/internal/domain"

	mock "github.com/stretchr/testify/mock"
)

// AuthProvider is an autogenerated mock type for the AuthProvider type
type AuthProvider struct {
	mock.Mock
}

// Login provides a mock function with given fields: ctx, login, password
func (_m *AuthProvider) Login(ctx context.Context, login string, password string) (*domain.Tokens, error) {
	ret := _m.Called(ctx, login, password)

	var r0 *domain.Tokens
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*domain.Tokens, error)); ok {
		return rf(ctx, login, password)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *domain.Tokens); ok {
		r0 = rf(ctx, login, password)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.Tokens)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, login, password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Refresh provides a mock function with given fields: ctx, refreshToken
func (_m *AuthProvider) Refresh(ctx context.Context, refreshToken string) (*domain.Tokens, error) {
	ret := _m.Called(ctx, refreshToken)

	var r0 *domain.Tokens
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.Tokens, error)); ok {
		return rf(ctx, refreshToken)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.Tokens); ok {
		r0 = rf(ctx, refreshToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.Tokens)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, refreshToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Register provides a mock function with given fields: ctx, login, password
func (_m *AuthProvider) Register(ctx context.Context, login string, password string) (*domain.User, error) {
	ret := _m.Called(ctx, login, password)

	var r0 *domain.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*domain.User, error)); ok {
		return rf(ctx, login, password)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *domain.User); ok {
		r0 = rf(ctx, login, password)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, login, password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewAuthProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewAuthProvider creates a new instance of AuthProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAuthProvider(t mockConstructorTestingTNewAuthProvider) *AuthProvider {
	mock := &AuthProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
