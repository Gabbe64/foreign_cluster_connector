package cmd

import (
	"context"
	"fmt"
	"os"

	liqov1beta1 "github.com/liqotech/liqo/apis/core/v1beta1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var foreignClustersCmd = &cobra.Command{
	Use:   "foreignclusters",
	Short: "List all ForeignClusters in the current Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listForeignClusters(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(foreignClustersCmd)
}

func listForeignClusters() error {
	ctx := context.Background()

	// Recupera la configurazione K8s
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	// Registra lo schema Liqo
	scheme := runtime.NewScheme()
	if err := liqov1beta1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Liqo schema: %w", err)
	}

	// Crea il client
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	// Lista i ForeignClusters
	var fcList liqov1beta1.ForeignClusterList
	if err := cl.List(ctx, &fcList); err != nil {
		return fmt.Errorf("unable to list ForeignClusters: %w", err)
	}

	if len(fcList.Items) == 0 {
		fmt.Println("No ForeignClusters found.")
		return nil
	}

	// Stampa i risultati
	for _, fc := range fcList.Items {
		fmt.Printf("- Name: %s\n  ClusterID: %s\n  Namespace: %s\n",
			fc.Name, fc.Spec.ClusterIdentity.ClusterID, fc.Namespace)
	}

	return nil
}
