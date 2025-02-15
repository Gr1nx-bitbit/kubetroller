package main

import (
	"context"
	// "fmt"
	"flag"
	"time"

	v1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// "k8s.io/apimachinery/pkg/util/wait"
	// "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	picturesv1 "github.com/eddiezane/that-conference-k8s-controller/pkg/apis/pictures/v1"
	customclientset "github.com/eddiezane/that-conference-k8s-controller/pkg/generated/clientset/versioned"
	informers "github.com/eddiezane/that-conference-k8s-controller/pkg/generated/informers/externalversions"
	// craiyon "github.com/eddiezane/that-conference-k8s-controller/pkg/craiyon"
	// deepai "github.com/eddiezane/that-conference-k8s-controller/pkg/text2image"
)

type controller struct {
	queue        workqueue.RateLimitingInterface
	informer     cache.SharedIndexInformer
	customClient customclientset.Interface
	client       *kubernetes.Clientset

	// craiyon *craiyon.Client
	// deepai *deepai.Client
}

// This method starts up the controllers processes like the informer.
func (c *controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	// since this method gets blocked by the channel we're starting
	// it in a seperate thread so the rest of the execution isn't halted
	go c.informer.Run(stopCh)

	// wait for the cache to sync (i'm presuming that it's getting info from the apiServer)
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		panic("can't sync cache and inforer")
	}

	// start workers in a seperate goroutine (i don't know what these are)
	// ok, so this starts the actual business logic in another goroutine and
	// I mean that is literally all it does. It just keeps checking to see
	// of processNextItem returns a false and that's how the controller keeps running!
	go c.runWorker()

	// wait for the channel
	<-stopCh
}

// this is a forever loop which processes items found in the controllers work queue
func (c *controller) runWorker() {
	klog.Info("starting worker")
	for c.processNextItem() {
		// loop forever
	}
}

func (c *controller) processNextItem() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	go func(key string) {
		defer c.queue.Done(key)

		err := c.processItem(key)
		if err == nil {
			klog.InfoS("item processed", "key", key)
		} else if c.queue.NumRequeues(key) > 3 {
			defer c.queue.Forget(key) // since we're no longer retrying this item, we'll just forget about it
			klog.InfoS("retry limit reached", "key", key)
			utilruntime.HandleError(err) // this will just log the error instead of killing the program
		} else {
			klog.InfoS("retrying", "key", key)
			c.queue.AddRateLimited(key)
		}
	}(key.(string))

	return true
}

// so now we have to acutally get info out of our CR
func (c *controller) processItem(key string) error {
	item, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	// if the item is in the queue but doesn't actually have any content, we'll get rid of it
	if !exists {
		klog.InfoS("item deleted", "key", key)
		return nil
	}

	// the method returns an empty interface{} so we have to type cast it to our CR
	podCustomizer := item.(*picturesv1.PodCustomizer)

	// TODO: Check if the customizer has work that needs to be done and react accordingly
	if !podCustomizer.Spec.Promote {
		err := c.client.CoreV1().Pods("default").Delete(context.TODO(), "test", metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}

		return err
	} else {
		pod, err := c.client.CoreV1().Pods("default").Get(context.TODO(), "test", metav1.GetOptions{})
		if err != nil {
			panic(err)
		}

		podSpec := apiv1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:   pod.Name,
				Labels: pod.Labels,
			},
			Spec: pod.Spec,
		}

		result, err := createDeployment("test-deployment", "default", 3, podSpec, c.client)
		if err != nil {
			return err
		}

		klog.InfoS("Pod promoted", "deployment", result.Name)

		return nil
	}

}

func createDeployment(name string, ns string, replicas int, podSpec apiv1.PodTemplateSpec, client *kubernetes.Clientset) (*v1.Deployment, error) {
	reps := intToPointer(replicas)

	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.DeploymentSpec{
			Replicas: reps,
			Selector: &metav1.LabelSelector{
				MatchLabels: podSpec.Labels,
			},
			Template: podSpec,
		},
	}

	result, err := client.AppsV1().Deployments(ns).Create(context.TODO(), deployment, metav1.CreateOptions{})

	return result, err
}

func intToPointer(num int) *int32 {
	n := int32(num)
	return &n
}

func main() {
	klog.InitFlags(nil)

	configFlags := genericclioptions.NewConfigFlags(true) // this will generate a client for the controller and figure out the best method to do so i.e. in cluster, --kubeconfig, client-go

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		panic(err)
	}

	customClient := customclientset.NewForConfigOrDie(config)                         // this is just the clientset from the code-gen
	factory := informers.NewSharedInformerFactory(customClient, 30*time.Second)       // the informers will resync every 30 seconds
	informer := factory.Kuberneddies().V1().PodCustomizers().Informer()               // make the actual informer
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()) // make a queue for the controller to use

	controller := NewController(queue, informer, customClient)

	// add callback functions for when these resources get added, updated, and deleted
	controller.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj) // this will return us the object which we can access
			if err != nil {
				panic(err)
			}

			klog.InfoS("adding to queue", "key", key)
			controller.queue.Add(key)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// type cast the interfaces returned as PodCustomizers or whatever CRD you're using
			old := oldObj.(*picturesv1.PodCustomizer)
			new := newObj.(*picturesv1.PodCustomizer)

			key, err := cache.MetaNamespaceKeyFunc(new)
			if err != nil {
				panic(err)
			}

			// now we can actually check if work needs to be done by comparing each resource versions of each object
			if old.ResourceVersion == new.ResourceVersion {
				klog.InfoS("preiodic resync", "key", key)
				return
			}

			klog.InfoS("queueing PodCustomizer for update", "key", key)
			queue.Add(key)
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				panic(err)
			}

			klog.InfoS("delete callback invoked. just letting you know", "key", key)
			// UP UNTIL THIS POINT, THIS IS ALL BOILERPLATE!! WE'RE GOING TO GET TO THE BUSINESS LOGIC SOON ENOUGH
		},
	})

	// uhh, this context is here in case we need to kill the controller via "control C" something similar
	ctx := signals.SetupSignalHandler()

	klog.Info("Starting controller")
	controller.Run(ctx.Done())
	klog.Info("Stopping controller")
}

func NewController(
	queue workqueue.RateLimitingInterface,
	informer cache.SharedIndexInformer,
	customClient customclientset.Interface,
) *controller {
	return &controller{
		queue:        queue,
		informer:     informer,
		customClient: customClient,
		// craiyon:      craiyon.NewClient(),
		client: newClient(),
		// deepai:   deepai.NewClient(),
	}
}

func (c *controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	klog.InfoS("adding to queue", "key", key)
	c.queue.Add(key)
}

func newClient() *kubernetes.Clientset {
	kubeconfig := flag.String("kubeconfig", "../config/config", "kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}
