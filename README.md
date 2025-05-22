# Foreign Cluster Connector

**Foreign Cluster Connector** is a Kubebuilder-based Liqo feature that runs in the **central Liqo cluster** and manages a direct network connection between two **foreign clusters**. 
---

## Description

This proof-of-concept controller watches a `ForeignClusterConnection` CR in the central cluster and:

- **On Create**: establishes a direct Liqo tunnel between two specified leaf clusters.  
- **On Delete**: gracefully tears down only that tunnel, preserving all other Liqo network configurations.

---

## Features

- **Native Liqo Integration**: Adds a `ForeignClusterConnection` CRD and controller logic to Liqo.  
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

- **Conditional IP Propagation**: IPAM and virtual-kubelet can choose direct IP propagation when a `ForeignClusterConnection` exists, instead of indirect paths through the central cluster.  
- **Dynamic Routing Optimization**: Allow Liqo’s routing logic to prefer direct leaf-to-leaf tunnels over indirect overlays, further reducing latency and load.

---


## Getting Started

> **Note:**  
> This controller has been tested with the `replicated-deployment` example from [Liqo](https://github.com/liqotech/liqo) using **Cilium** as the CNI, since `liqoctl disconnect/reset` commands had known issues with the default Kindnet. We recommend testing it in the same context to verify its functionality.

### Prerequisites

- Go **v1.23.0+**
- Docker **17.03+**
- `kubectl` **v1.11.3+**
- Access to a Kubernetes cluster **v1.11.3+**

---

### Deploying to the Cluster



#### 1. **Install the CRDs into the cluster**

```sh
make install
```

#### 2. **Apply permissions in the leaf clusters**

Apply the `clusterrole.yaml` file **in each leaf cluster** to grant the controller the necessary permissions to create the required components.
>
> ```bash
> kubectl apply -f clusterrole.yaml \
>   --kubeconfig /path/to/foreign/kubeconfig
> ```
>
> ⚠️ This is a temporary setup intended for testing purposes only. A proper ServiceAccount with the necessary ClusterRole and ClusterRoleBinding should be configured for production use.
>⚠️ If a non-default setup is used, ensure that the `subjects` field in the `ClusterRoleBinding` is updated to reference the correct ServiceAccount, User, or Group of the main cluster.


#### 3. **Deploy the controller manager in the central cluster**

```sh
make deploy
```

---

### Using the Controller

#### ✅ Create a CR in the central cluster to establish the connection

```sh
kubectl apply -f shortcutExample.yaml
```

#### ❌ Delete the CR to remove the connection

```sh
kubectl delete fcc europe-rome-edge-europe-milan-edge
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

