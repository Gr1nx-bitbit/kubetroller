package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	// v1 "k8s.io/api/apps/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	deployinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type ClusterConfig struct {
	clusterName string
	configPath  string
}

type Controller struct {
	clusterName        string
	configPath         string
	client             kubernetes.Interface
	kInformerFactory   kubeinformers.SharedInformerFactory
	deploymentInformer deployinformers.DeploymentInformer
	workqueue          workqueue.TypedRateLimitingInterface[cache.ObjectName]
	recorder           record.EventRecorder
}

func main() {
	ctx := signals.SetupSignalHandler()
	var clusterString string
	flag.StringVar(&clusterString, "clusters", "EMPTY", "specify the names of the clusters and their kubeconfig file in a colon-pair comma seperated format, e.g. -clusters='name1:config,name2:config' ")
	flag.Parse()

	// so now that we can get all the kubeconfig files, we have to build each client seperately...
	// idk if trying to build the same client twice will break the program... guess we'll see!
	controllers := make(map[string]*Controller)
	clusterConfigs := getClustersFromFlag(clusterString)
	for index, clusterConfig := range clusterConfigs {
		config, err := clientcmd.BuildConfigFromFlags("", clusterConfig.configPath)
		if err != nil {
			fmt.Printf("Something went wrong with cluster config #%d! Error: %s\n", index, err.Error())
			os.Exit(1)
		}

		kclient, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Println("Trouble building client! Error: ", err.Error())
			os.Exit(2)
		}

		controllers[clusterConfig.clusterName] = NewController(ctx, kclient, clusterConfig)
	}

	msg := fmt.Sprintf("Invoking controller %s", controllers["default"].clusterName)
	klog.InfoS(msg)
	if err := controllers["default"].Run(ctx); err != nil {
		klog.FlushAndExit(klog.ExitFlushTimeout, 2)
	}

	// var wg sync.WaitGroup
	// for controllerName, controller := range controllers {
	// 	msg := fmt.Sprintf("Invoking controller %s", controllerName)
	// 	klog.InfoS(msg)
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		if err := controller.Run(ctx); err != nil {
	// 			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	// 		}
	// 	}()
	// }

	// wg.Wait()
}

func getClustersFromFlag(clusterString string) []ClusterConfig {
	var clusterConfigs []ClusterConfig

	for _, clusterPair := range strings.Split(clusterString, ",") {
		pair := strings.Split(clusterPair, ":")
		clusterConfigs = append(clusterConfigs, ClusterConfig{
			clusterName: pair[0],
			configPath:  pair[1],
		})
	}

	return clusterConfigs
}

/*
	So now that we have multiple clients, we need to spawn several controllers... well, we could do that
	or is there a way of collapsing all the controllers into one and just having seperate clients? Well, each
	client will also need an informer, and a workqueue so it is already a controller. Well, ok. By the way,
	do we even need access to the controllers after we put the event handlers on them? Well, no... you kind
	of just put the business logic and the event listeners and then you're hands off. Ok, so let's create
	multiple instances of a controller that each listens to a diff namespace maybe?
*/

func NewController(
	ctx context.Context,
	clientset kubernetes.Interface,
	config ClusterConfig) *Controller {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientset.CoreV1().Events("")})

	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: config.clusterName})
	ratelimiter := workqueue.NewTypedMaxOfRateLimiter(
		workqueue.NewTypedItemExponentialFailureRateLimiter[cache.ObjectName](5*time.Millisecond, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[cache.ObjectName]{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)
	informerFactory := kubeinformers.NewSharedInformerFactory(clientset, time.Second*30)

	controller := &Controller{
		clusterName:        config.clusterName,
		configPath:         config.configPath,
		client:             clientset,
		kInformerFactory:   informerFactory,
		deploymentInformer: informerFactory.Apps().V1().Deployments(),
		workqueue:          workqueue.NewTypedRateLimitingQueue(ratelimiter),
		recorder:           recorder,
	}

	message := fmt.Sprintf("Setting up event handler for controller %s", config.clusterName)
	klog.Info(message)

	// need to make the method for this thing -- HERE
	controller.deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueDeployment,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueDeployment(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			if objRef, err := cache.ObjectToName(obj); err != nil {
				utilruntime.HandleError(err)
			} else {
				logger.Info("delete callback invoked!", "key", objRef)
			}
		},
	})

	return controller
}

// So next we have to start the informer factories which we can do in new controller
// Then we have to actually make the method to start the controller

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	if ok := cache.WaitForCacheSync(ctx.Done(), c.deploymentInformer.Informer().HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync! controller: %s", c.clusterName)
	}

	logger.Info("Starting controller, workers, and informer!", "controller", c.clusterName)
	c.kInformerFactory.Start(ctx.Done())

	go wait.UntilWithContext(ctx, c.runWorker, time.Second)

	logger.Info("Started workers", "controller", c.clusterName)
	<-ctx.Done()
	logger.Info("Sutting down workers", "controller", c.clusterName)

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.workqueue.Get()
	// logger := klog.FromContext(ctx)

	if shutdown {
		// logger.Info("Queue signaled shutdown!", "controller", c.clusterName)
		return false
	}

	defer c.workqueue.Done(objRef)

	err := c.syncHandler(ctx, objRef)

	if err == nil {
		c.workqueue.Forget(objRef)
		// logger.Info("Successfully synced", "object reference", objRef, "controller", c.clusterName)
		return true
	}

	// yeah, if we get an error, we'll just retry
	utilruntime.HandleErrorWithContext(ctx, err, "Error syncing; requeuing for later retry", "objectReference", objRef, "controller", c.clusterName)

	// I don't know if this will forget an object after it's been retried after a certain amount of requeues
	// I guess we'll see (we can delibretally fail an object)
	c.workqueue.AddRateLimited(objRef)
	return true
}

/*
Ok, so for this controller we want to get info about all of the deployments in the cluster (it'll be specific ns later)
I think just as a sanity check let's print out the name, namespace and image of each deployment in the cluster
*/
func (c *Controller) syncHandler(ctx context.Context, objref cache.ObjectName) error {

	logger := klog.FromContext(ctx)
	msg := fmt.Sprintf("%s : %s | controller: %s", objref.Namespace, objref.Name, c.clusterName)
	logger.Info(msg)
	return nil
	// namespaces, err := c.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	return err
	// }

	// for _, namespace := range namespaces.Items {
	// 	deployList, err := c.client.AppsV1().Deployments(namespace.Name).List(context.TODO(), metav1.ListOptions{})
	// 	if err != nil {
	// 		return err
	// 	}

	// 	for _, deployment := range deployList.Items {
	// 		msg := fmt.Sprintf("%s : %s", deployment.Namespace, deployment.Name)
	// 		logger.Info(msg, "controller", c.clusterName)
	// 	}
	// }

	// return nil
}

func (c *Controller) enqueueDeployment(obj interface{}) {
	if objref, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		// klog.InfoS("Adding to queue", "key", objref, "controller", c.clusterName)
		c.workqueue.Add(objref)
	}
}
