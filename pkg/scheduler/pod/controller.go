//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2017] Last.Backend LLC
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

package pod

import (
	"github.com/lastbackend/lastbackend/pkg/common/types"
	"github.com/lastbackend/lastbackend/pkg/scheduler/context"
)

type PodController struct {
	context *context.Context
	pods chan *types.Pod
	active bool
}

func (pc *PodController) Watch () {
	var (
		log = pc.context.GetLogger()
		stg = pc.context.GetStorage()
	)

	log.Debug("Scheduler:PodController: start watch")
	go func(){
		for {
			select {
			case p := <- pc.pods : {
				if !pc.active {
					log.Debug("Scheduler:PodController: skip management cause it is in slave mode")
					continue
				}

				log.Debugf("Pod needs to be allocated to node: %s", p.Meta.Name )
				Provision(p)
			}
			}
		}
	}()

	stg.Pod().Watch(pc.context.Background(), pc.pods)
}

func (pc *PodController) Pause () {
	pc.active = false
}

func (pc *PodController) Resume () {

	var (
		log = pc.context.GetLogger()
		stg = pc.context.GetStorage()
	)

	pc.active = true

	log.Debug("Scheduler:PodController: start check pods state")
	nss, err := stg.Namespace().List(pc.context.Background())
	if err != nil {
		log.Errorf("Scheduler:PodController: Get namespaces list err: %s", err.Error())
	}

	for _, ns := range nss {
		log.Debugf("Get pods in namespace: %s", ns.Meta.Name)
		pods, err := stg.Pod().ListByNamespace(pc.context.Background(), ns.Meta.Name)
		if err != nil {
			log.Errorf("Scheduler:PodController: Get pods list err: %s", err.Error())
		}

		for _, p := range pods {
			if p.State.Provision == true {
				pc.pods <- p
			}
		}
	}
}

func NewPodController (ctx *context.Context) *PodController {
	sc := new(PodController)
	sc.context = ctx
	sc.active = false
	sc.pods = make (chan *types.Pod)

	return sc
}