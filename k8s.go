package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
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
	client *kubernetes.Clientset
	config SmokeTestConfig
}

func k8sTestNew(config SmokeTestConfig) SmokeTest {
	if config.KubeconfigPath == "" {
		return nil
	}

	konfig, err := clientcmd.BuildConfigFromFlags("", config.KubeconfigPath)
	if err != nil {
		log.Printf("Unable to load kubeconfig: %s", err.Error())
		return nil
	}

	cs, err := kubernetes.NewForConfig(konfig)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}

	return &k8sTest{
		client: cs,
		config: config,
	}
}

func (k *k8sTest) run() SmokeTestResult {

	var results []SmokeTestResult

	RunTestPart(k.CreateDeployment, "Create Deployment", &results)

	//skip other tests if deployment fails
	if !results[0].Result {
		RunTestPart(k.DeleteDeployment, "Delete Deployment", &results)
		return OverallResult(k8sKey, k8sName, results)
	}

	RunTestPart(k.CreateService, "Create Service", &results)
	RunTestPart(k.CreateIngresses, "Create Ingresses", &results)

	RunTestPart(k.TestConnections, "Test Connection", &results)

	RunTestPart(k.DeleteIngresses, "Delete Ingresses", &results)
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
					Containers:       []corev1.Container{{Image: k.config.K8sTestImage, Name: "webserver"}},
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: k.config.K8sImgPullSecret}},
				},
			},
		},
	}

	_, err := k.client.AppsV1().Deployments(k.config.K8sNamespace).Create(ctx, deployment, metav1.CreateOptions{})
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
	if err := k.client.AppsV1().Deployments(k.config.K8sNamespace).Delete(ctx, "smoketest", metav1.DeleteOptions{}); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to delete deployment: %v", err)
	}

	return true, nil
}

func (k *k8sTest) CreateIngresses() (interface{}, error) {
	var errs []error

	for i, hostname := range k.config.K8sIngHosts {
		err := k.CreateIngress(hostname, k.config.K8sIngHostsTlsSecret[i], k.config.K8sIngHostsClass[i])
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("One or more ingresses failed to create: %v", errs)
	}

	return true, nil
}

func (k *k8sTest) DeleteIngresses() (interface{}, error) {
	var errs []error

	for _, hostname := range k.config.K8sIngHosts {
		err := k.DeleteIngress(hostname)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("One or more ingresses failed to delete: %v", errs)
	}

	return true, nil
}

func (k *k8sTest) CreateIngress(hostname string, tlsSecret string, ingressClass string) error {
	log.Println("Creating k8s ingress")
	ctx := context.Background()

	pathType := networkingV1.PathType("Prefix")

	ingress := networkingV1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "smoketest-ingress-" + hostname,
		},
		Spec: networkingV1.IngressSpec{
			TLS: []networkingV1.IngressTLS{{Hosts: []string{hostname}, SecretName: tlsSecret}},
			Rules: []networkingV1.IngressRule{
				{Host: hostname,
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

	if ingressClass != "" && ingressClass != "-" {
		ingress.ObjectMeta.Annotations = map[string]string{
			"kubernetes.io/ingress.class": ingressClass,
		}

	}

	_, err := k.client.NetworkingV1().Ingresses(k.config.K8sNamespace).Create(ctx, &ingress, metav1.CreateOptions{})
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (k *k8sTest) DeleteIngress(hostname string) error {
	log.Println("Deleting k8s ingress")
	ctx := context.Background()
	if err := k.client.NetworkingV1().Ingresses(k.config.K8sNamespace).Delete(ctx, "smoketest-ingress-"+hostname, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete ingress: %v", err)
	}

	return nil
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
	k.client.CoreV1().Services(k.config.K8sNamespace).Create(ctx, &service, metav1.CreateOptions{})

	return true, nil
}

func (k *k8sTest) DeleteService() (interface{}, error) {
	log.Println("Deleting k8s service")
	ctx := context.Background()
	if err := k.client.CoreV1().Services(k.config.K8sNamespace).Delete(ctx, "smoketest-svc", metav1.DeleteOptions{}); err != nil {
		return nil, fmt.Errorf("failed to delete service: %v", err)
	}

	return true, nil
}

func (k *k8sTest) TestConnections() (interface{}, error) {
	var errs []error

	for _, hostname := range k.config.K8sIngHosts {
		err := k.TestConnection(hostname)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("One or more connection tests failed: %v", errs)
	}

	return true, nil
}

func (k *k8sTest) TestConnection(hostname string) error {
	log.Println("Testing connection to deployment")
	var status int

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	for retries := 60; retries > 0 && status != 200; retries-- {
		r, err := httpClient.Get("https://" + hostname)
		if err != nil {
			return err
		}

		status = r.StatusCode
		time.Sleep(500 * time.Millisecond)
	}

	if status != 200 {
		return fmt.Errorf("failed to reach test deployment (hostname: %v)", hostname)
	}

	return nil
}
