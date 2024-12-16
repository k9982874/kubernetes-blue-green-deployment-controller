package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	v1 "k8s.kexinlife.com/controller/bgd/pkg/apis/controller/v1alpha1"
	clientset "k8s.kexinlife.com/controller/bgd/pkg/generated/clientset/versioned/typed/controller/v1alpha1"
	bgdutil "k8s.kexinlife.com/controller/bgd/util"
)

func newReplicaSet(bgd *v1.BlueGreenDeployment, color string) *appsv1.ReplicaSet {
	replicas := bgd.Spec.Replicas
	if bgd.Status.ActiveReplicaSetColor != color {
		replicas = int32(0)
	}

	rsName := BlueRSName
	if color != BlueColor {
		rsName = GreenRSName
	}

	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rsName,
			Namespace: bgd.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(bgd, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    BlueGreenDeploymentKind,
				}),
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"color": color},
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"color": color},
				},
				Spec: bgd.Spec.PodSpec,
			},
		},
	}
}

func newService(bgd *v1.BlueGreenDeployment) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: bgd.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{LabelColorKey: BlueColor},
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(443),
				},
			},
		},
	}
}

func createReplicaSetsIfNotFound(ctx context.Context, bgd *v1.BlueGreenDeployment, clientset kubernetes.Interface) (*appsv1.ReplicaSet, *appsv1.ReplicaSet, error) {
	blueRS, err := createReplicaSetIfNotFound(ctx, bgd, clientset, BlueRSName, BlueColor)
	if err != nil {
		return nil, nil, fmt.Errorf("when creating blue ReplicaSet, %v", err)
	}

	greenRS, err := createReplicaSetIfNotFound(ctx, bgd, clientset, GreenRSName, GreenColor)
	if err != nil {
		return nil, nil, fmt.Errorf("when creating green ReplicaSet, %v", err)
	}

	return blueRS, greenRS, nil
}

func createReplicaSetIfNotFound(ctx context.Context, bgd *v1.BlueGreenDeployment, clientset kubernetes.Interface, rsName, rsColor string) (*appsv1.ReplicaSet, error) {
	ns := bgd.Namespace
	rs, err := clientset.AppsV1().ReplicaSets(ns).Get(ctx, rsName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			rs, err = clientset.AppsV1().ReplicaSets(ns).Create(ctx, newReplicaSet(bgd, rsColor), metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to create ReplicaSet %q: %v", rsName, err)
			}
			if err = waitAllReplicaSetPodsAvailable(ctx, clientset, rs); err != nil {
				log.Printf("some pods for ReplicaSet %q failed to become available: %v", rs.Name, err)
			}
			return rs, nil
		} else {
			return nil, fmt.Errorf("failed to get ReplicaSet %q: %v", rsName, err)
		}
	}

	return rs, nil
}

func updateBlueGreenDeployment(ctx context.Context, c clientset.ControllerV1alpha1Interface, bgdName, ns string, updateFunc func(*v1.BlueGreenDeployment)) (*v1.BlueGreenDeployment, error) {
	bgdClient := c.BlueGreenDeployments(ns)

	var bgd *v1.BlueGreenDeployment
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var err error
		bgd, err = bgdClient.Get(ctx, bgdName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get BlueGreenDeployment object: %v", err)
		}
		bgd = bgd.DeepCopy()
		updateFunc(bgd)
		bgd, err = bgdClient.UpdateStatus(ctx, bgd, metav1.UpdateOptions{FieldManager: FieldManager})
		if err != nil {
			return fmt.Errorf("failed to update BlueGreenDeployment object: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bgd, nil
}

func waitAllReplicaSetPodsAvailable(ctx context.Context, clientset kubernetes.Interface, replicaSet *appsv1.ReplicaSet) error {
	desiredGeneration := replicaSet.Generation
	return wait.PollUntilContextTimeout(ctx, PollInterval, PollTimeout, true, func(ctx context.Context) (bool, error) {
		rs, err := clientset.AppsV1().ReplicaSets(replicaSet.Namespace).Get(ctx, replicaSet.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get ReplicaSet %q: %v", replicaSet.Name, err)
		}
		return rs.Status.ObservedGeneration >= desiredGeneration && rs.Status.AvailableReplicas == *replicaSet.Spec.Replicas, nil
	})
}

func createServiceIfNotFound(ctx context.Context, bgd *v1.BlueGreenDeployment, clientset kubernetes.Interface) (*corev1.Service, error) {
	ns := bgd.Namespace
	svc, err := clientset.CoreV1().Services(ns).Get(ctx, ServiceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			svc, err = clientset.CoreV1().Services(ns).Create(ctx, newService(bgd), metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to create service %q: %v", ServiceName, err)
			}
		} else {
			return nil, fmt.Errorf("failed to get service %q: %v", ServiceName, err)
		}
	}

	return svc, nil
}

func scaleReplicaSet(ctx context.Context, clientset kubernetes.Interface, replicas int32, replicaSet *appsv1.ReplicaSet) error {
	replicaSet, err := updateReplicaSet(
		ctx,
		clientset,
		replicaSet.Name,
		replicaSet.Namespace,
		func(rs *appsv1.ReplicaSet) {
			*rs.Spec.Replicas = replicas
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update its .spec.replicas: %v", err)
	}

	err = waitAllReplicaSetPodsAvailable(ctx, clientset, replicaSet)
	if err != nil {
		log.Printf("some pods for ReplicaSet %q failed to become available: %v", replicaSet.Name, err)
	}

	return nil
}

func updateReplicaSet(ctx context.Context, clientset kubernetes.Interface, rsName, ns string, updateFunc func(*appsv1.ReplicaSet)) (*appsv1.ReplicaSet, error) {
	var replicaSet *appsv1.ReplicaSet
	rsClient := clientset.AppsV1().ReplicaSets(ns)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		rs, err := rsClient.Get(ctx, rsName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get ReplicaSet %q: %v", rsName, err)
		}
		updateFunc(rs)
		replicaSet, err = rsClient.Update(ctx, rs, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update ReplicaSet %q: %v", rsName, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return replicaSet, nil
}

func replaceInactiveReplicaSet(ctx context.Context, clientset kubernetes.Interface, bgd *v1.BlueGreenDeployment, inactiveRS *appsv1.ReplicaSet) (*appsv1.ReplicaSet, error) {
	ns := bgd.Namespace
	rsName := inactiveRS.Name
	color := inactiveRS.Spec.Selector.MatchLabels[LabelColorKey]
	deletePropagationOrphanPolicy := metav1.DeletePropagationBackground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePropagationOrphanPolicy,
	}
	err := clientset.AppsV1().ReplicaSets(ns).Delete(ctx, rsName, deleteOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inactive ReplicaSet %q: %v", rsName, err)
	}

	replicaSet, err := createReplicaSetIfNotFound(ctx, bgd, clientset, rsName, color)
	if err != nil {
		return nil, fmt.Errorf("when re-creating inactive ReplicaSet %q, %v", rsName, err)
	}

	return replicaSet, nil
}

func updateService(ctx context.Context, clientset kubernetes.Interface, ns string, updateFunc func(*corev1.Service)) (*corev1.Service, error) {
	var service *corev1.Service
	svcClient := clientset.CoreV1().Services(ns)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		svc, err := svcClient.Get(ctx, ServiceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get service %q: %v", ServiceName, err)
		}
		updateFunc(svc)
		service, err = svcClient.Update(ctx, svc, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update service %q: %v", ServiceName, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return service, nil
}

func defaultFields(ctx context.Context, c clientset.ControllerV1alpha1Interface, bgd *v1.BlueGreenDeployment) (*v1.BlueGreenDeployment, error) {
	return updateBlueGreenDeployment(ctx, c, bgd.Name, bgd.Namespace, func(bgd *v1.BlueGreenDeployment) {
		container := &bgd.Spec.PodSpec.Containers[0]
		if container.TerminationMessagePath == "" {
			container.TerminationMessagePath = "/dev/termination-log"
		}
		if container.TerminationMessagePolicy == "" {
			container.TerminationMessagePolicy = "File"
		}
		if container.ImagePullPolicy == "" {
			container.ImagePullPolicy = "IfNotPresent"
		}
		if bgd.Spec.PodSpec.RestartPolicy == "" {
			bgd.Spec.PodSpec.RestartPolicy = "Always"
		}
		if bgd.Spec.PodSpec.TerminationGracePeriodSeconds == nil {
			bgd.Spec.PodSpec.TerminationGracePeriodSeconds = bgdutil.Int64Ptr(0)
		}
		if bgd.Spec.PodSpec.DNSPolicy == "" {
			bgd.Spec.PodSpec.DNSPolicy = "ClusterFirst"
		}
		if bgd.Spec.PodSpec.SecurityContext == nil {
			bgd.Spec.PodSpec.SecurityContext = &corev1.PodSecurityContext{}
		}
		if bgd.Spec.PodSpec.SchedulerName == "" {
			bgd.Spec.PodSpec.SchedulerName = "default-scheduler"
		}
	})
}
