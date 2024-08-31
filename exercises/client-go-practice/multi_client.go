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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

type ClusterConfig struct {
	clusterName string
	configPath  string
}

type Cluster struct {
	clusterName string
	configPath  string
	client      kubernetes.Interface
}

type Controller struct {
	clusterName string
	configPath  string
	client      kubernetes.Interface
	// kInformerFactory   kubeinformers.SharedInformerFactory
	deploymentInformer deployinformers.DeploymentInformer
	workqueue          workqueue.TypedRateLimitingInterface[cache.ObjectName]
	recorder           record.EventRecorder
}

func main() {
	var clusterString string
	flag.StringVar(&clusterString, "clusters", "EMPTY", "specify the names of the clusters and their kubeconfig file in a colon-pair comma seperated format, e.g. -clusters='name1:config,name2:config' ")
	flag.Parse()

	// so now that we can get all the kubeconfig files, we have to build each client seperately...
	// idk if trying to build the same client twice will break the program... guess we'll see!
	controllers := make(map[string]Controller)
	var clusters []Cluster
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
		}

		clusters = append(clusters, Cluster{
			clusterName: clusterConfig.clusterName,
			configPath:  clusterConfig.configPath,
			client:      kclient,
		})
	}

	for index, cluster := range clusters {
		pod, err := cluster.client.CoreV1().Pods("default").Get(context.TODO(), "test", metav1.GetOptions{})
		if err != nil {
			fmt.Println("Error occured while getting pod with client #"+string(index)+"! Error:", err.Error())
		}

		fmt.Printf("Client %s: %s\n", cluster.clusterName, pod.Name)
	}
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
		clusterName: config.clusterName,
		configPath:  config.configPath,
		client:      clientset,
		// kInformerFactory: informerFactory,
		deploymentInformer: informerFactory.Apps().V1().Deployments(),
		workqueue:          workqueue.NewTypedRateLimitingQueue(ratelimiter),
		recorder:           recorder,
	}

	message := fmt.Sprintf("Setting up event handler for controller %s", config.clusterName)
	klog.Info(message)

	// need to make the method for this thing -- HERE
	controller.deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{})

	return controller
}

// So next we have to start the informer factories which we can do in new controller
// Then we have to actually make the method to start the controller

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	logger := klog.FromContext(ctx)

	logger.Info("Starting controller and workers!")

	go wait.UntilWithContext(ctx, c.runWorker, time.Second)

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Sutting down workers")

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	objRef, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		logger.Info("Queue signaled shutdown!")
		return false
	}

	defer c.workqueue.Done(objRef)

	err := c.syncHandler(ctx, objRef)

	if err == nil {
		c.workqueue.Forget(objRef)
		logger.Info("Successfully synced", "object reference", objRef)
		return true
	}

	// yeah, if we get an error, we'll just retry
	utilruntime.HandleErrorWithContext(ctx, err, "Error syncing; requeuing for later retry", "objectReference", objRef)

	// I don't know if this will forget an object after it's been retried after a certain amount of requeues
	// I guess we'll see (we can delibretally fail an object)
	c.workqueue.AddRateLimited(objRef)
	return true
}

func (c *Controller) syncHandler(ctx context.Context, objref cache.ObjectName) error {
	return nil
}
