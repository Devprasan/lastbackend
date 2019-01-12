//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2019] Last.Backend LLC
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

package types

import "fmt"

type DeploymentMap struct {
	Runtime
	Items map[string]*Deployment
}

type DeploymentList struct {
	Runtime
	Items []*Deployment
}

type Deployment struct {
	Runtime
	// Deployment Meta
	Meta DeploymentMeta `json:"meta"`
	// Deployment status
	Status DeploymentStatus `json:"status"`
	// Deployment spec
	Spec DeploymentSpec `json:"spec"`
}

type DeploymentMeta struct {
	Meta
	// Version
	Version int `json:"version"`
	// Namespace id
	Namespace string `json:"namespace"`
	// Service id
	Service string `json:"service"`
	// Upstream
	Endpoint string `json:"endpoint"`
	// Self Link
	Status string `json:"status"`
}

type DeploymentSpec struct {
	Replicas int          `json:"replicas"`
	State    SpecState    `json:"state"`
	Selector SpecSelector `json:"selector"`
	Template SpecTemplate `json:"template"`
}

type DeploymentStatus struct {
	State        string                       `json:"state"`
	Message      string                       `json:"message"`
	Dependencies DeploymentStatusDependencies `json:"dependencies"`
}

type DeploymentStatusDependencies struct {
	Volumes map[string]DeploymentStatusDependency `json:"volumes"`
	Secrets map[string]DeploymentStatusDependency `json:"secrets"`
	Configs map[string]DeploymentStatusDependency `json:"configs"`
}

type DeploymentStatusDependency struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DeploymentReplicas struct {
	Total     int `json:"total"`
	Provision int `json:"provision"`
	Pulling   int `json:"pulling"`
	Created   int `json:"created"`
	Started   int `json:"started"`
	Stopped   int `json:"stopped"`
	Errored   int `json:"errored"`
}

type DeploymentOptions struct {
	Replicas int `json:"replicas"`
}

func (d *Deployment) SelfLink() string {
	if d.Meta.SelfLink == "" {
		d.Meta.SelfLink = d.CreateSelfLink(d.Meta.Namespace, d.Meta.Service, d.Meta.Name)
	}
	return d.Meta.SelfLink
}

func (d *Deployment) CreateSelfLink(namespace, service, name string) string {
	return fmt.Sprintf("%s:%s:%s", namespace, service, name)
}

func (d *Deployment) ServiceLink() string {
	return fmt.Sprintf("%s:%s", d.Meta.Namespace, d.Meta.Service)
}

func (ds *DeploymentStatus) CheckDeps() bool {

	for _, d := range ds.Dependencies.Volumes {
		if d.Status != StateReady {
			return false
		}
	}

	for _, d := range ds.Dependencies.Secrets {
		if d.Status != StateReady {
			return false
		}
	}

	for _, d := range ds.Dependencies.Configs {
		if d.Status != StateReady {
			return false
		}
	}

	return true
}

func (d *DeploymentStatus) SetCreated() {
	d.State = StateCreated
	d.Message = ""
}

func (d *DeploymentStatus) SetProvision() {
	d.State = StateProvision
	d.Message = ""
}

func (d *DeploymentStatus) SetReady() {
	d.State = StateReady
	d.Message = ""
}

func (d *DeploymentStatus) SetCancel() {
	d.State = StateCancel
	d.Message = ""
}

func (d *DeploymentStatus) SetDestroy() {
	d.State = StateDestroy
	d.Message = ""
}

type DeploymentUpdateOptions struct {
	// Number of replicas
	Replicas *int
	// Deployment status for update
	Status *struct {
		State   string
		Message string
	}
}

func NewDeploymentList() *DeploymentList {
	dm := new(DeploymentList)
	dm.Items = make([]*Deployment, 0)
	return dm
}

func NewDeploymentMap() *DeploymentMap {
	dm := new(DeploymentMap)
	dm.Items = make(map[string]*Deployment)
	return dm
}
