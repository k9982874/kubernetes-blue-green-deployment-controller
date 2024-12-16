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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
	controllerv1alpha1 "k8s.kexinlife.com/controller/bgd/pkg/apis/controller/v1alpha1"
	scheme "k8s.kexinlife.com/controller/bgd/pkg/generated/clientset/versioned/scheme"
)

// BlueGreenDeploymentsGetter has a method to return a BlueGreenDeploymentInterface.
// A group's client should implement this interface.
type BlueGreenDeploymentsGetter interface {
	BlueGreenDeployments(namespace string) BlueGreenDeploymentInterface
}

// BlueGreenDeploymentInterface has methods to work with BlueGreenDeployment resources.
type BlueGreenDeploymentInterface interface {
	Create(ctx context.Context, blueGreenDeployment *controllerv1alpha1.BlueGreenDeployment, opts v1.CreateOptions) (*controllerv1alpha1.BlueGreenDeployment, error)
	Update(ctx context.Context, blueGreenDeployment *controllerv1alpha1.BlueGreenDeployment, opts v1.UpdateOptions) (*controllerv1alpha1.BlueGreenDeployment, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, blueGreenDeployment *controllerv1alpha1.BlueGreenDeployment, opts v1.UpdateOptions) (*controllerv1alpha1.BlueGreenDeployment, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*controllerv1alpha1.BlueGreenDeployment, error)
	List(ctx context.Context, opts v1.ListOptions) (*controllerv1alpha1.BlueGreenDeploymentList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *controllerv1alpha1.BlueGreenDeployment, err error)
	BlueGreenDeploymentExpansion
}

// blueGreenDeployments implements BlueGreenDeploymentInterface
type blueGreenDeployments struct {
	*gentype.ClientWithList[*controllerv1alpha1.BlueGreenDeployment, *controllerv1alpha1.BlueGreenDeploymentList]
}

// newBlueGreenDeployments returns a BlueGreenDeployments
func newBlueGreenDeployments(c *ControllerV1alpha1Client, namespace string) *blueGreenDeployments {
	return &blueGreenDeployments{
		gentype.NewClientWithList[*controllerv1alpha1.BlueGreenDeployment, *controllerv1alpha1.BlueGreenDeploymentList](
			"bluegreendeployments",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *controllerv1alpha1.BlueGreenDeployment { return &controllerv1alpha1.BlueGreenDeployment{} },
			func() *controllerv1alpha1.BlueGreenDeploymentList {
				return &controllerv1alpha1.BlueGreenDeploymentList{}
			},
		),
	}
}