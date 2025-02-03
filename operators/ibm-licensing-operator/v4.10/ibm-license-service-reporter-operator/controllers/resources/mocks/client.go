//
// Copyright 2023 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package mocks

import (
	"context"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockClient is mocking client.Client, do not create it, use GetMockClient to ensure the type in the interface is filled
type MockClient struct {
	storedObject client.Object
	UpdateCount  int
}

func GetMockClient(object client.Object) MockClient {
	return MockClient{storedObject: object}
}

//goland:noinspection GoUnusedParameter
func (m MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.storedObject != nil && key.Namespace == m.storedObject.GetNamespace() && key.Name == m.storedObject.GetName() {
		overrideFirstParamWithSecond(obj, m.storedObject)
		return nil
	}
	return apierrors.NewNotFound(schema.GroupResource{
		Group:    "mockClientGroup",
		Resource: "mockClientResource",
	}, key.Name)
}

//goland:noinspection GoUnusedParameter
func (m MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	panic("implement me")
}

//goland:noinspection GoUnusedParameter
func (m MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	// TODO: add handling if exists
	// TODO: maybe change somehow else to deepcopy
	overrideFirstParamWithSecond(m.storedObject, obj)
	return nil
}

//goland:noinspection GoUnusedParameter
func (m MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	panic("implement me")
}

//goland:noinspection GoUnusedParameter
func (m MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	//TODO: add handling if exists?
	m.UpdateCount++
	overrideFirstParamWithSecond(m.storedObject, obj)
	return nil
}

func overrideFirstParamWithSecond(to, from client.Object) {
	reflect.Indirect(reflect.ValueOf(to)).Set(reflect.Indirect(reflect.ValueOf(from)))
}

//goland:noinspection GoUnusedParameter
func (m MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	panic("implement me")
}

//goland:noinspection GoUnusedParameter
func (m MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	panic("implement me")
}

func (m MockClient) Status() client.StatusWriter {
	panic("implement me")
}

func (m MockClient) Scheme() *runtime.Scheme {
	panic("implement me")
}

func (m MockClient) RESTMapper() meta.RESTMapper {
	panic("implement me")
}
