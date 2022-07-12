package main

import (
	"context"
	"fmt"
	"log"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	k8sKey  = "kubernetes"
	k8sName = "Kubernetes"
)

type k8sTest struct {
	client    *kubernetes.Clientset
	namespace string
	testImage string
}

func k8sTestNew() SmokeTest {
	kubeconfig := os.Getenv("KUBECONFIG_PATH")
	if kubeconfig == "" {
		return nil
	}

	konfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Printf("unable to load kubeconfig: %s", err.Error())
		return nil
	}

	cs, err := kubernetes.NewForConfig(konfig)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}

	return &k8sTest{
		client:    cs,
		namespace: "smoketest",
		testImage: os.Getenv("K8S_TESTIMAGE"),
	}
}

func (k *k8sTest) run() SmokeTestResult {

	var results []SmokeTestResult

	RunTestPart(k.CreateDeployment, "Create Deployment", &results)
	//test connection to test deploy here
	RunTestPart(k.DeleteDeployment, "Delete Deployment", &results)

	return OverallResult(k8sKey, k8sName, results)
}

// CreateDeployment creates a dummy nginx deployment of 2 pods
func (k *k8sTest) CreateDeployment() (interface{}, error) {
	ctx := context.Background()

	numReplicas := int32(2)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "smoketest",
			Labels: map[string]string{
				"testName": "deployment",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "smoketest",
				},
			},
			MinReadySeconds: int32(7),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"app": "smoketest",
					},
				},
				Spec: corev1.PodSpec{
//					Volumes:                       []corev1.Volume{},					
					Containers:                    []corev1.Container{Image: k.testImage, Name: "webserver"},					
					ImagePullSecrets:              []corev1.LocalObjectReference{Name: os.Getenv("K8S_IMG_PULL_SECRET")},					
				},
				},
			},
		},
	}

	_, err := k.client.AppsV1().Deployments(k.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to create deployment: %v", err)
	}

	if err = k.WaitFor(ctx, k.client, Deployment, WithNumReady(numReplicas)); err != nil {
		return nil, fmt.Errorf("Failed to create deployment: %v", err)
	}

	return true, nil
}

// DeleteDeployment deletes the deployment ..
func (k *k8sTest) DeleteDeployment() (interface{}, error) {
	ctx := context.Background()
	if err := k.client.AppsV1().Deployments(k.namespace).Delete(ctx, "smoketest", metav1.DeleteOptions{}); err != nil {
		return nil, fmt.Errorf("failed to delete deployment: %v", err)
	}

	return true, nil
}
