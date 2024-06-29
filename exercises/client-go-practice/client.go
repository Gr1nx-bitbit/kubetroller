package main

import (
	"context"
	"flag"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	pointer "k8s.io/utils/pointer"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "../../config/config", "kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	myPod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:  "custom-pod-container",
					Image: "nginx",
				},
			},
		},
	}

	testPod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), myPod, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Sprintf("Successfully create %q pod!", testPod.Name)

	// podSpec := apiv1.PodTemplateSpec{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Labels: map[string]string{
	// 			"app": "test",
	// 		},
	// 	},
	// 	Spec: apiv1.PodSpec{
	// 		Containers: []apiv1.Container{
	// 			{
	// 				Name:  "deployment-pod",
	// 				Image: "nginx",
	// 			},
	// 		},
	// 	},
	// }

	pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), "custom-pod", metav1.GetOptions{})
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

	result, err := createDeployment("test-deployment", "blue", 3, podSpec, clientset)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deployment %q created!\n", result.GetObjectMeta().GetName())

	// if err = deleteDeployment("test-deployment", "blue", clientset); err != nil {
	// 	panic(err)
	// }

	// fmt.Println("Deployment deleted!")

	// pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), "test", metav1.GetOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(pod.Name)
	// I wanna try creating resources at some point; we'll get there at some point
}

// currently this doesn't do any error checking i.e. it does not check if a deployment under the same name already exists
// or if the namespace is valid although I don't think the second problem actually matters with the right cluster parameters
func createDeployment(name string, ns string, replicas int, podSpec apiv1.PodTemplateSpec, client *kubernetes.Clientset) (*v1.Deployment, error) {
	reps := pointer.Int32(int32(replicas))

	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.DeploymentSpec{
			Replicas: reps,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: podSpec,
		},
	}

	result, err := client.AppsV1().Deployments(ns).Create(context.TODO(), deployment, metav1.CreateOptions{})

	return result, err
}

// currently, this function does not do any error checking or handling
func deleteDeployment(name string, ns string, client *kubernetes.Clientset) error {
	err := client.AppsV1().Deployments(ns).Delete(context.TODO(), name, *metav1.NewDeleteOptions(int64(5)))
	return err
}

func createNamespace(name string, client *kubernetes.Clientset) (*apiv1.Namespace, error) {
	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	ns, err := client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	return ns, err
}
