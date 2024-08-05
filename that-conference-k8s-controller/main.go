package main

import (
	// "context"
	// "fmt"
	"time"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// "k8s.io/apimachinery/pkg/util/wait"
	// "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	picturesv1 "github.com/eddiezane/that-conference-k8s-controller/pkg/apis/pictures/v1"
	clientset "github.com/eddiezane/that-conference-k8s-controller/pkg/generated/clientset/versioned"
	informers "github.com/eddiezane/that-conference-k8s-controller/pkg/generated/informers/externalversions"

	craiyon "github.com/eddiezane/that-conference-k8s-controller/pkg/craiyon"
	// deepai "github.com/eddiezane/that-conference-k8s-controller/pkg/text2image"
)

type controller struct {
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
	client   clientset.Interface

	craiyon *craiyon.Client
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
			klog.InfoS("work finished", "key", key)
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
	}

	// the method returns an empty interface{} so we have to type cast it to our CR
	podCustomizer := item.(*picturesv1.PodCustomizer)

	// TODO: Check if the customizer has work that needs to be done and react accordingly
}

func main() {
	klog.InitFlags(nil)

	configFlags := genericclioptions.NewConfigFlags(true) // this will generate a client for the controller and figure out the best method to do so i.e. in cluster, --kubeconfig, client-go

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		panic(err)
	}

	client := clientset.NewForConfigOrDie(config)                                     // this is just the clientset from the code-gen
	factory := informers.NewSharedInformerFactory(client, 30*time.Second)             // the informers will resync every 30 seconds
	informer := factory.Kuberneddies().V1().PodCustomizers().Informer()               // make the actual informer
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()) // make a queue for the controller to use

	controller := NewController(queue, informer, client)

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
	client clientset.Interface,
) *controller {
	return &controller{
		queue:    queue,
		informer: informer,
		client:   client,
		craiyon:  craiyon.NewClient(),
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
