package tools

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// MustBuildKubeClient returns a Kubernetes client.
func MustBuildKubeClient() (*kubernetes.Clientset, error) {
	config, err := mustGetRestConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientSet, err
}

// MustBuildDynamicClient returns a dynamic client.
func MustBuildDynamicClient() (dynamic.Interface, error) {
	config, err := mustGetRestConfig()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynamicClient, err
}

func mustGetRestConfig() (*rest.Config, error) {
	kubeConfig := os.Getenv(KubeConfigPathEnv)
	if kubeConfig == "" {
		kubeConfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetMinTime returns the minimum time of the given times.
func GetMinTime(sorted []ObjectFinishTime) (string, time.Duration) {
	if len(sorted) == 0 {
		return "", 0
	}

	return sorted[0].Name, sorted[0].Finished.Sub(sorted[0].Started)
}

// GetMaxTime returns the maximum time of the given times.
func GetMaxTime(sorted []ObjectFinishTime) (string, time.Duration) {
	if len(sorted) == 0 {
		return "", 0
	}

	return sorted[len(sorted)-1].Name, sorted[len(sorted)-1].Finished.Sub(sorted[len(sorted)-1].Started)
}

// GetAvgTime returns the average time of the given times.
func GetAvgTime(times []ObjectFinishTime) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range times {
		total += t.Finished.Sub(t.Started)
	}

	return total / time.Duration(len(times))
}

// GetMedianTime returns the median time of the given times.
func GetMedianTime(sorted []ObjectFinishTime) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	if len(sorted)%2 == 0 {
		return sorted[len(sorted)/2].Finished.Sub(sorted[len(sorted)/2].Started)
	}

	return (sorted[len(sorted)/2].Finished.Sub(sorted[len(sorted)/2].Started) + sorted[len(sorted)/2+1].Finished.Sub(sorted[len(sorted)/2+1].Started)) / 2
}

// GetTotalTime returns the total time of the given times.
func GetTotalTime(times []ObjectFinishTime) time.Duration {
	var total time.Duration
	for _, t := range times {
		total += t.Finished.Sub(t.Started)
	}

	return total
}

// Sort sorts the given times.
func Sort(times []ObjectFinishTime) {
	sort.Slice(times, func(i, j int) bool {
		return times[i].Finished.Sub(times[i].Started) < times[j].Finished.Sub(times[j].Started)
	})
}
