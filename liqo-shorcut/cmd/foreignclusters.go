package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/liqotech/liqo/pkg/liqoctl/factory"
	"github.com/liqotech/liqo/apis/discovery/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var foreignClustersCmd = &cobra.Command{
	Use:   "foreignclusters",
	Short: "Stampa un saluto",
	Run: func(cmd *cobra.Command, args []string) error {
		return listForeignClusters()
	},
}

func init() {
	rootCmd.AddCommand(foreignClustersCmd)
}

func listForeignClusters() error {
	// Creiamo una factory per il client del cluster locale
	f, err := factory.NewForLocal()
	if err != nil {
		return fmt.Errorf("errore nella creazione della factory: %w", err)
	}

	// Otteniamo un client Kubernetes dal factory
	k8sClient, err := f.NewControllerRuntimeClient()
	if err != nil {
		return fmt.Errorf("errore nella creazione del client: %w", err)
	}

	// Otteniamo la lista di ForeignCluster
	var fcList v1.ForeignClusterList
	if err := k8sClient.List(context.TODO(), &fcList, &client.ListOptions{}); err != nil {
		return fmt.Errorf("errore nel recupero dei ForeignClusters: %w", err)
	}

	// Verifica se ci sono ForeignClusters e stampa i risultati
	if len(fcList.Items) == 0 {
		fmt.Println("Nessun ForeignCluster trovato.")
		return nil
	}

	// Stampa i ForeignClusters
	fmt.Println("ForeignClusters trovati:")
	for _, fc := range fcList.Items {
		fmt.Printf("- %s (ClusterID: %s)\n", fc.Name, fc.Spec.ClusterIdentity.ClusterID)
	}

	return nil
}