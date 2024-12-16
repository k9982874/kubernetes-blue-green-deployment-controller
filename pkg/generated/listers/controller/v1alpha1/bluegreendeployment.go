/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
	controllerv1alpha1 "k8s.kexinlife.com/controller/bgd/pkg/apis/controller/v1alpha1"
)

// BlueGreenDeploymentLister helps list BlueGreenDeployments.
// All objects returned here must be treated as read-only.
type BlueGreenDeploymentLister interface {
	// List lists all BlueGreenDeployments in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*controllerv1alpha1.BlueGreenDeployment, err error)
	// BlueGreenDeployments returns an object that can list and get BlueGreenDeployments.
	BlueGreenDeployments(namespace string) BlueGreenDeploymentNamespaceLister
	BlueGreenDeploymentListerExpansion
}

// blueGreenDeploymentLister implements the BlueGreenDeploymentLister interface.
type blueGreenDeploymentLister struct {
	listers.ResourceIndexer[*controllerv1alpha1.BlueGreenDeployment]
}

// NewBlueGreenDeploymentLister returns a new BlueGreenDeploymentLister.
func NewBlueGreenDeploymentLister(indexer cache.Indexer) BlueGreenDeploymentLister {
	return &blueGreenDeploymentLister{listers.New[*controllerv1alpha1.BlueGreenDeployment](indexer, controllerv1alpha1.Resource("bluegreendeployment"))}
}

// BlueGreenDeployments returns an object that can list and get BlueGreenDeployments.
func (s *blueGreenDeploymentLister) BlueGreenDeployments(namespace string) BlueGreenDeploymentNamespaceLister {
	return blueGreenDeploymentNamespaceLister{listers.NewNamespaced[*controllerv1alpha1.BlueGreenDeployment](s.ResourceIndexer, namespace)}
}

// BlueGreenDeploymentNamespaceLister helps list and get BlueGreenDeployments.
// All objects returned here must be treated as read-only.
type BlueGreenDeploymentNamespaceLister interface {
	// List lists all BlueGreenDeployments in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*controllerv1alpha1.BlueGreenDeployment, err error)
	// Get retrieves the BlueGreenDeployment from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*controllerv1alpha1.BlueGreenDeployment, error)
	BlueGreenDeploymentNamespaceListerExpansion
}

// blueGreenDeploymentNamespaceLister implements the BlueGreenDeploymentNamespaceLister
// interface.
type blueGreenDeploymentNamespaceLister struct {
	listers.ResourceIndexer[*controllerv1alpha1.BlueGreenDeployment]
}