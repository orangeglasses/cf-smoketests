package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"

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
	RunTestPart(k.CreateService, "Create Service", &results)
	RunTestPart(k.CreateIngress, "Create Ingress", &results)
	//TODO: test connection to test deploy here
	//sleep for now :(

	RunTestPart(k.TestReachable, "Test Connection", &results)
	RunTestPart(k.DeleteIngress, "Delete Ingress", &results)
	RunTestPart(k.DeleteService, "Delete Service", &results)
	RunTestPart(k.DeleteDeployment, "Delete Deployment", &results)

	return OverallResult(k8sKey, k8sName, results)
}

// CreateDeployment creates a dummy nginx deployment of 2 pods
func (k *k8sTest) CreateDeployment() (interface{}, error) {
	log.Println("Creating k8s deployment")
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
					Containers:       []corev1.Container{{Image: k.testImage, Name: "webserver"}},
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: os.Getenv("K8S_IMG_PULL_SECRET")}},
				},
			},
		},
	}

	_, err := k.client.AppsV1().Deployments(k.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to create deployment: %v", err)
	}

	if err = k.WaitFor(ctx, k.client, Deployment, WithNumReady(numReplicas)); err != nil {
		return nil, fmt.Errorf("Failed waiting for deployment to be created: %v", err)
	}

	return true, nil
}

// DeleteDeployment deletes the deployment ..
func (k *k8sTest) DeleteDeployment() (interface{}, error) {
	log.Println("Deleting k8s deployment")
	ctx := context.Background()
	if err := k.client.AppsV1().Deployments(k.namespace).Delete(ctx, "smoketest", metav1.DeleteOptions{}); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to delete deployment: %v", err)
	}

	return true, nil
}

func (k *k8sTest) CreateIngress() (interface{}, error) {
	log.Println("Creating k8s ingress")
	ctx := context.Background()

	host1 := os.Getenv("K8S_ING_HOST1")

	if host1 == "" {
		return true, nil
	}

	pathType := networkingV1.PathType("Prefix")

	ingress := networkingV1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "smoketest-ingress",
		},
		Spec: networkingV1.IngressSpec{
			TLS: []networkingV1.IngressTLS{{Hosts: []string{host1}, SecretName: os.Getenv("K8S_ING_HOST1_TLS_SECRET")}},
			Rules: []networkingV1.IngressRule{
				{Host: host1,
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingV1.IngressBackend{
									Service: &networkingV1.IngressServiceBackend{
										Name: "smoketest-svc",
										Port: networkingV1.ServiceBackendPort{Number: 80},
									},
								},
							}},
						},
					},
				},
			},
		},
	}

	_, err := k.client.NetworkingV1().Ingresses(k.namespace).Create(ctx, &ingress, metav1.CreateOptions{})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return true, nil
}

// DeleteDeployment deletes the deployment ..
func (k *k8sTest) DeleteIngress() (interface{}, error) {
	log.Println("Deleting k8s ingress")
	ctx := context.Background()
	if err := k.client.NetworkingV1().Ingresses(k.namespace).Delete(ctx, "smoketest-ingress", metav1.DeleteOptions{}); err != nil {
		return nil, fmt.Errorf("failed to delete ingress: %v", err)
	}

	return true, nil
}

func (k *k8sTest) CreateService() (interface{}, error) {
	log.Println("Creating k8s service")
	ctx := context.Background()

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "smoketest-svc",
		},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Name: "http", Port: 80, Protocol: "TCP"}},
			Selector: map[string]string{"app": "smoketest"},
			Type:     "ClusterIP",
		},
	}
	k.client.CoreV1().Services(k.namespace).Create(ctx, &service, metav1.CreateOptions{})

	return true, nil
}

func (k *k8sTest) DeleteService() (interface{}, error) {
	log.Println("Deleting k8s service")
	ctx := context.Background()
	if err := k.client.CoreV1().Services(k.namespace).Delete(ctx, "smoketest-svc", metav1.DeleteOptions{}); err != nil {
		return nil, fmt.Errorf("failed to delete service: %v", err)
	}

	return true, nil
}

func (k *k8sTest) TestReachable() (interface{}, error) {
	log.Println("Testing connection to deployment")
	var status int

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	for retries := 12; retries > 0 && status != 200; retries-- {
		r, err := httpClient.Get("https://" + os.Getenv("K8S_ING_HOST1"))
		if err != nil {
			return nil, err
		}

		status = r.StatusCode
		time.Sleep(250 * time.Millisecond)
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to reach test deployment")
	}

	return true, nil
}
