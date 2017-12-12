// Code generated by MockGen. DO NOT EDIT.
// Source: code.uber.internal/infra/peloton/.gen/peloton/api/respool (interfaces: ResourceManagerYARPCClient)

package mocks

import (
	context "context"
	reflect "reflect"

	respool "code.uber.internal/infra/peloton/.gen/peloton/api/respool"
	gomock "github.com/golang/mock/gomock"
	yarpc "go.uber.org/yarpc"
)

// MockResourceManagerYARPCClient is a mock of ResourceManagerYARPCClient interface
type MockResourceManagerYARPCClient struct {
	ctrl     *gomock.Controller
	recorder *MockResourceManagerYARPCClientMockRecorder
}

// MockResourceManagerYARPCClientMockRecorder is the mock recorder for MockResourceManagerYARPCClient
type MockResourceManagerYARPCClientMockRecorder struct {
	mock *MockResourceManagerYARPCClient
}

// NewMockResourceManagerYARPCClient creates a new mock instance
func NewMockResourceManagerYARPCClient(ctrl *gomock.Controller) *MockResourceManagerYARPCClient {
	mock := &MockResourceManagerYARPCClient{ctrl: ctrl}
	mock.recorder = &MockResourceManagerYARPCClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockResourceManagerYARPCClient) EXPECT() *MockResourceManagerYARPCClientMockRecorder {
	return _m.recorder
}

// CreateResourcePool mocks base method
func (_m *MockResourceManagerYARPCClient) CreateResourcePool(_param0 context.Context, _param1 *respool.CreateRequest, _param2 ...yarpc.CallOption) (*respool.CreateResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "CreateResourcePool", _s...)
	ret0, _ := ret[0].(*respool.CreateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateResourcePool indicates an expected call of CreateResourcePool
func (_mr *MockResourceManagerYARPCClientMockRecorder) CreateResourcePool(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "CreateResourcePool", reflect.TypeOf((*MockResourceManagerYARPCClient)(nil).CreateResourcePool), _s...)
}

// DeleteResourcePool mocks base method
func (_m *MockResourceManagerYARPCClient) DeleteResourcePool(_param0 context.Context, _param1 *respool.DeleteRequest, _param2 ...yarpc.CallOption) (*respool.DeleteResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "DeleteResourcePool", _s...)
	ret0, _ := ret[0].(*respool.DeleteResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteResourcePool indicates an expected call of DeleteResourcePool
func (_mr *MockResourceManagerYARPCClientMockRecorder) DeleteResourcePool(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "DeleteResourcePool", reflect.TypeOf((*MockResourceManagerYARPCClient)(nil).DeleteResourcePool), _s...)
}

// GetResourcePool mocks base method
func (_m *MockResourceManagerYARPCClient) GetResourcePool(_param0 context.Context, _param1 *respool.GetRequest, _param2 ...yarpc.CallOption) (*respool.GetResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "GetResourcePool", _s...)
	ret0, _ := ret[0].(*respool.GetResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetResourcePool indicates an expected call of GetResourcePool
func (_mr *MockResourceManagerYARPCClientMockRecorder) GetResourcePool(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "GetResourcePool", reflect.TypeOf((*MockResourceManagerYARPCClient)(nil).GetResourcePool), _s...)
}

// LookupResourcePoolID mocks base method
func (_m *MockResourceManagerYARPCClient) LookupResourcePoolID(_param0 context.Context, _param1 *respool.LookupRequest, _param2 ...yarpc.CallOption) (*respool.LookupResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "LookupResourcePoolID", _s...)
	ret0, _ := ret[0].(*respool.LookupResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LookupResourcePoolID indicates an expected call of LookupResourcePoolID
func (_mr *MockResourceManagerYARPCClientMockRecorder) LookupResourcePoolID(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "LookupResourcePoolID", reflect.TypeOf((*MockResourceManagerYARPCClient)(nil).LookupResourcePoolID), _s...)
}

// Query mocks base method
func (_m *MockResourceManagerYARPCClient) Query(_param0 context.Context, _param1 *respool.QueryRequest, _param2 ...yarpc.CallOption) (*respool.QueryResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "Query", _s...)
	ret0, _ := ret[0].(*respool.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Query indicates an expected call of Query
func (_mr *MockResourceManagerYARPCClientMockRecorder) Query(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Query", reflect.TypeOf((*MockResourceManagerYARPCClient)(nil).Query), _s...)
}

// UpdateResourcePool mocks base method
func (_m *MockResourceManagerYARPCClient) UpdateResourcePool(_param0 context.Context, _param1 *respool.UpdateRequest, _param2 ...yarpc.CallOption) (*respool.UpdateResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "UpdateResourcePool", _s...)
	ret0, _ := ret[0].(*respool.UpdateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateResourcePool indicates an expected call of UpdateResourcePool
func (_mr *MockResourceManagerYARPCClientMockRecorder) UpdateResourcePool(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "UpdateResourcePool", reflect.TypeOf((*MockResourceManagerYARPCClient)(nil).UpdateResourcePool), _s...)
}