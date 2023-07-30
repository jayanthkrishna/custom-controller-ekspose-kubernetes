package main

import (
	"flag"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	kubeconfig := flag.String("kubeconfig", "/home/eangrki/.kube/config", "location to your config file")

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		fmt.Printf("error %s building from flags", err.Error())

		config, err = rest.InClusterConfig()

		if err != nil {
			fmt.Printf("error %s from InClusterConfig", err.Error())
		}
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		fmt.Printf("error %s building the client", err.Error())
	}

	fmt.Println(clientSet)
}
