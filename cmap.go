package cmap

import (
	"context"
	"flag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
)

// setupConfMap
func SetupConfMap(namespace, cmap_name string, data map[string]string, clientset *kubernetes.Clientset) *v1.ConfigMap {
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmap_name,
			Namespace: namespace,
		},
		Data: data,
	}

	conf := createConfMap(namespace, configMap, clientset)
	return conf
}

// create configMap
func createConfMap(namespace string, cmap *v1.ConfigMap, clientset *kubernetes.Clientset) *v1.ConfigMap {
	conf, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cmap, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		log.Printf("configMap %s already found in namespace %s \n", cmap.Name, namespace)
		return conf
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("error: configMap %s not created in namespace %s: %v\n",
			cmap.Name, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		log.Printf("error: configMap %s not created in namespace %s", cmap.Name, namespace)
	} else {
		log.Printf("configMap %s created", cmap.Name)
		return conf
	}
	return nil
}

// get configMap details
func GetConfMap(namespace, cmap string, clientset *kubernetes.Clientset) *v1.ConfigMap {
	conf, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), cmap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("error: configMap %s not found in namespace %s\n", cmap, namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("error: configMap  %s not found in namespace %s: %v\n",
			cmap, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		log.Printf("error: configMap %s not found in namespace %s\n", cmap, namespace)
	} else {
		log.Printf("configMap %s found in namespace %s\n", cmap, namespace)
		return conf
	}
	return nil
}

func PutConfMap(b []byte, namespace, cmap string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Patch(context.TODO(), cmap, types.MergePatchType, b, metav1.PatchOptions{})

	if errors.IsNotFound(err) {
		log.Printf("error: configMap %s not found in %s\n", cmap, namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("error: configMap %s not found in namespace %s: %v\n",
			cmap, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		log.Printf("error: configMap %s not found in %s\n", cmap, namespace)
	}
	return err
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// creates in-cluster config
func setupInK8sClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

// creates out-of-cluster config
func setupOutK8sClient() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

func SetupK8sClient(namespace, cmap_name string, confMapData map[string]string, outClusterPtr *bool) *kubernetes.Clientset {
	var clientset *kubernetes.Clientset
	if *outClusterPtr {
		// create the clientset outside cluster
		clientset = setupOutK8sClient()
	} else {
		clientset = setupInK8sClient()
	}

	cm := SetupConfMap(namespace, cmap_name, confMapData, clientset)
	if cm == nil {
		return nil
	}
	return clientset
}
