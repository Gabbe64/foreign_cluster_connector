## Description

This is an initial implementation of the feature.............., intended to test its behavior and provide a working proof of concept.

The controller must be deployed in the **central cluster**, where it will be responsible for creating a direct network connection between the two leaf clusters. It is triggered by the creation or deletion of a custom resource (CR) that contains all necessary metadata to establish the connection. The CR must be created within the central cluster.

# Controller

This project was created using **Kubebuilder**. Its purpose is to allow a central Kubernetes cluster — peered with two leaf clusters — to establish a direct connection between the leaf clusters. This connection optimizes network traffic and reduces the load on the central cluster.

#Advantages and Limitations


## Getting Started

> **Note:**  
> This controller has been tested with the `replicated-deployment` example from [Liqo](https://github.com/liqotech/liqo). We recommend testing it in the same context to verify its functionality.

### Prerequisites

- Go **v1.23.0+**
- Docker **17.03+**
- `kubectl` **v1.11.3+**
- Access to a Kubernetes cluster **v1.11.3+**

---

### Deploying to the Cluster

#### 1. **Build and push the controller image**

```sh
make docker-build IMG=vnc-controller:latest
```

#### 2. **Load the image into the cluster**

```sh
kind load docker-image vnc-controller:latest
```

#### 3. **Install the CRDs into the cluster**

```sh
make install
```

#### 4. **Apply permissions in the leaf clusters**

Apply the `clusterrole.yaml` file **in each leaf cluster** to grant the controller the necessary permissions to create the required components.

> ⚠️ This is a temporary setup intended for testing purposes only.

```sh
kubectl apply -f clusterrole.yaml
```

#### 5. **Deploy the controller manager in the central cluster**

```sh
make deploy
```

---

### Using the Controller

#### ✅ Create a CR in the central cluster to establish the connection

```sh
kubectl apply -f test-connection.yaml
```

#### ❌ Delete the CR to remove the connection

```sh
kubectl delete vnc test-connection
```

---

### Uninstallation

#### 🔻 Remove the CRDs from the cluster

```sh
make uninstall
```

#### 🔻 Undeploy the controller from the cluster

```sh
make undeploy
```

