# kubernetes-blue-green-deployment-controller

This repository features a Kubernetes controller designed to consistently manage two ReplicaSets blue and green while alternating between these two colors for new rollouts.

This repository was created based on https://github.com/kubernetes/sample-controller.

# How to run Locally
<mark>
Install the latest version of golang first. https://go.dev/doc/install
Install and setup the kubernetes in you system.
</mark>

### Pulling the code
```bash
# clone the code
$ git clone https://github.com/k9982874/kubernetes-blue-green-deployment-controller.git bgd
```

### Change work directory to `bgd` folder
```bash
$ cd bgd
```

### Compile the executable binary
```bash
$ go build -o bgd-controller .
```

### Run bgd-controller in another terminal
```bash
$ ./bgd-controller -kubeconfig=$HOME/.kube/config

# output
# There are some errors in the output because the BlueGreenDeployment CRD was not found. That's OK.
W1216 10:05:47.421877   75544 reflector.go:569] pkg/mod/k8s.io/client-go@v0.0.0-20241214015128-61ee2c5802c7/tools/cache/reflector.go:251: failed to list *v1alpha1.BlueGreenDeployment: the server could not find the requested resource (get bluegreendeployments.controller.k8s.kexinlife.com)
...
```

### Create BlueGreenDeployment CRD
```bash
$ kubectl create -f artifacts/examples/crd-status-subresource.yaml
```

### Create PODs
```bash
$ kubectl create -f artifacts/examples/example-foo.yaml
```

# Check the status of pods and services
The service bgd-svc is running.
Two pods has been created, one for the working pod and one for the backup pod.
Also, there are two apps blue-rs and green-rs, one of them is activated.
```bash
$ kubectl get all
NAME                 READY   STATUS    RESTARTS   AGE
pod/blue-rs-xnkhh    1/1     Running   0          36s
pod/green-rs-cpvh4   0/1     Pending   0          30s

NAME                 TYPE        CLUSTER-IP        EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   192.168.194.129   <none>        443/TCP   41h
service/bgd-svc      ClusterIP   192.168.194.194   <none>        80/TCP    31s

NAME                       DESIRED   CURRENT   READY   AGE
replicaset.apps/blue-rs    1         1         1       36s
replicaset.apps/green-rs   1         1         0       30s
```

# Change the activate service group
Edit the configuration with below command.
```bash
$ kubectl edit bluegreendeployment blue-green-deployment
```
After a few seconds, the current working pod is changed to `green-rs-ggfbt` and the `green-rs` app enters the ready state.
```
$ kubectl get all
NAME                 READY   STATUS    RESTARTS   AGE
pod/blue-rs-nt92w    0/1     Pending   0          13s
pod/green-rs-ggfbt   1/1     Running   0          44s

NAME                 TYPE        CLUSTER-IP        EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   192.168.194.129   <none>        443/TCP   42h
service/bgd-svc      ClusterIP   192.168.194.189   <none>        80/TCP    82s

NAME                       DESIRED   CURRENT   READY   AGE
replicaset.apps/blue-rs    1         1         0       13s
replicaset.apps/green-rs   1         1         1       44s
```

# How to cleanup
By default, terminating the blue-green deployment controller won't automatically remove the resources it created. These resources will remain and be picked up again if the controller is restarted.

### Manual Cleanup Options
If you want to remove the resources after stopping the controller, you can use the following commands.

Delete the Custom Resource Definition (CRD):
```bash
$ kubectl create -f artifacts/examples/crd-status-subresource.yaml
```
This command deletes the CRD, all associated bluegreendeployment custom resources, and any ReplicaSets they manage.

Delete Individual Blue-Green Deployments (Optional):
```bash
$ kubectl delete bluegreendeployment blue-green-deployment 
```
This command deletes the specified bluegreendeployment resource and its associated ReplicaSets. You can use this command if you only want to remove specific deployments.

For now, the bgd-svc service needs to be deleted manually:
```bash
kubectl delete service bgd-svc
```