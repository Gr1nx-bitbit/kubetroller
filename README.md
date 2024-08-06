<h1>Instructions</h1>
<ul>
    <li>Fork the repo</li>
    <li>At the root level, create a directory called "config" and copy your kubernetes config file (~/.kube/config) into it</li>
    <li>cd to the that-conference-k8s-controller directory</li>
    <li>Run ```kubectl run --image=nginx test``` to make a pod in your cluster</li>
    <li>Run ```kubectl apply -f crd.yaml``` to register the CRD to the cluster</li>
    <li>In a **seperate** termianl run ```go run main.go``` to start the controller. It will fail if the CRD isn't applied to the cluster! If it is running, the controller will log "starting worker"</li>
    <li>Run ```kubectl apply -f promote.yaml``` to promote the pod from earlier into a deployment or run ```kubectl apply -f destroy.yaml``` to destroy the pod from earlier.</li>
</ul>