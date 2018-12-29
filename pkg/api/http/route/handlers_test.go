//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2018] Last.Backend LLC
// All Rights Reserved.
//
// NOTICE:  All information contained herein is, and remains
// the property of Last.Backend LLC and its suppliers,
// if any.  The intellectual and technical concepts contained
// herein are proprietary to Last.Backend LLC
// and its suppliers and may be covered by Russian Federation and Foreign Patents,
// patents in process, and are protected by trade secret or copyright law.
// Dissemination of this information or reproduction of this material
// is strictly forbidden unless prior written permission is obtained
// from Last.Backend LLC.
//

package route_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/lastbackend/lastbackend/pkg/api/envs"
	"github.com/lastbackend/lastbackend/pkg/api/http/route"
	"github.com/lastbackend/lastbackend/pkg/api/types/v1"
	"github.com/lastbackend/lastbackend/pkg/api/types/v1/request"
	"github.com/lastbackend/lastbackend/pkg/api/types/v1/views"
	"github.com/lastbackend/lastbackend/pkg/distribution/errors"
	"github.com/lastbackend/lastbackend/pkg/distribution/types"
	"github.com/lastbackend/lastbackend/pkg/storage"
	"github.com/stretchr/testify/assert"
)

// Testing RouteInfoH handler
func TestRouteInfo(t *testing.T) {

	var ctx = context.Background()

	stg, _ := storage.Get("mock")
	envs.Get().SetStorage(stg)

	ns1 := getNamespaceAsset("demo", "")
	ns2 := getNamespaceAsset("test", "")
	r1 := getRouteAsset(ns1.Meta.Name, "demo")
	r2 := getRouteAsset(ns2.Meta.Name, "test")

	type fields struct {
		stg storage.Storage
	}

	type args struct {
		ctx       context.Context
		namespace *types.Namespace
		route     *types.Route
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		headers      map[string]string
		handler      func(http.ResponseWriter, *http.Request)
		err          string
		want         *views.Route
		wantErr      bool
		expectedCode int
	}{
		{
			name:         "checking get route if not exists",
			args:         args{ctx, ns1, r2},
			fields:       fields{stg},
			handler:      route.RouteInfoH,
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Route not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking get route if namespace not exists",
			args:         args{ctx, ns2, r1},
			fields:       fields{stg},
			handler:      route.RouteInfoH,
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Namespace not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking get route successfully",
			args:         args{ctx, ns1, r1},
			fields:       fields{stg},
			handler:      route.RouteInfoH,
			want:         v1.View().Route().New(r1),
			wantErr:      false,
			expectedCode: http.StatusOK,
		},
	}

	clear := func() {
		err := stg.Del(context.Background(), stg.Collection().Namespace(), types.EmptyString)
		assert.NoError(t, err)

		err = stg.Del(context.Background(), stg.Collection().Route(), types.EmptyString)
		assert.NoError(t, err)
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			clear()
			defer clear()

			err := tc.fields.stg.Put(context.Background(), stg.Collection().Namespace(), tc.fields.stg.Key().Namespace(ns1.Meta.Name), ns1, nil)
			assert.NoError(t, err)

			err = stg.Put(context.Background(), stg.Collection().Route(), stg.Key().Route(r1.Meta.Namespace, r1.Meta.Name), r1, nil)
			assert.NoError(t, err)

			// Create assert request to pass to our handler. We don't have any query parameters for now, so we'll
			// pass 'nil' as the third parameter.
			req, err := http.NewRequest("GET", fmt.Sprintf("/namespace/%s/route/%s", tc.args.namespace.Meta.Name, tc.args.route.Meta.Name), nil)
			assert.NoError(t, err)

			if tc.headers != nil {
				for key, val := range tc.headers {
					req.Header.Set(key, val)
				}
			}

			r := mux.NewRouter()
			r.HandleFunc("/namespace/{namespace}/route/{route}", tc.handler)

			setRequestVars(r, req)

			// We create assert ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			res := httptest.NewRecorder()

			// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
			// directly and pass in our Request and ResponseRecorder.
			r.ServeHTTP(res, req)

			// Check the status code is what we expect.
			assert.Equal(t, tc.expectedCode, res.Code, "status code not equal")

			body, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err)

			if tc.wantErr && res.Code != 200 {
				assert.Equal(t, tc.err, string(body), "incorrect status code")
			} else {

				n := new(views.Route)
				err := json.Unmarshal(body, &n)
				assert.NoError(t, err)

			}
		})
	}

}

// Testing RouteListH handler
func TestRouteList(t *testing.T) {

	var ctx = context.Background()

	stg, _ := storage.Get("mock")
	envs.Get().SetStorage(stg)

	ns1 := getNamespaceAsset("demo", "")
	ns2 := getNamespaceAsset("test", "")
	r1 := getRouteAsset(ns1.Meta.Name, "demo")
	r2 := getRouteAsset(ns1.Meta.Name, "test")

	rl := types.NewRouteMap()
	rl.Items[r1.SelfLink()] = r1
	rl.Items[r2.SelfLink()] = r2

	type fields struct {
		stg storage.Storage
	}

	type args struct {
		ctx       context.Context
		namespace *types.Namespace
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		headers      map[string]string
		handler      func(http.ResponseWriter, *http.Request)
		err          string
		want         *types.RouteMap
		wantErr      bool
		expectedCode int
	}{
		{
			name:         "checking get routes list if namespace not found",
			args:         args{ctx, ns2},
			fields:       fields{stg},
			handler:      route.RouteListH,
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Namespace not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking get routes list successfully",
			args:         args{ctx, ns1},
			fields:       fields{stg},
			handler:      route.RouteListH,
			want:         rl,
			wantErr:      false,
			expectedCode: http.StatusOK,
		},
	}

	clear := func() {
		err := envs.Get().GetStorage().Del(context.Background(), stg.Collection().Namespace(), types.EmptyString)
		assert.NoError(t, err)

		err = stg.Del(context.Background(), stg.Collection().Route(), types.EmptyString)
		assert.NoError(t, err)
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			clear()
			defer clear()

			err := tc.fields.stg.Put(context.Background(), stg.Collection().Namespace(), tc.fields.stg.Key().Namespace(ns1.Meta.Name), ns1, nil)
			assert.NoError(t, err)

			err = stg.Put(context.Background(), stg.Collection().Route(), stg.Key().Route(r1.Meta.Namespace, r1.Meta.Name), r1, nil)
			assert.NoError(t, err)

			err = stg.Put(context.Background(), stg.Collection().Route(), stg.Key().Route(r2.Meta.Namespace, r2.Meta.Name), r2, nil)
			assert.NoError(t, err)

			// Create assert request to pass to our handler. We don't have any query parameters for now, so we'll
			// pass 'nil' as the third parameter.
			req, err := http.NewRequest("GET", fmt.Sprintf("/namespace/%s", tc.args.namespace.Meta.Name), nil)
			assert.NoError(t, err)

			if tc.headers != nil {
				for key, val := range tc.headers {
					req.Header.Set(key, val)
				}
			}

			r := mux.NewRouter()
			r.HandleFunc("/namespace/{namespace}", tc.handler)

			setRequestVars(r, req)

			// We create assert ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			res := httptest.NewRecorder()

			// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
			// directly and pass in our Request and ResponseRecorder.
			r.ServeHTTP(res, req)

			// Check the status code is what we expect.
			if !assert.Equal(t, tc.expectedCode, res.Code, "status code not equal") {
				return
			}

			body, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err)

			if tc.wantErr && res.Code != 200 {
				assert.Equal(t, tc.err, string(body), "incorrect status code")
			} else {

				r := new(views.RouteList)
				err := json.Unmarshal(body, &r)
				assert.NoError(t, err)

				for _, item := range *r {
					if _, ok := tc.want.Items[item.Meta.SelfLink]; !ok {
						assert.Error(t, errors.New("not equals"))
					}
				}
			}
		})
	}

}

// Testing RouteCreateH handler
func TestRouteCreate(t *testing.T) {

	var ctx = context.Background()

	stg, _ := storage.Get("mock")
	envs.Get().SetStorage(stg)

	ns1 := getNamespaceAsset("demo", "")
	ns2 := getNamespaceAsset("test", "")

	sv1 := getServiceAsset(ns1.Meta.Name, "demo", "")

	r1 := getRouteAsset(ns1.Meta.Name, "demo")

	sl := new(types.ServiceList)
	sl.Items = append(sl.Items, sv1)

	mf := getRouteManifest(sv1.Meta.Name)
	mf.SetRouteSpec(r1, sl)

	mf1, _ := mf.ToJson()
	mf2, _ := getRouteManifest("not_found").ToJson()

	type fields struct {
		stg storage.Storage
	}

	type args struct {
		ctx       context.Context
		namespace *types.Namespace
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		headers      map[string]string
		handler      func(http.ResponseWriter, *http.Request)
		data         string
		err          string
		want         *views.Route
		wantErr      bool
		expectedCode int
	}{
		// TODO: need checking for unique
		{
			name:         "checking create route if namespace not found",
			args:         args{ctx, ns2},
			fields:       fields{stg},
			handler:      route.RouteCreateH,
			data:         string(mf1),
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Namespace not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking create route if service not found in rules",
			args:         args{ctx, ns1},
			fields:       fields{stg},
			handler:      route.RouteCreateH,
			data:         string(mf2),
			err:          "{\"code\":400,\"status\":\"Bad Parameter\",\"message\":\"Bad rules parameter\"}",
			wantErr:      true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "check create route if failed incoming json data",
			args:         args{ctx, ns1},
			fields:       fields{stg},
			handler:      route.RouteCreateH,
			data:         "{name:demo}",
			err:          "{\"code\":400,\"status\":\"Incorrect Json\",\"message\":\"Incorrect json\"}",
			wantErr:      true,
			expectedCode: http.StatusBadRequest,
		},
		// TODO: need checking incoming data for validity
		{
			name:         "check create route success",
			args:         args{ctx, ns1},
			fields:       fields{stg},
			handler:      route.RouteCreateH,
			data:         string(mf1),
			want:         v1.View().Route().New(r1),
			wantErr:      false,
			expectedCode: http.StatusOK,
		},
	}

	clear := func() {
		err := envs.Get().GetStorage().Del(context.Background(), stg.Collection().Namespace(), types.EmptyString)
		assert.NoError(t, err)

		err = stg.Del(context.Background(), stg.Collection().Route(), types.EmptyString)
		assert.NoError(t, err)

		err = envs.Get().GetStorage().Del(context.Background(), stg.Collection().Service(), types.EmptyString)
		assert.NoError(t, err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			clear()
			defer clear()

			err := tc.fields.stg.Put(context.Background(), stg.Collection().Namespace(), tc.fields.stg.Key().Namespace(ns1.Meta.Name), ns1, nil)
			assert.NoError(t, err)

			err = tc.fields.stg.Put(context.Background(), stg.Collection().Service(), stg.Key().Service(sv1.Meta.Namespace, sv1.Meta.Name), sv1, nil)
			assert.NoError(t, err)

			// Create assert request to pass to our handler. We don't have any query parameters for now, so we'll
			// pass 'nil' as the third parameter.
			req, err := http.NewRequest("POST", fmt.Sprintf("/namespace/%s/route", tc.args.namespace.Meta.Name), strings.NewReader(tc.data))
			assert.NoError(t, err)

			if tc.headers != nil {
				for key, val := range tc.headers {
					req.Header.Set(key, val)
				}
			}

			r := mux.NewRouter()
			r.HandleFunc("/namespace/{namespace}/route", tc.handler)

			setRequestVars(r, req)

			// We create assert ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			res := httptest.NewRecorder()

			// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
			// directly and pass in our Request and ResponseRecorder.
			r.ServeHTTP(res, req)

			// Check the status code is what we expect.
			body, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err)

			if !assert.Equal(t, tc.expectedCode, res.Code, "status code not equal") {
				t.Error(string(body))
				return
			}

			if tc.wantErr {
				assert.Equal(t, tc.err, string(body), "incorrect code message")
			} else {

				got := new(types.Route)
				err := tc.fields.stg.Get(tc.args.ctx, stg.Collection().Route(), stg.Key().Route(tc.args.namespace.Meta.Name, tc.want.Meta.Name), got, nil)
				assert.NoError(t, err)
				if assert.NotEmpty(t, got, "route is empty") {

					if !assert.Equal(t, tc.want.Meta.Name, got.Meta.Name, "names mismatch") {
						return
					}

					if !assert.Equal(t, len(tc.want.Spec.Rules), len(got.Spec.Rules), "rules count mismatch") {
						return
					}

					assert.Equal(t, tc.want.Spec.Rules[0].Endpoint, got.Spec.Rules[0].Endpoint, "endpoints mismatch")
				}
			}
		})
	}

}

// Testing RouteUpdateH handler
func TestRouteUpdate(t *testing.T) {

	var ctx = context.Background()

	stg, _ := storage.Get("mock")
	envs.Get().SetStorage(stg)

	ns1 := getNamespaceAsset("demo", "")
	ns2 := getNamespaceAsset("test", "")

	sv1 := getServiceAsset(ns1.Meta.Name, "demo", "")
	sv2 := getServiceAsset(ns1.Meta.Name, "test1", "")
	sv3 := getServiceAsset(ns1.Meta.Name, "test2", "")

	r1 := getRouteAsset(ns1.Meta.Name, "demo")
	r2 := getRouteAsset(ns1.Meta.Name, "test")
	r3 := getRouteAsset(ns1.Meta.Name, "demo")

	r3.Spec.Rules = append(r3.Spec.Rules, types.RouteRule{
		Path:     "/",
		Endpoint: fmt.Sprintf("%s.%s", ns1.Meta.Name, sv1.Meta.Name),
		Port:     80,
	})

	mf1, _ := getRouteManifest(sv1.Meta.Name).ToJson()
	mf2, _ := getRouteManifest(sv1.Meta.Name).ToJson()
	mf3, _ := getRouteManifest(sv3.Meta.Name).ToJson()

	type fields struct {
		stg storage.Storage
	}

	type args struct {
		ctx       context.Context
		namespace *types.Namespace
		route     *types.Route
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		headers      map[string]string
		handler      func(http.ResponseWriter, *http.Request)
		data         string
		err          string
		want         *views.Route
		wantErr      bool
		expectedCode int
	}{
		{
			name:         "checking update route if name not exists",
			args:         args{ctx, ns1, r2},
			fields:       fields{stg},
			handler:      route.RouteUpdateH,
			data:         string(mf3),
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Route not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking update route if namespace not found",
			args:         args{ctx, ns2, r1},
			fields:       fields{stg},
			handler:      route.RouteUpdateH,
			data:         string(mf2),
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Namespace not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking update route if service not found",
			args:         args{ctx, ns2, r1},
			fields:       fields{stg},
			handler:      route.RouteUpdateH,
			data:         string(mf1),
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Namespace not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "check update route if failed incoming json data",
			args:         args{ctx, ns1, r1},
			fields:       fields{stg},
			handler:      route.RouteUpdateH,
			data:         "{name:demo}",
			err:          "{\"code\":400,\"status\":\"Incorrect Json\",\"message\":\"Incorrect json\"}",
			wantErr:      true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "check update route success",
			args:         args{ctx, ns1, r1},
			fields:       fields{stg},
			handler:      route.RouteUpdateH,
			data:         string(mf2),
			want:         v1.View().Route().New(r3),
			wantErr:      false,
			expectedCode: http.StatusOK,
		},
	}

	clear := func() {
		err := stg.Del(context.Background(), stg.Collection().Namespace(), types.EmptyString)
		assert.NoError(t, err)

		err = stg.Del(context.Background(), stg.Collection().Service(), types.EmptyString)
		assert.NoError(t, err)

		err = stg.Del(context.Background(), stg.Collection().Route(), types.EmptyString)
		assert.NoError(t, err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			clear()
			defer clear()

			err := tc.fields.stg.Put(context.Background(), stg.Collection().Namespace(), tc.fields.stg.Key().Namespace(ns1.Meta.Name), ns1, nil)
			assert.NoError(t, err)

			err = tc.fields.stg.Put(context.Background(), stg.Collection().Service(), stg.Key().Service(sv1.Meta.Namespace, sv1.Meta.Name), sv1, nil)
			assert.NoError(t, err)

			err = tc.fields.stg.Put(context.Background(), stg.Collection().Service(), stg.Key().Service(sv2.Meta.Namespace, sv2.Meta.Name), sv2, nil)
			assert.NoError(t, err)

			err = stg.Put(context.Background(), stg.Collection().Route(), stg.Key().Route(r1.Meta.Namespace, r1.Meta.Name), r1, nil)
			assert.NoError(t, err)

			// Create assert request to pass to our handler. We don't have any query parameters for now, so we'll
			// pass 'nil' as the third parameter.
			req, err := http.NewRequest("PUT", fmt.Sprintf("/namespace/%s/route/%s", tc.args.namespace.Meta.Name, tc.args.route.Meta.Name), strings.NewReader(tc.data))
			assert.NoError(t, err)

			if tc.headers != nil {
				for key, val := range tc.headers {
					req.Header.Set(key, val)
				}
			}

			r := mux.NewRouter()
			r.HandleFunc("/namespace/{namespace}/route/{route}", tc.handler)

			setRequestVars(r, req)

			// We create assert ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			res := httptest.NewRecorder()

			// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
			// directly and pass in our Request and ResponseRecorder.
			r.ServeHTTP(res, req)

			// Check the status code is what we expect.
			body, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err)

			if !assert.Equal(t, tc.expectedCode, res.Code, "status code not equal") {
				t.Error(string(body))
				return
			}

			if tc.wantErr {
				assert.Equal(t, tc.err, string(body), "incorrect code message")
			} else {
				got := new(types.Route)
				err := tc.fields.stg.Get(tc.args.ctx, stg.Collection().Route(), stg.Key().Route(tc.args.namespace.Meta.Name, tc.want.Meta.Name), got, nil)
				assert.NoError(t, err)
				if assert.NotEmpty(t, got, "route is empty") {
					assert.Equal(t, tc.want.Meta.Name, got.Meta.Name, "names mismatch")
					assert.Equal(t, len(tc.want.Spec.Rules), len(got.Spec.Rules), "rules count mismatch")
					assert.Equal(t, tc.want.Spec.Rules[0].Endpoint, got.Spec.Rules[0].Endpoint, "endpoints mismatch")
				}
			}
		})
	}

}

// Testing RouteRemoveH handler
func TestRouteRemove(t *testing.T) {

	var ctx = context.Background()

	stg, _ := storage.Get("mock")
	envs.Get().SetStorage(stg)

	ns1 := getNamespaceAsset("demo", "")
	ns2 := getNamespaceAsset("test", "")
	r1 := getRouteAsset(ns1.Meta.Name, "demo")
	r2 := getRouteAsset(ns1.Meta.Name, "test")

	type fields struct {
		stg storage.Storage
	}

	type args struct {
		ctx       context.Context
		namespace *types.Namespace
		route     *types.Route
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		headers      map[string]string
		handler      func(http.ResponseWriter, *http.Request)
		err          string
		want         string
		wantErr      bool
		expectedCode int
	}{
		{
			name:         "checking get route if not exists",
			args:         args{ctx, ns1, r2},
			fields:       fields{stg},
			handler:      route.RouteRemoveH,
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Route not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking get route if namespace not exists",
			args:         args{ctx, ns2, r1},
			fields:       fields{stg},
			handler:      route.RouteRemoveH,
			err:          "{\"code\":404,\"status\":\"Not Found\",\"message\":\"Namespace not found\"}",
			wantErr:      true,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "checking del route successfully",
			args:         args{ctx, ns1, r1},
			fields:       fields{stg},
			handler:      route.RouteRemoveH,
			want:         "",
			wantErr:      false,
			expectedCode: http.StatusOK,
		},
	}

	clear := func() {
		err := envs.Get().GetStorage().Del(context.Background(), stg.Collection().Namespace(), types.EmptyString)
		assert.NoError(t, err)

		err = stg.Del(context.Background(), stg.Collection().Route(), types.EmptyString)
		assert.NoError(t, err)
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			clear()
			defer clear()

			err := tc.fields.stg.Put(context.Background(), stg.Collection().Namespace(), tc.fields.stg.Key().Namespace(ns1.Meta.Name), ns1, nil)
			assert.NoError(t, err)

			err = stg.Put(context.Background(), stg.Collection().Route(), stg.Key().Route(r1.Meta.Namespace, r1.Meta.Name), r1, nil)
			assert.NoError(t, err)

			// Create assert request to pass to our handler. We don't have any query parameters for now, so we'll
			// pass 'nil' as the third parameter.
			req, err := http.NewRequest("DELETE", fmt.Sprintf("/namespace/%s/route/%s", tc.args.namespace.Meta.Name, tc.args.route.Meta.Name), nil)
			assert.NoError(t, err)

			if tc.headers != nil {
				for key, val := range tc.headers {
					req.Header.Set(key, val)

				}
			}

			r := mux.NewRouter()
			r.HandleFunc("/namespace/{namespace}/route/{route}", tc.handler)

			setRequestVars(r, req)

			// We create assert ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			res := httptest.NewRecorder()

			// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
			// directly and pass in our Request and ResponseRecorder.
			r.ServeHTTP(res, req)

			// Check the status code is what we expect.
			if !assert.Equal(t, tc.expectedCode, res.Code, "status code not equal") {
				return
			}

			body, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err)

			if tc.wantErr {
				assert.Equal(t, tc.err, string(body), "incorrect status code")
			} else {
				var got = new(types.Route)
				err := tc.fields.stg.Get(tc.args.ctx, stg.Collection().Route(), stg.Key().Route(tc.args.namespace.Meta.Name, tc.args.route.Meta.Name), got, nil)
				if err != nil {
					assert.NoError(t, err)
				}

				if got != nil {
					assert.Equal(t, got.Status.State, types.StateDestroy, "can not be set to destroy")
				}

				assert.Equal(t, tc.want, string(body), "response not empty")
			}
		})
	}

}

func getNamespaceAsset(name, desc string) *types.Namespace {
	var n = types.Namespace{}
	n.Meta.SetDefault()
	n.Meta.Name = name
	n.Meta.Description = desc
	n.Meta.Endpoint = fmt.Sprintf("%s", name)
	return &n
}

func getServiceAsset(namespace, name, desc string) *types.Service {
	var s = types.Service{}
	s.Meta.SetDefault()
	s.Meta.Namespace = namespace
	s.Meta.Name = name
	s.Meta.Description = desc
	s.Meta.Endpoint = fmt.Sprintf("%s.%s", namespace, name)
	return &s
}

func getRouteAsset(namespace, name string) *types.Route {
	var r = types.Route{}
	r.Meta.SetDefault()
	r.Meta.Namespace = namespace
	r.Meta.Name = name
	r.Spec.Domain = fmt.Sprintf("%s.test-domain.com", name)
	r.Spec.Rules = make([]types.RouteRule, 0)
	return &r
}

func getRouteManifest(name string) *request.RouteManifest {
	var mf = new(request.RouteManifest)

	mf.Meta.Name = &name
	mf.Spec.Port = 80
	mf.Spec.Rules = make([]request.RouteManifestSpecRulesOption, 0)
	mf.Spec.Rules = append(mf.Spec.Rules, request.RouteManifestSpecRulesOption{
		Port:    80,
		Path:    "/",
		Service: name,
	})

	return mf
}

func setRequestVars(r *mux.Router, req *http.Request) {
	var match mux.RouteMatch
	// Take the request and match it
	r.Match(req, &match)
	// Push the variable onto the context
	req = mux.SetURLVars(req, match.Vars)
}
