package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientSet kubernetes.Interface

	deploymentLister appslisters.DeploymentLister

	depCacheSynced cache.InformerSynced

	queue workqueue.RateLimitingInterface
}

func newController(clientSet kubernetes.Interface, deployInformer appsinformers.DeploymentInformer) *controller {

	c := &controller{
		clientSet:        clientSet,
		deploymentLister: deployInformer.Lister(),
		depCacheSynced:   deployInformer.Informer().HasSynced,
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekspose"),
	}

	deployInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    handleAdd,
			DeleteFunc: handleDel,
		},
	)

	return c
}

func (c *controller) run(ch <-chan struct{}) {

	fmt.Println("Starting the controller")
	if !cache.WaitForCacheSync(ch, c.depCacheSynced) {

		fmt.Println("wait for cache to be synced")
	}

	go wait.Until(c.worker, 1*time.Second, ch)

	<-ch

}

func (c *controller) worker() {

}

func handleAdd(obj interface{}) {

	fmt.Println("Add was called")

}

func handleDel(obj interface{}) {

	fmt.Println("Delete was called")
}
