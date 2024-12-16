/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"golang.org/x/time/rate"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	bgdv1alpha1 "k8s.kexinlife.com/controller/bgd/pkg/apis/controller/v1alpha1"
	clientset "k8s.kexinlife.com/controller/bgd/pkg/generated/clientset/versioned"
	bgdscheme "k8s.kexinlife.com/controller/bgd/pkg/generated/clientset/versioned/scheme"
	informers "k8s.kexinlife.com/controller/bgd/pkg/generated/informers/externalversions/controller/v1alpha1"
	listers "k8s.kexinlife.com/controller/bgd/pkg/generated/listers/controller/v1alpha1"
	"k8s.kexinlife.com/controller/bgd/util"
)

const controllerAgentName = "blue-green-deployment-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a BlueGreenDeployment is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a BlueGreenDeployment fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by BlueGreenDeployment"
	// MessageResourceSynced is the message used for an Event fired when a BlueGreenDeployment
	// is synced successfully
	MessageResourceSynced = "BlueGreenDeployment synced successfully"
	// FieldManager distinguishes this controller from other things writing to API objects
	FieldManager = controllerAgentName
)

const (
	BlueColor               = "blue"
	GreenColor              = "green"
	BlueGreenDeploymentKind = "BlueGreenDeployment"
	BlueRSName              = BlueColor + "-rs"
	GreenRSName             = GreenColor + "-rs"
	ServiceName             = "bgd-svc"

	LabelColorKey = "color"

	PollInterval = 100 * time.Millisecond
	PollTimeout  = 30 * time.Second
)

// Controller is the controller implementation for BlueGreenDeployment resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset clientset.Interface

	deploymentsLister          appslisters.DeploymentLister
	deploymentsSynced          cache.InformerSynced
	blueGreenDeploymentLister  listers.BlueGreenDeploymentLister
	blueGreenDeploymentsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.TypedRateLimitingInterface[cache.ObjectName]
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	blueGreenDeploymentInformer informers.BlueGreenDeploymentInformer) *Controller {
	logger := klog.FromContext(ctx)

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(bgdscheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	ratelimiter := workqueue.NewTypedMaxOfRateLimiter(
		workqueue.NewTypedItemExponentialFailureRateLimiter[cache.ObjectName](5*time.Millisecond, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[cache.ObjectName]{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)

	controller := &Controller{
		kubeclientset:              kubeclientset,
		sampleclientset:            sampleclientset,
		deploymentsLister:          deploymentInformer.Lister(),
		deploymentsSynced:          deploymentInformer.Informer().HasSynced,
		blueGreenDeploymentLister:  blueGreenDeploymentInformer.Lister(),
		blueGreenDeploymentsSynced: blueGreenDeploymentInformer.Informer().HasSynced,
		workqueue:                  workqueue.NewTypedRateLimitingQueue(ratelimiter),
		recorder:                   recorder,
	}

	logger.Info("Setting up event handlers")
	// Set up an event handler for when BlueGreenDeployment resources change
	blueGreenDeploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueBlueGreenDeployment,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueBlueGreenDeployment(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a BlueGreenDeployment resource then the handler will enqueue that BlueGreenDeployment resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting BlueGreenDeployment controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentsSynced, c.blueGreenDeploymentsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers", "count", workers)
	// Launch two workers to process BlueGreenDeployment resources
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	// We call Done at the end of this func so the workqueue knows we have
	// finished processing this item. We also must remember to call Forget
	// if we do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer c.workqueue.Done(objRef)

	// Run the syncHandler, passing it the structured reference to the object to be synced.
	err := c.syncHandler(ctx, objRef)
	if err == nil {
		// If no error occurs then we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(objRef)
		logger.Info("Successfully synced", "objectName", objRef)
		return true
	}
	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring.
	utilruntime.HandleErrorWithContext(ctx, err, "Error syncing; requeuing for later retry", "objectReference", objRef)
	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.workqueue.AddRateLimited(objRef)
	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the BlueGreenDeployment resource
// with the current status of the resource.
func (c *Controller) syncHandler(ctx context.Context, objectRef cache.ObjectName) error {
	// logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", objectRef)

	// Get the BlueGreenDeployment resource with this namespace/name
	bgd, err := c.blueGreenDeploymentLister.BlueGreenDeployments(objectRef.Namespace).Get(objectRef.Name)
	if err != nil {
		// The BlueGreenDeployment resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleErrorWithContext(ctx, err, "BlueGreenDeployment referenced by item in work queue no longer exists", "objectReference", objectRef)
			return nil
		}

		return err
	}

	// Initialize .status.activeReplicaSetColor field to blue color if it is not set yet
	if bgd.Status.ActiveReplicaSetColor == "" {
		bgd, err = updateBlueGreenDeployment(
			ctx,
			c.sampleclientset.ControllerV1alpha1(),
			bgd.Name,
			bgd.Namespace,
			func(bgd *bgdv1alpha1.BlueGreenDeployment) {
				bgd.Status.ActiveReplicaSetColor = BlueColor
			},
		)
		// Update .status.activeReplicaSetColor field to blue color at the beginning
		if err != nil {
			return fmt.Errorf("failed to update .status.activeReplicaSetColor field of BlueGreenDeployment: %v", err)
		}
	}

	blueRS, greenRS, err := createReplicaSetsIfNotFound(ctx, bgd, c.kubeclientset)
	if err != nil {
		return err
	}

	service, err := createServiceIfNotFound(ctx, bgd, c.kubeclientset)
	if err != nil {
		return err
	}

	err = c.reconcileActiveReplicaSet(ctx, bgd, blueRS, greenRS, service)
	if err != nil {
		return err
	}

	c.recorder.Event(bgd, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueueBlueGreenDeployment takes a BlueGreenDeployment resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than BlueGreenDeployment.
func (c *Controller) enqueueBlueGreenDeployment(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.workqueue.Add(objectRef)
	}
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the BlueGreenDeployment resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that BlueGreenDeployment resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	logger := klog.FromContext(context.Background())
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			// If the object value is not too big and does not contain sensitive information then
			// it may be useful to include it.
			utilruntime.HandleErrorWithContext(context.Background(), nil, "Error decoding object, invalid type", "type", fmt.Sprintf("%T", obj))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			// If the object value is not too big and does not contain sensitive information then
			// it may be useful to include it.
			utilruntime.HandleErrorWithContext(context.Background(), nil, "Error decoding object tombstone, invalid type", "type", fmt.Sprintf("%T", tombstone.Obj))
			return
		}
		logger.V(4).Info("Recovered deleted object", "resourceName", object.GetName())
	}
	logger.V(4).Info("Processing object", "object", klog.KObj(object))
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a BlueGreenDeployment, we should not do anything more
		// with it.
		if ownerRef.Kind != "BlueGreenDeployment" {
			return
		}

		bgd, err := c.blueGreenDeploymentLister.BlueGreenDeployments(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			logger.V(4).Info("Ignore orphaned object", "object", klog.KObj(object), "BlueGreenDeployment", ownerRef.Name)
			return
		}

		c.enqueueBlueGreenDeployment(bgd)
		return
	}
}

// reconcileActiveReplicaSet checks active and inactive ReplicaSets' pod specs to
// see if any of the pod specs matches BlueGreenDeployment object's pod spec.
// If the active ReplicaSet has the matching spec, the controller does nothing.
// Else if the inactive ReplicaSet has the matching spec, the controller:
// 1. scales up the inactive ReplicaSet
// 2. modifies label selectors of the service to point to the newly active ReplicaSet
// 3. scales down previously active ReplicaSet to make it inactive
// Else (i.e., none of the Replicasets has the matching spec), the controller:
// 1. deletes the inactive ReplicaSet to make room for a new ReplicaSet
// 2. creates and scales up the new ReplicaSet
// 3. modifies label selectors of the service to point to the new ReplicaSet
// 4. scales down previously active ReplicaSet to make it inactive
// The function updates .status.activeReplicaSetColor field and the service's label
// selectors with color of the active ReplicaSet.
func (c *Controller) reconcileActiveReplicaSet(ctx context.Context, b *bgdv1alpha1.BlueGreenDeployment, blueRS, greenRS *appsv1.ReplicaSet, service *corev1.Service) error {
	var err error

	// If a ReplicaSet has the same color as indicated by .status.activeReplicaSetColor field
	// of BlueGreenDeployment object, it is the active ReplicaSet.
	activeColor := b.Status.ActiveReplicaSetColor
	activeRS := blueRS
	inactiveRS := greenRS
	if activeColor == GreenColor {
		activeRS = greenRS
		inactiveRS = blueRS
	}

	// Hash pod spec of active ReplicaSet, inactive ReplicaSet, and BlueGreenDeployment object for comparison
	hashedActiveRSPodSpec := util.ComputeHash(&activeRS.Spec.Template.Spec)
	hashedInactiveRSPodSpec := util.ComputeHash(&inactiveRS.Spec.Template.Spec)
	hashedBGDPodSpec := util.ComputeHash(&b.Spec.PodSpec)

	// Case: active ReplicaSet has the matching spec
	if reflect.DeepEqual(hashedActiveRSPodSpec, hashedBGDPodSpec) {
		return nil
	} else {
		// Update .status.activeReplicaSetColor field to inactive ReplicaSet's color as it will become active soon
		b, err = updateBlueGreenDeployment(
			ctx,
			c.sampleclientset.ControllerV1alpha1(),
			b.Name,
			b.Namespace,
			func(bgd *bgdv1alpha1.BlueGreenDeployment) {
				bgd.Status.ActiveReplicaSetColor = inactiveRS.Spec.Selector.MatchLabels[LabelColorKey]
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update .status.activeReplicaSetColor field of BlueGreenDeployment: %v", err)
		}

		// Case: inactive ReplicaSet has the matching spec
		if reflect.DeepEqual(hashedInactiveRSPodSpec, hashedBGDPodSpec) {
			err = scaleReplicaSet(ctx, c.kubeclientset, b.Spec.Replicas, inactiveRS)
			if err != nil {
				return fmt.Errorf("during scaling up of inactive ReplicaSet %q, %v", inactiveRS.Name, err)
			}
		} else { // Case: no Replicaset has the matching spec
			_, err = replaceInactiveReplicaSet(ctx, c.kubeclientset, b, inactiveRS)
			if err != nil {
				return fmt.Errorf("during replacement of inactive ReplicaSet %q, %v", inactiveRS.Name, err)
			}
		}

		// Scale down previously active ReplicaSet
		err = scaleReplicaSet(ctx, c.kubeclientset, 0, activeRS)
		if err != nil {
			return fmt.Errorf("during scaling down of active ReplicaSet %q, %v", activeRS.Name, err)
		}

		// Point the service to current active ReplicaSet by updating its "color" label selector to match active ReplicaSet's color
		service, err = updateService(
			ctx,
			c.kubeclientset,
			service.Namespace,
			func(service *corev1.Service) {
				service.Spec.Selector[LabelColorKey] = b.Status.ActiveReplicaSetColor
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update %q label selector of service %q to match active ReplicaSet's color: %v", LabelColorKey, service.Name, err)
		}
	}

	return nil
}
