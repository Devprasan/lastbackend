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

package cluster

import (
	"context"

	"github.com/lastbackend/lastbackend/pkg/controller/envs"
	"github.com/lastbackend/lastbackend/pkg/controller/ipam/ipam"
	"github.com/lastbackend/lastbackend/pkg/distribution"
	"github.com/lastbackend/lastbackend/pkg/distribution/types"
	"github.com/lastbackend/lastbackend/pkg/log"
)

const (
	logLevel  = 3
	logPrefix = "observer:cluster"
)

// ClusterState is cluster current state struct
type ClusterState struct {
	cluster *types.Cluster
	ingress struct {
		list map[string]*types.Ingress
	}
	volume struct {
		observer chan *types.Volume
		list     map[string]*types.Volume
	}
	node struct {
		observer chan *types.Node
		lease    chan *NodeLease
		release  chan *NodeLease
		list     map[string]*types.Node
	}
}

// Runtime cluster describes main cluster state loop
func (cs *ClusterState) Observe() {
	// Watch node changes
	for {
		select {
		case l := <-cs.node.lease:
			handleNodeLease(cs, l)
			break
		case l := <-cs.node.release:
			handleNodeRelease(cs, l)
			break
		case n := <-cs.node.observer:
			log.V(7).Debugf("node: %s", n.Meta.Name)
			cs.node.list[n.SelfLink()] = n
			break
		case v := <-cs.volume.observer:
			log.V(7).Debugf("volume: %s", v.SelfLink())
			if err := volumeObserve(cs, v); err != nil {
				log.Errorf("%s", err.Error())
			}
			break
		}
	}
}

// Loop cluster state from storage
func (cs *ClusterState) Loop() error {

	log.V(logLevel).Debug("restore cluster state")
	var err error

	// Get cluster info
	cm := distribution.NewClusterModel(context.Background(), envs.Get().GetStorage())
	cs.cluster, err = cm.Get()
	if err != nil {
		return err
	}

	// Get all nodes in cluster
	nm := distribution.NewNodeModel(context.Background(), envs.Get().GetStorage())
	nl, err := nm.List()
	if err != nil {
		return err
	}

	for _, n := range nl.Items {
		// Add node to local cache
		cs.SetNode(n)
		// Run node observers
	}

	go cs.subscribe(context.Background(), &nl.System.Revision)
	return nil
}

func (cs *ClusterState) subscribe(ctx context.Context, rev *int64) {

	var (
		p = make(chan types.NodeEvent)
	)

	nm := distribution.NewNodeModel(ctx, envs.Get().GetStorage())

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case w := <-p:

				if w.Data == nil {
					continue
				}

				if w.IsActionRemove() {
					cs.DelNode(w.Data)
					continue
				}

				cs.SetNode(w.Data)
			}
		}
	}()

	nm.Watch(p, rev)
}

// lease new node for requests by parameters
func (cs *ClusterState) lease(opts NodeLeaseOptions) (*types.Node, error) {

	// Work as node lease requests queue
	req := new(NodeLease)
	req.Request = opts
	req.done = make(chan bool)
	cs.node.lease <- req
	req.Wait()
	return req.Response.Node, req.Response.Err
}

// release node
func (cs *ClusterState) release(opts NodeLeaseOptions) (*types.Node, error) {
	// Work as node release
	req := new(NodeLease)
	req.Request = opts
	req.done = make(chan bool)
	cs.node.release <- req
	req.Wait()
	return req.Response.Node, req.Response.Err
}

func (cs *ClusterState) leaseSync(opts NodeLeaseOptions) (*types.Node, error) {

	// Work as node lease requests queue
	req := new(NodeLease)
	req.Request = opts
	req.sync = true
	if err := handleNodeLease(cs, req); err != nil {
		log.Errorf("sync lease error: %s", err.Error())
		return nil, err
	}

	return req.Response.Node, req.Response.Err
}

// release node
func (cs *ClusterState) releaseSync(opts NodeLeaseOptions) (*types.Node, error) {
	// Work as node release
	req := new(NodeLease)
	req.Request = opts
	req.sync = true

	if err := handleNodeRelease(cs, req); err != nil {
		log.Errorf("sync release error: %s", err.Error())
		return nil, err
	}

	return req.Response.Node, req.Response.Err
}

// IPAM management
func (cs *ClusterState) IPAM() ipam.IPAM {
	return envs.Get().GetIPAM()
}

func (cs *ClusterState) SetNode(n *types.Node) {
	cs.node.observer <- n
}

func (cs *ClusterState) DelNode(n *types.Node) {
	delete(cs.node.list, n.Meta.SelfLink)
}

func (cs *ClusterState) SetVolume(v *types.Volume) {
	cs.volume.observer <- v
}

func (cs *ClusterState) DelVolume(v *types.Volume) {
	delete(cs.volume.list, v.Meta.SelfLink)
}

func (cs *ClusterState) PodLease(p *types.Pod) (*types.Node, error) {

	var RAM int64

	for _, s := range p.Spec.Template.Containers {
		RAM += s.Resources.Request.RAM
	}

	opts := NodeLeaseOptions{
		Selector: p.Spec.Selector,
		Memory:   &RAM,
	}

	node, err := cs.lease(opts)
	if err != nil {
		log.Errorf("%s:> pod lease err: %s", logPrefix, err)
		return nil, err
	}

	return node, err
}

func (cs *ClusterState) PodRelease(p *types.Pod) (*types.Node, error) {
	var RAM int64

	for _, s := range p.Spec.Template.Containers {
		RAM += s.Resources.Request.RAM
	}

	opts := NodeLeaseOptions{
		Node:   &p.Meta.Node,
		Memory: &RAM,
	}

	node, err := cs.release(opts)
	if err != nil {
		log.Errorf("%s:> pod lease err: %s", logPrefix, err)
		return nil, err
	}

	return node, err
}

func (cs *ClusterState) VolumeLease(v *types.Volume) (*types.Node, error) {

	opts := NodeLeaseOptions{
		Node:     &v.Spec.Selector.Node,
		Selector: v.Spec.Selector,
		Storage:  &v.Spec.Capacity.Storage,
	}

	node, err := cs.leaseSync(opts)
	if err != nil {
		log.Errorf("%s:> volume lease err: %s", logPrefix, err)
		return nil, err
	}

	return node, err
}

func (cs *ClusterState) VolumeRelease(v *types.Volume) (*types.Node, error) {

	opts := NodeLeaseOptions{
		Node:    &v.Meta.Node,
		Storage: &v.Spec.Capacity.Storage,
	}

	node, err := cs.releaseSync(opts)
	if err != nil {
		log.Errorf("%s:> volume lease err: %s", logPrefix, err)
		return nil, err
	}

	return node, err
}

// NewClusterState returns new cluster state instance
func NewClusterState() *ClusterState {

	var cs = new(ClusterState)

	cs.ingress.list = make(map[string]*types.Ingress)

	cs.volume.list = make(map[string]*types.Volume)
	cs.volume.observer = make(chan *types.Volume)

	cs.node.observer = make(chan *types.Node)
	cs.node.list = make(map[string]*types.Node)

	cs.node.lease = make(chan *NodeLease)
	cs.node.release = make(chan *NodeLease)

	cs.node.observer = make(chan *types.Node)
	go cs.Observe()

	return cs
}
