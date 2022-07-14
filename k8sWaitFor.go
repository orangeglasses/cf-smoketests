package main

import (
	"context"
	"errors"
	"time"

	"github.com/golang/glog"
	"github.com/jpillora/backoff"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ErrNotImplemented is returned when the resource type provided is not being dealt with (yet)
var ErrNotImplemented = errors.New("not yet implemented")

// ErrUnknownResourceType is returned when the resource type provided is not being dealt with (yet)
var ErrUnknownResourceType = errors.New("unknown resource type")

// ErrWrongTypeForArgument is returned when a optinal argument was given, but its value had the wrong type (e.g. int instead of int32)
var ErrWrongTypeForArgument = errors.New("wrong type for optional argument")

// ErrUnknownPodStatus is returned when a pod status is not being handled yet
var ErrUnknownPodStatus = errors.New("unknown pod status")

// PodStatus describes a Pod's status
type PodStatus int

// Enums for Pod states
const (
	PodRunning PodStatus = iota + 1
	PodCompleted
)

func (s PodStatus) String() string {
	switch s {
	case PodRunning:
		return "Running"
	case PodCompleted:
		return "Completed"
	default:
		return ErrUnknownPodStatus.Error()
	}
}

// Resource is a k8s resource
type Resource int

// Enums for Resource
const (
	Namespace Resource = iota + 1
	Pod
	Deployment
	StatefulSet
	PVC
	ConfigMap
	Secret
)

// --- optinoal arguments to WaitFor

type options struct {
	NumReady int32
	PodName  string
	Status   PodStatus
}

// Option represents a optional argument to WaitFor
type Option interface {
	apply(*options)
}

// ---
type podNameOption string

func (s podNameOption) apply(opts *options) {
	opts.PodName = string(s)
}

// WithPodName sets PodName
func WithPodName(n string) Option {
	return podNameOption(n)
}

// ---
type numReadyOption int32

func (s numReadyOption) apply(opts *options) {
	opts.NumReady = int32(s)
}

// WithNumReady sets NumReady as int32
func WithNumReady(n int32) Option {
	return numReadyOption(n)
}

// ---
type status PodStatus

func (s status) apply(opts *options) {
	opts.Status = PodStatus(s)
}

// WithStatus sets the expected Pod status
func WithStatus(n PodStatus) Option {
	return status(n)
}

// ---

// WaitFor waits for a resource to be in a ready, unready, etc. state and
// returns with an error when the ctx timed out, or with nil
func (k *k8sTest) WaitFor(ctx context.Context, client *kubernetes.Clientset, resource Resource, opts ...Option) error {

	options := options{}

	for _, o := range opts {
		o.apply(&options)
	}

	bo := backoff.Backoff{
		Min:    time.Second,
		Max:    5 * time.Second,
		Jitter: true,
		// Factor: float64(0.3),
	}

	t := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(bo.Duration()):
			// continue below
		}

		switch resource {
		case Namespace:
			ns, err := client.CoreV1().Namespaces().Get(ctx, k.config.K8sNamespace, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if ns != nil {
				return nil
			}
		case Deployment:
			deployment, err := client.AppsV1().Deployments(k.config.K8sNamespace).Get(ctx, "smoketest", metav1.GetOptions{})
			if err != nil {
				continue
			}
			if deployment != nil {
				if deployment.Status.AvailableReplicas == options.NumReady {
					return nil
				}
			}
			glog.V(2).Infof("waiting for pods to become available: %v", time.Since(t))

		case Pod:

			if options.Status == 0 {
				options.Status = PodRunning // default to waiting for a Running pod
			}

			tmpPod, err := client.CoreV1().Pods(k.config.K8sNamespace).Get(ctx, options.PodName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			switch options.Status {
			case PodRunning:
				if tmpPod.Status.Phase == v1.PodRunning {
					return nil
				}
			case PodCompleted:
				if len(tmpPod.Status.ContainerStatuses) > 0 {
					if tmpPod.Status.ContainerStatuses[0].State.Terminated != nil {
						if tmpPod.Status.ContainerStatuses[0].State.Terminated.Reason == PodCompleted.String() {
							return nil
						}
					}
				}
			}

			glog.V(2).Infof("waiting for pod to be %s: %v", options.Status.String(), time.Since(t))

		case StatefulSet:
			return ErrNotImplemented
		case PVC:
			return ErrNotImplemented
		case ConfigMap:
			return ErrNotImplemented
		case Secret:
			return ErrNotImplemented
		default:
			return ErrUnknownResourceType
		}

	}
}
