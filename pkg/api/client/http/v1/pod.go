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

package v1

import (
	"context"
	"fmt"
	"io"
	"strconv"

	rv1 "github.com/lastbackend/lastbackend/pkg/api/types/v1/request"
	vv1 "github.com/lastbackend/lastbackend/pkg/api/types/v1/views"
	"github.com/lastbackend/lastbackend/pkg/distribution/errors"
	"github.com/lastbackend/lastbackend/pkg/util/http/request"
)

type PodClient struct {
	client *request.RESTClient

	parent struct {
		kind     string
		selflink string
	}

	namespace  string
	service    string
	deployment string

	name string
}

func (pc *PodClient) List(ctx context.Context) (*vv1.PodList, error) {

	var s *vv1.PodList
	var e *errors.Http

	err := pc.client.Get(fmt.Sprintf("/namespace/%s/service/%s/deploymet/%s/pod", pc.namespace, pc.service, pc.deployment)).
		AddHeader("Content-Type", "application/json").
		JSON(&s, &e)

	if err != nil {
		return nil, err
	}
	if e != nil {
		return nil, errors.New(e.Message)
	}

	if s == nil {
		list := make(vv1.PodList, 0)
		s = &list
	}

	return s, nil
}

func (pc *PodClient) Get(ctx context.Context) (*vv1.Pod, error) {

	var s *vv1.Pod
	var e *errors.Http

	err := pc.client.Get(fmt.Sprintf("/namespace/%s/service/%s/deploymet/%s/pod/%s", pc.namespace, pc.service, pc.deployment, pc.name)).
		AddHeader("Content-Type", "application/json").
		JSON(&s, &e)

	if err != nil {
		return nil, err
	}
	if e != nil {
		return nil, errors.New(e.Message)
	}

	return s, nil
}

func (pc *PodClient) Logs(ctx context.Context, opts *rv1.PodLogsOptions) (io.ReadCloser, error) {

	res := pc.client.Get(fmt.Sprintf("/namespace/%s/service/%s/logs", pc.namespace, pc.service))

	if opts != nil {
		res.Param("deployment", pc.deployment)
		res.Param("pod", pc.name)
		res.Param("container", opts.Container)

		if opts.Follow {
			res.Param("follow", strconv.FormatBool(opts.Follow))
		}
	}

	return res.Stream()
}

func newPodClient(client *request.RESTClient, namespace, name string) *PodClient {
	return &PodClient{client: client, namespace: namespace, name: name}
}
