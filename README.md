# Foreign Cluster Connector

**Foreign Cluster Connector** is a Kubebuilder-based Liqo feature that runs in the **central Liqo cluster** and manages a direct network peering between two Liqo-peered **leaf clusters**. Integrated into Liqo’s control-plane, it leverages Liqo’s networking stack to optimize east–west traffic and maintain a seamless multi-cluster environment.

---

## Description

This proof-of-concept controller watches a `VirtualNodeConnection` CR in the central cluster and:

- **On Create**: establishes a direct Liqo tunnel between two specified leaf clusters.  
- **On Delete**: gracefully tears down only that tunnel, preserving all other Liqo network configurations.

**Note:** Must run inside the central Liqo control-plane. Requires Liqo components on all participating clusters.

---

## Features

- **Native Liqo Integration**: Adds a `VirtualNodeConnection` CRD and controller logic to Liqo.  
- **Automated Tenant Namespaces**: Creates `liqo-tenant-<clusterID>` namespaces to isolate per-connection resources.  
- **Selective Teardown**: Uses finalizers to disconnect a single peering without impacting others.  
- **Declarative Workflow**: Entire lifecycle managed by applying or deleting one CR.

---

## Advantages & Benefits

- **Optimized Multi-Cluster Traffic**: Leaf clusters communicate directly, avoiding double-hop through the central control-plane.  
- **Scoped Impact**: Only the targeted peering link is modified, preserving all other Liqo-managed tunnels.  
- **Kubernetes-Native**: Fully declarative via Liqo CRDs and finalizers—fits seamlessly into GitOps workflows.  
- **Centralized Awareness**: Central cluster gains visibility into direct leaf-to-leaf connections without altering global routing.  
- **Future-Proof Foundation**: Lays groundwork for conditional IP propagation and dynamic routing optimizations.  
- **Automation & Simplification**: Eliminates manual tracking of cluster-to-cluster network shortcuts.  
- **Improved Network Coherence**: Ensures consistent CIDR remapping and avoids configuration drift.

---

## Next Steps & Future Use Cases

Building on this, future enhancements could include:

- **Conditional IP Propagation**: IPAM and virtual-kubelet can choose direct IP propagation when a `VirtualNodeConnection` exists, instead of indirect paths through the central cluster.  
- **Dynamic Routing Optimization**: Allow Liqo’s routing logic to prefer direct leaf-to-leaf tunnels over indirect overlays, further reducing latency and load.

---


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

