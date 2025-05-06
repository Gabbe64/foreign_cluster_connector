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
	"os"
	"path/filepath"
	"time"

	"github.com/liqotech/liqo/pkg/liqo-controller-manager/networking/forge"
	"github.com/liqotech/liqo/pkg/liqoctl/factory"
	"github.com/liqotech/liqo/pkg/liqoctl/network"
	"github.com/liqotech/liqo/pkg/liqoctl/output"
	networkingv1alpha1 "github.com/scal110/foreign_cluster_connector/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VirtualNodeConnectionReconciler reconciles a VirtualNodeConnection object
type VirtualNodeConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.liqo.io,resources=virtualnodeconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.liqo.io,resources=virtualnodeconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.liqo.io,resources=virtualnodeconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VirtualNodeConnection object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile

const finalizerName = "virtualnodeconnection.finalizers.networking.liqo.io"

// Reconcile gestisce la creazione e la cancellazione delle VirtualNodeConnection
func (r *VirtualNodeConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling VirtualNodeConnection", "namespace", req.Namespace, "name", req.Name)

	var connection networkingv1alpha1.VirtualNodeConnection
	if err := r.Get(ctx, req.NamespacedName, &connection); err != nil {
		// Se la risorsa non viene trovata, non è necessario riconciliarla ulteriormente.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Se l'oggetto è in fase di eliminazione, esegui la disconnessione
	if !connection.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("VirtualNodeConnection è in fase di eliminazione, avvio disconnessione", "name", req.Name)
		if err := r.disconnectLiqoctl(ctx, &connection); err != nil {
			logger.Error(err, "Errore durante la disconnessione")
			return ctrl.Result{}, err
		}

		// Rimuove il finalizer per permettere l'eliminazione dell'oggetto
		controllerutil.RemoveFinalizer(&connection, finalizerName)
		if err := r.Update(ctx, &connection); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Finalizer rimosso, VirtualNodeConnection può essere eliminata", "name", req.Name)
		return ctrl.Result{}, nil
	}

	// Aggiunge il finalizer se non è già presente
	if !controllerutil.ContainsFinalizer(&connection, finalizerName) {
		logger.Info("Aggiungo il finalizer", "name", req.Name)
		controllerutil.AddFinalizer(&connection, finalizerName)
		if err := r.Update(ctx, &connection); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Se i nodi sono già connessi, non eseguire ulteriori azioni
	if connection.Status.IsConnected {
		logger.Info("Nodi già connessi", "nodeA", connection.Spec.VirtualNodeA, "nodeB", connection.Spec.VirtualNodeB)
		return ctrl.Result{}, nil
	}

	// Inizializza lo stato se non è stato impostato
	if connection.Status.Phase == "" {
		connection.Status = networkingv1alpha1.VirtualNodeConnectionStatus{
			IsConnected:  false,
			LastUpdated:  time.Now().Format(time.RFC3339),
			Phase:        "Pending",
			ErrorMessage: "",
		}
		if err := r.Status().Update(ctx, &connection); err != nil {
			logger.Error(err, "Errore nell'inizializzazione dello stato")
			return ctrl.Result{}, err
		}
	}

	logger.Info("Avvio connessione", "nodeA", connection.Spec.VirtualNodeA, "nodeB", connection.Spec.VirtualNodeB)
	if err := r.updateStatus(ctx, &connection, "Connecting", ""); err != nil {
		return ctrl.Result{}, err
	}

	// Esegue il comando liqoctl connect
	output, err := r.executeLiqoctlConnect(ctx, &connection)
	if err != nil {
		logger.Error(err, "Errore durante l'esecuzione di liqoctl connect", "output", output)
		_ = r.updateStatus(ctx, &connection, "Failed", fmt.Sprintf("Errore: %v, Output: %s", err, output))
		return ctrl.Result{}, err
	}

	logger.Info("Connessione riuscita", "nodeA", connection.Spec.VirtualNodeA, "nodeB", connection.Spec.VirtualNodeB)
	if err := r.updateStatus(ctx, &connection, "Connected", ""); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// executeLiqoctlConnect esegue il comando "liqoctl network connect" utilizzando i kubeconfig recuperati
func (r *VirtualNodeConnectionReconciler) executeLiqoctlConnect(ctx context.Context, connection *networkingv1alpha1.VirtualNodeConnection) (string, error) {
	// Recupera il kubeconfig per VirtualNodeA
	kubeconfigA, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.VirtualNodeA)
	if err != nil {
		return "", fmt.Errorf("errore nel recupero del kubeconfig per VirtualNodeA: %v", err)
	}
	defer os.Remove(kubeconfigA)

	// Recupera il kubeconfig per VirtualNodeB
	kubeconfigB, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.VirtualNodeB)
	if err != nil {
		return "", fmt.Errorf("errore nel recupero del kubeconfig per VirtualNodeB: %v", err)
	}
	defer os.Remove(kubeconfigB)

	// Crea un contesto con timeout
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Imposta la variabile d'ambiente KUBECONFIG e crea la factory per il cluster locale
	os.Setenv("KUBECONFIG", kubeconfigA)
	localFactory := factory.NewForLocal()
	if err := localFactory.Initialize(); err != nil {
		return "", fmt.Errorf("errore nell'inizializzazione della localFactory: %v", err)
	}

	// Imposta la variabile d'ambiente KUBECONFIG e crea la factory per il cluster remoto
	os.Setenv("KUBECONFIG", kubeconfigB)
	remoteFactory := factory.NewForRemote()
	if err := remoteFactory.Initialize(); err != nil {
		return "", fmt.Errorf("errore nell'inizializzazione della remoteFactory: %v", err)
	}

	os.Setenv("KUBECONFIG", kubeconfigA)
	localFactory.Namespace = ""
	remoteFactory.Namespace = ""
	// Crea le opzioni per il comando "network connect"
	opts := network.NewOptions(localFactory)
	opts.RemoteFactory = remoteFactory
	opts.ServerGatewayType = forge.DefaultGwServerType
	opts.ServerTemplateName = forge.DefaultGwServerTemplateName // Nome del template
	opts.ServerServiceType.Set("NodePort")
	opts.ServerTemplateNamespace = "liqo"
	opts.ServerServicePort = forge.DefaultGwServerPort
	// Imposta i parametri per il Gateway Client, se necessario:
	opts.ClientGatewayType = forge.DefaultGwClientType
	opts.ClientTemplateName = forge.DefaultGwClientTemplateName
	opts.ClientTemplateNamespace = "liqo"
	// Parametri comuni
	opts.MTU = forge.DefaultMTU
	opts.DisableSharingKeys = false
	// Timeout, wait, skip-validation e altri parametri possono essere impostati anch'essi:
	opts.Timeout = 120 * time.Second
	opts.Wait = true

	localFactory.Printer = output.NewLocalPrinter(true, true)
	remoteFactory.Printer = output.NewRemotePrinter(true, true)

	fmt.Println("Informazioni localFactory:")
	fmt.Printf("Namespace: %s\n", localFactory.Namespace)
	fmt.Printf("RESTConfig: %+v\n\n", localFactory.RESTConfig)

	fmt.Println("Informazioni remoteFactory:")
	fmt.Printf("Namespace: %s\n", remoteFactory.Namespace)
	fmt.Printf("RESTConfig: %+v\n\n", remoteFactory.RESTConfig)

	fmt.Println("Esecuzione del comando 'network connect'...")
	if err := opts.RunConnect(ctx); err != nil {
		return "", fmt.Errorf("errore durante l'esecuzione di 'network connect': %v", err)
	}

	fmt.Println("Operazione 'network connect' completata con successo.")
	return "Operazione 'network connect' completata con successo.", nil
}

// getKubeconfigFromLiqo recupera il kubeconfig dal Secret associato al virtual node,
// ne modifica il namespace nel contesto corrente e lo salva in un file temporaneo.
func (r *VirtualNodeConnectionReconciler) getKubeconfigFromLiqo(ctx context.Context, virtualNode string) (string, error) {
	// Costruisce namespace e nome del secret in base al virtual node.
	namespace := fmt.Sprintf("liqo-tenant-%s", virtualNode)
	secretName := fmt.Sprintf("kubeconfig-controlplane-%s", virtualNode)

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, &secret); err != nil {
		return "", fmt.Errorf("Errore nel recupero del Secret %s nel namespace %s: %v", secretName, namespace, err)
	}

	kubeconfigData, exists := secret.Data["kubeconfig"]
	if !exists {
		return "", fmt.Errorf("Il Secret %s non contiene la chiave 'kubeconfig'", secretName)
	}

	// Carica il kubeconfig YAML in memoria come oggetto Config.
	config, err := clientcmd.Load(kubeconfigData)
	if err != nil {
		return "", fmt.Errorf("Errore nel parsing del kubeconfig: %v", err)
	}

	// Verifica che sia impostato un contesto corrente.
	if config.CurrentContext == "" {
		return "", fmt.Errorf("Il kubeconfig non ha un contesto corrente impostato")
	}

	// Modifica il namespace del contesto corrente con quello preso dal secret.
	config.Contexts[config.CurrentContext].Namespace = ""

	// Scrive l'oggetto Config modificato in YAML.
	modifiedData, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("Errore nel marshalling del kubeconfig modificato: %v", err)
	}

	// Salva il kubeconfig modificato in un file temporaneo.
	kubeconfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("kubeconfig-%s.yaml", virtualNode))
	if err := os.WriteFile(kubeconfigPath, modifiedData, 0600); err != nil {
		return "", fmt.Errorf("Errore nella scrittura del file kubeconfig: %v", err)
	}

	return kubeconfigPath, nil
}

// updateStatus aggiorna lo stato dell'oggetto VirtualNodeConnection utilizzando una patch per evitare conflitti
func (r *VirtualNodeConnectionReconciler) updateStatus(ctx context.Context, connection *networkingv1alpha1.VirtualNodeConnection, phase, errorMsg string) error {
	patch := client.MergeFrom(connection.DeepCopy())

	connection.Status.Phase = phase
	connection.Status.LastUpdated = time.Now().Format(time.RFC3339)
	connection.Status.ErrorMessage = errorMsg
	connection.Status.IsConnected = (phase == "Connected")

	if err := r.Status().Patch(ctx, connection, patch); err != nil {
		log.FromContext(ctx).Error(err, "Errore nell'aggiornamento dello stato")
		return err
	}
	return nil
}

// disconnectLiqoctl esegue il comando "liqoctl network disconnect" per disconnettere i nodi
func (r *VirtualNodeConnectionReconciler) disconnectLiqoctl(ctx context.Context, connection *networkingv1alpha1.VirtualNodeConnection) error {
    logger := log.FromContext(ctx)
    logger.Info("Avvio disconnessione", "name", connection.Name)

    // Recupera i kubeconfig
    kubeconfigA, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.VirtualNodeA)
    if err != nil {
        return err
    }
    defer os.Remove(kubeconfigA)

    kubeconfigB, err := r.getKubeconfigFromLiqo(ctx, connection.Spec.VirtualNodeB)
    if err != nil {
        return err
    }
    defer os.Remove(kubeconfigB)

    // Timeout sul contesto
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Inizializza le factory
    os.Setenv("KUBECONFIG", kubeconfigA)
    localFactory := factory.NewForLocal()
    if err := localFactory.Initialize(); err != nil {
        return fmt.Errorf("errore nella inizializzazione della localFactory: %v", err)
    }

    os.Setenv("KUBECONFIG", kubeconfigB)
    remoteFactory := factory.NewForRemote()
    if err := remoteFactory.Initialize(); err != nil {
        return fmt.Errorf("errore nella inizializzazione della remoteFactory: %v", err)
    }

    // Crea i due cluster helper (senza toccare i namespace tenant)
    cluster1, err := network.NewCluster(ctx, localFactory, remoteFactory, false)
    if err != nil {
        return fmt.Errorf("errore nella creazione di cluster1 per disconnect: %v", err)
    }
    cluster2, err := network.NewCluster(ctx, remoteFactory, localFactory, false)
    if err != nil {
        return fmt.Errorf("errore nella creazione di cluster2 per disconnect: %v", err)
    }

    // Configura le opzioni
    opts := network.NewOptions(localFactory)
    opts.RemoteFactory = remoteFactory
    opts.Timeout = 120 * time.Second
    opts.Wait = true
    localFactory.Printer = output.NewLocalPrinter(true, true)
    remoteFactory.Printer = output.NewRemotePrinter(true, true)

    // Esegui il solo disconnect (toglie client/server ma lascia intatta la network-config)
    fmt.Println("Esecuzione del comando 'network disconnect'...")
    if err := opts.RunDisconnect(ctx, cluster1, cluster2); err != nil {
        return fmt.Errorf("errore durante l'esecuzione di 'network disconnect': %v", err)
    }

    fmt.Println("Operazione 'network disconnect' completata con successo.")
    return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualNodeConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.VirtualNodeConnection{}).
		Named("virtualnodeconnection").
		Complete(r)
}
