package cmd

import (
	"context"
	"fmt"
	"os"

	liqov1beta1 "github.com/liqotech/liqo/apis/core/v1beta1"
	"github.com/liqotech/liqo/pkg/consts"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---------------------------
// LIST FOREIGNCLUSTERS
// ---------------------------
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

func listForeignClusters() error {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := liqov1beta1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Liqo schema: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	var fcList liqov1beta1.ForeignClusterList
	if err := cl.List(ctx, &fcList); err != nil {
		return fmt.Errorf("unable to list ForeignClusters: %w", err)
	}

	if len(fcList.Items) == 0 {
		fmt.Println("No ForeignClusters found.")
		return nil
	}

	for _, fc := range fcList.Items {
		fmt.Printf("- Name: %s\n  ClusterID: %s\n  Namespace: %s\n",
			fc.Name, fc.Spec.ClusterID, fc.Namespace)
	}

	return nil
}

// ---------------------------
// CHECK IF A FOREIGNCLUSTER EXISTS
// ---------------------------
var clusterID string

var checkForeignClusterCmd = &cobra.Command{
	Use:   "foreigncluster-exists",
	Short: "Check if a ForeignCluster with a given ClusterID exists",
	Run: func(cmd *cobra.Command, args []string) {
		exists, name, err := foreignClusterExists(clusterID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if exists {
			fmt.Printf("✅ ForeignCluster with ClusterID '%s' exists (Name: %s).\n", clusterID, name)
		} else {
			fmt.Printf("❌ ForeignCluster with ClusterID '%s' does NOT exist.\n", clusterID)
		}
	},
}

func foreignClusterExists(clusterID string) (bool, string, error) {
	ctx := context.Background()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return false, "", fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := liqov1beta1.AddToScheme(scheme); err != nil {
		return false, "", fmt.Errorf("unable to add Liqo schema: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return false, "", fmt.Errorf("unable to create client: %w", err)
	}

	labelSelector := labels.SelectorFromSet(labels.Set{
		consts.RemoteClusterID: clusterID,
	})

	var fcList liqov1beta1.ForeignClusterList
	if err := cl.List(ctx, &fcList, &client.ListOptions{
		LabelSelector: labelSelector,
	}); err != nil {
		return false, "", fmt.Errorf("unable to list ForeignClusters: %w", err)
	}

	if len(fcList.Items) > 0 {
		return true, fcList.Items[0].Name, nil
	}
	return false, "", nil
}

// ---------------------------
// INIT
// ---------------------------
func init() {
	rootCmd.AddCommand(foreignClustersCmd)

	checkForeignClusterCmd.Flags().StringVar(&clusterID, "id", "", "ClusterID of the ForeignCluster to check")
	checkForeignClusterCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(checkForeignClusterCmd)
}
