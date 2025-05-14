/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	ipamv1alpha1 "github.com/liqotech/liqo/apis/ipam/v1alpha1"
	"github.com/liqotech/liqo/pkg/liqoctl/factory"
	"github.com/liqotech/liqo/pkg/liqoctl/network"
	"github.com/liqotech/liqo/pkg/liqoctl/output"
	networkingv1alpha1 "github.com/scal110/foreign_cluster_connector/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// ForeignClusterConnectionReconciler reconciles a ForeignClusterConnection object
type ForeignClusterConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.liqo.io,resources=foreignclusterconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.liqo.io,resources=foreignclusterconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.liqo.io,resources=foreignclusterconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile handles the creation and deletion of ForeignClusterConnection
func (r *ForeignClusterConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling ForeignClusterConnection", "namespace", req.Namespace, "name", req.Name)

	var connection networkingv1alpha1.ForeignClusterConnection
	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !connection.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("ForeignClusterConnection is being deleted, starting disconnection", "name", req.Name)
		if err := r.disconnectLiqoctl(ctx, &connection); err != nil {
			logger.Error(err, "Error during disconnection")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(&connection, finalizerName)
		if err := r.Update(ctx, &connection); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Finalizer removed, ForeignClusterConnection can be deleted", "name", req.Name)
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&connection, finalizerName) {
		logger.Info("Adding finalizer", "name", req.Name)
		controllerutil.AddFinalizer(&connection, finalizerName)
		if err := r.Update(ctx, &connection); err != nil {
			return ctrl.Result{}, err
		}
	}

	if connection.Status.IsConnected {
		logger.Info("Nodes already connected", "nodeA", connection.Spec.ForeignClusterA, "nodeB", connection.Spec.ForeignClusterB)
		return ctrl.Result{}, nil
	}

	if connection.Status.Phase == "" {
		connection.Status = networkingv1alpha1.ForeignClusterConnectionStatus{
			IsConnected:  false,
			LastUpdated:  time.Now().Format(time.RFC3339),
			Phase:        "Pending",
			ErrorMessage: "",
		}
		if err := r.Status().Update(ctx, &connection); err != nil {
			logger.Error(err, "Error initializing status")
			return ctrl.Result{}, err
		}
	}

	logger.Info("Starting connection", "nodeA", connection.Spec.ForeignClusterA, "nodeB", connection.Spec.ForeignClusterB)
	if err := r.updateStatus(ctx, &connection, "Connecting", ""); err != nil {
		return ctrl.Result{}, err
	}

	output, err := r.executeLiqoctlConnect(ctx, &connection)
	if err != nil {
		logger.Error(err, "Error during liqoctl connect execution", "output", output)
		_ = r.updateStatus(ctx, &connection, "Failed", fmt.Sprintf("Error: %v, Output: %s", err, output))
		return ctrl.Result{}, err
	}

	logger.Info("Connection succeeded", "nodeA", connection.Spec.ForeignClusterA, "nodeB", connection.Spec.ForeignClusterB)
	if err := r.updateStatus(ctx, &connection, "Connected", ""); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ForeignClusterConnectionReconciler) executeLiqoctlConnect(ctx context.Context, connection *networkingv1alpha1.ForeignClusterConnection) (string, error) {
	kubeconfigA, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.ForeignClusterA)
	if err != nil {
		return "", fmt.Errorf("error retrieving kubeconfig for ForeignClusterA: %v", err)
	}
	defer os.Remove(kubeconfigA)

	kubeconfigB, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.ForeignClusterB)
	if err != nil {
		return "", fmt.Errorf("error retrieving kubeconfig for ForeignClusterB: %v", err)
	}
	defer os.Remove(kubeconfigB)

	timeout := 120 * time.Second
	if connection.Spec.Networking.TimeoutSeconds > 0 {
		timeout = time.Duration(connection.Spec.Networking.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	os.Setenv("KUBECONFIG", kubeconfigA)
	localFactory := factory.NewForLocal()
	if err := localFactory.Initialize(); err != nil {
		return "", fmt.Errorf("localFactory initialization error: %v", err)
	}

	os.Setenv("KUBECONFIG", kubeconfigB)
	remoteFactory := factory.NewForRemote()
	if err := remoteFactory.Initialize(); err != nil {
		return "", fmt.Errorf("remoteFactory initialization error: %v", err)
	}

	os.Setenv("KUBECONFIG", kubeconfigA)
	localFactory.Namespace = ""
	remoteFactory.Namespace = ""

	netCfg := connection.Spec.Networking
	opts := network.NewOptions(localFactory)
	opts.RemoteFactory = remoteFactory

	opts.ServerGatewayType = netCfg.ServerGatewayType
	opts.ServerTemplateName = netCfg.ServerTemplateName
	opts.ServerTemplateNamespace = netCfg.ServerTemplateNamespace
	opts.ServerServiceType.Set(netCfg.ServerServiceType)
	opts.ServerServicePort = netCfg.ServerServicePort

	opts.ClientGatewayType = netCfg.ClientGatewayType
	opts.ClientTemplateName = netCfg.ClientTemplateName
	opts.ClientTemplateNamespace = netCfg.ClientTemplateNamespace

	opts.MTU = int(netCfg.MTU)
	opts.DisableSharingKeys = false
	opts.Timeout = timeout
	opts.Wait = netCfg.Wait

	localFactory.Printer = output.NewLocalPrinter(true, true)
	remoteFactory.Printer = output.NewRemotePrinter(true, true)

	fmt.Println("Executing 'network connect'...")
	if err := opts.RunConnect(ctx); err != nil {
		return "", fmt.Errorf("error during 'network connect': %v", err)
	}

	if err := r.populateCIDRsFromNetworkConfig(ctx, connection, *localFactory, *remoteFactory); err != nil {
		return "", fmt.Errorf("unable to load CIDRs: %v", err)
	}

	return "Operation 'network connect' completed successfully.", nil
}

func (r *ForeignClusterConnectionReconciler) populateCIDRsFromNetworkConfig(
	ctx context.Context,
	connection *networkingv1alpha1.ForeignClusterConnection,
	localFactory factory.Factory,
	remoteFactory factory.Factory,
) error {

	update := connection.DeepCopy()

	cidrA, err := r.retrieveCIDRInfoFromFactory(ctx, localFactory, connection.Spec.ForeignClusterB)
	if err != nil {
		return fmt.Errorf("error retrieving CIDR from cluster A: %w", err)
	}

	cidrB, err := r.retrieveCIDRInfoFromFactory(ctx, remoteFactory, connection.Spec.ForeignClusterA)
	if err != nil {
		return fmt.Errorf("error retrieving CIDR from cluster B: %w", err)
	}

	update.Status.ForeignClusterANetworking = cidrA
	update.Status.ForeignClusterBNetworking = cidrB

	patch := client.MergeFrom(connection)
	if err := r.Status().Patch(ctx, update, patch); err != nil {
		return fmt.Errorf("error updating status with CIDRs: %w", err)
	}

	return nil
}

func (r *ForeignClusterConnectionReconciler) retrieveCIDRInfoFromFactory(
	ctx context.Context,
	factory factory.Factory,
	remoteClusterName string,
) (networkingv1alpha1.ClusterNetworkingStatus, error) {

	var result networkingv1alpha1.ClusterNetworkingStatus
	tenantNs := fmt.Sprintf("liqo-tenant-%s", remoteClusterName)
	name := fmt.Sprintf("%s-pod", remoteClusterName)

	c, err := client.New(factory.RESTConfig, client.Options{Scheme: r.Scheme})
	if err != nil {
		return result, fmt.Errorf("error creating client from factory: %w", err)
	}

	var netCfg ipamv1alpha1.Network
	if err := c.Get(ctx, client.ObjectKey{Namespace: tenantNs, Name: name}, &netCfg); err != nil {
		return result, fmt.Errorf("error retrieving Network CR in namespace %q: %w", tenantNs, err)
	}

	result.PodCIDR = string(netCfg.Spec.CIDR)
	result.RemappedPodCIDR = string(netCfg.Status.CIDR)
	return result, nil
}

func (r *ForeignClusterConnectionReconciler) getKubeconfigFromLiqo(ctx context.Context, ForeignCluster string) (string, error) {
	namespace := fmt.Sprintf("liqo-tenant-%s", ForeignCluster)
	secretName := fmt.Sprintf("kubeconfig-controlplane-%s", ForeignCluster)

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, &secret); err != nil {
		return "", fmt.Errorf("Error retrieving Secret %s in namespace %s: %v", secretName, namespace, err)
	}

	kubeconfigData, exists := secret.Data["kubeconfig"]
	if !exists {
		return "", fmt.Errorf("Secret %s does not contain 'kubeconfig' key", secretName)
	}

	config, err := clientcmd.Load(kubeconfigData)
	if err != nil {
		return "", fmt.Errorf("Error parsing kubeconfig: %v", err)
	}

	if config.CurrentContext == "" {
		return "", fmt.Errorf("Kubeconfig has no current context set")
	}

	config.Contexts[config.CurrentContext].Namespace = ""

	modifiedData, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("Error marshaling modified kubeconfig: %v", err)
	}

	kubeconfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("kubeconfig-%s.yaml", ForeignCluster))
	if err := os.WriteFile(kubeconfigPath, modifiedData, 0600); err != nil {
		return "", fmt.Errorf("Error writing kubeconfig file: %v", err)
	}

	return kubeconfigPath, nil
}

func (r *ForeignClusterConnectionReconciler) updateStatus(ctx context.Context, connection *networkingv1alpha1.ForeignClusterConnection, phase, errorMsg string) error {
	patch := client.MergeFrom(connection.DeepCopy())

	connection.Status.Phase = phase
	connection.Status.LastUpdated = time.Now().Format(time.RFC3339)
	connection.Status.ErrorMessage = errorMsg
	connection.Status.IsConnected = (phase == "Connected")

	if err := r.Status().Patch(ctx, connection, patch); err != nil {
		log.FromContext(ctx).Error(err, "Error updating status")
		return err
	}
	return nil
}

func (r *ForeignClusterConnectionReconciler) disconnectLiqoctl(ctx context.Context, connection *networkingv1alpha1.ForeignClusterConnection) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting disconnection", "name", connection.Name)

	kubeconfigA, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.ForeignClusterA)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfigA)

	kubeconfigB, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.ForeignClusterB)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfigB)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	os.Setenv("KUBECONFIG", kubeconfigA)
	localFactory := factory.NewForLocal()
	if err := localFactory.Initialize(); err != nil {
		return fmt.Errorf("error initializing localFactory: %v", err)
	}

	os.Setenv("KUBECONFIG", kubeconfigB)
	remoteFactory := factory.NewForRemote()
	if err := remoteFactory.Initialize(); err != nil {
		return fmt.Errorf("error initializing remoteFactory: %v", err)
	}

	localFactory.Namespace = fmt.Sprintf("liqo-tenant-%s", connection.Spec.ForeignClusterB)
	remoteFactory.Namespace = fmt.Sprintf("liqo-tenant-%s", connection.Spec.ForeignClusterA)

	opts := network.NewOptions(localFactory)
	opts.RemoteFactory = remoteFactory
	opts.Timeout = 120 * time.Second
	opts.Wait = true
	localFactory.Printer = output.NewLocalPrinter(true, true)
	remoteFactory.Printer = output.NewRemotePrinter(true, true)

	fmt.Println("Executing 'network reset'...")
	if err := opts.RunReset(ctx); err != nil {
		return fmt.Errorf("error during 'network reset': %v", err)
	}

	fmt.Println("Operation 'network reset' completed successfully.")
	return nil
}

func (r *ForeignClusterConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.ForeignClusterConnection{}).
		Named("ForeignClusterConnection").
		Complete(r)
}
