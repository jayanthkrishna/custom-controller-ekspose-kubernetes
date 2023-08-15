package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
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

	for c.processItem() {

	}

}

func (c *controller) processItem() bool {

	item, shutdown := c.queue.Get()

	if shutdown {
		return false
	}

	defer c.queue.Forget(item)

	key, err := cache.MetaNamespaceKeyFunc(item)

	if err != nil {
		fmt.Printf("Error getting key from cache : %s\n", err.Error())
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)

	if err != nil {
		fmt.Printf("Error splitting key into namespace and name : %s\n", err.Error())
		return false
	}

	err = c.syncDeployment(ns, name)

	return true

}

func (c *controller) syncDeployment(ns, name string) error {

	// create service
	ctx := context.Background()

	deployment, err := c.deploymentLister.Deployments(ns).Get(name)

	if err != nil {
		fmt.Printf("Error getting deployment from lister. namespace - %s and deployment name - %s \n", ns, name)
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: deplLabels(*deployment),
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}

	s, err := c.clientSet.CoreV1().Services(ns).Create(ctx, &svc, metav1.CreateOptions{})

	if err != nil {
		fmt.Printf("Error creating service : %s\n", err.Error())
	}

	return createIngress(ctx, c.clientSet, s)
}

func createIngress(ctx context.Context, client kubernetes.Interface, svc *corev1.Service) error {
	pathType := "Prefix"
	ingress := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     fmt.Sprintf("%s", svc.Name),
									PathType: (*netv1.PathType)(&pathType),
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: svc.Name,
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := client.NetworkingV1().Ingresses(svc.Namespace).Create(ctx, &ingress, metav1.CreateOptions{})

	return err

}
func deplLabels(depl appsv1.Deployment) map[string]string {
	return depl.Spec.Template.Labels
}

func (c *controller) handleAdd(obj interface{}) {

	fmt.Println("Add was called")

	c.queue.Add(obj)

}

func (c *controller) handleDel(obj interface{}) {

	fmt.Println("Delete was called")

	c.queue.Add(obj)
}
