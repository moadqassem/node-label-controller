# Container Linux Node Labeler 

### Description

_Container Linux Node Labeler is used in order to add a label to the kubernetes cluster nodes which run Container Linux
an operating system_

### Design

The Container Linux Node Labeler contains of two main components, `engine` and `linux_controller`.

The role of the `engine` is to register the controller(s) and to start/stop them at any stage during the life cycle of 
labeler. The `engine` has a tracking channel for any error which occurs during the processing of the controllers.

The `linux_controller` is responsible for the listing all the cluster nodes, and detect which one is running Container 
Linux as an operating system, check the node labels, add the label and update the node. 

The `linux_controller` comes in two modes, the first mode(default) is the Watcher mode where a watcher is attached to the node interface 
and watches the status of the nodes and update them accordingly. The second mode, uses a ticker loop which runs every second 
to track the status of the nodes and update them accordingly.

### Requirements

In order to run the prebuilt docker image directly without making any changes, the only requirement is a Kubernetes Cluster.

For changing the controller and a custom business logic:

- An installed `Golang 1.11+` SDK. To install `Golang` please check out  this [link](https://golang.org/doc/install). 
This projects uses go modules so no need for configuring `GOPATH`.

- An installed `docker` to build and push the new image. To install `docker` please check out this [link](https://docs.docker.com/install/).

### Usage

**Running the prebuilt image directly:**

From the project root directory, run `make install`

**Running a customized build:**

- After all the requirements are installed, customize the controller and the config file as it's wished.

- Run make `docker-push` to build the new image and deploy it or `make image` ro build the image only.

**Running the project from source code:**

There is a possibility of running the labeler from the source code directly on the dev machine, by doing the following:

- Acquire the `kubeconfig` file in order to connect and authenticate to Kubernetes cluster.

- Update the config file in the config lib and add the path of `kubeconfig` file to the property `kube_config_path`.

- After updating `kube_config_path` the engine will identify that, this is a remote machine(not part of the k8s cluster)
and will run a different clientset creation method. To start the controller run: `make run` 

- After running the labeler run `kubectl describe node [container-linux-node-name]` and observe the new added label.

**Running the second mode(the ticker mode):**

The watcher mode is the default mode, however if the user chooses to use the second mode, which works as a scheduled process,
in the config file(map) change the `watcher: true` to `watcher: false`

**Validating Container Linux Node Labeler:** 

In order to validate that the labeler has worked correctly, deploy the `container-linux-update-operator.yaml` in the deploy
directory by running `kubectl apply -f deploy/container-linux-update-operator.yaml` to deploy the operator and 
`kubectl apply -f deploy/update-agent.yaml` to deploy the agent.