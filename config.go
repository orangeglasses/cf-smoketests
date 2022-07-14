package main

import "github.com/kelseyhightower/envconfig"

type SmokeTestConfig struct {
	KubeconfigPath       string   `envconfig:"KUBECONFIG_PATH" required:"false"`
	K8sNamespace         string   `envconfig:"K8S_NAMESPACE" required:"false"`
	K8sTestImage         string   `envconfig:"K8S_TESTIMAGE" required:"false"`
	K8sImgPullSecret     string   `envconfig:"K8S_IMG_PULL_SECRET" required:"false"`
	K8sIngHosts          []string `envconfig:"K8S_ING_HOSTS" required:"false"`
	K8sIngHostsTlsSecret []string `envconfig:"K8S_ING_HOSTS_TLS" required:"false"`
	K8sIngHostsClass     []string `envconfig:"K8S_ING_HOSTS_CLASS" required:"false"`
}

func smokeTestsConfigLoad() (SmokeTestConfig, error) {
	var config SmokeTestConfig

	err := envconfig.Process("", &config)
	if err != nil {
		return SmokeTestConfig{}, err
	}

	return config, nil
}
