package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/nilo19/kube-timer/pkg/timer/svc"
	"github.com/nilo19/kube-timer/pkg/tools"
	"github.com/spf13/cobra"
)

var (
	definitionFile      string
	startedEventReason  string
	finishedEventReason string
	svcName             string
	svcNamespace        string
	count               int
	asyncCreate         bool
	deleteSvc           bool
	deleteAll           bool
)

// svcCmd represents the svc command
var svcCmd = &cobra.Command{
	Use:   "svc",
	Short: "Get the provision/deletion times of LoadBalancer typed services.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		mode := tools.ServiceTimerModeCreate
		if deleteSvc {
			mode = tools.ServiceTimerModeDelete
		} else if deleteAll {
			mode = tools.ServiceTimerModeDeleteAll
		} else if asyncCreate {
			mode = tools.ServiceTimerModeCreateAsync
		}

		kubeClient, err := tools.MustBuildKubeClient()
		if err != nil {
			return errors.New("error building kube client")
		}

		dynamicClient, err := tools.MustBuildDynamicClient()
		if err != nil {
			return errors.New("error building dynamic client")
		}

		svcTimer := svc.NewServiceTimer(
			kubeClient,
			dynamicClient,
			definitionFile,
			svcName,
			svcNamespace,
			startedEventReason,
			finishedEventReason,
			count,
			mode,
			debug)
		if err := svcTimer.Validate(); err != nil {
			log.Fatalf("Error validating service timer: %v", err)
		}

		return svcTimer.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(svcCmd)

	svcCmd.Flags().StringVarP(&definitionFile, "file", "f", "", "Service definition file")

	svcCmd.Flags().IntVarP(&count, "count", "c", 1,
		"Number of services to create, need to specify `metadata.generateName` in the definition file")

	svcCmd.Flags().BoolVarP(&asyncCreate, "async", "a", false,
		"Create multiple services at a time and wait for all to be created. This requires the cloud provider to support creating multiple services asynchronously")

	svcCmd.Flags().StringVar(&startedEventReason, "started-event-reason", "", "event.Reason for the service start provisioning event")

	svcCmd.Flags().StringVar(&finishedEventReason, "finished-event-reason", "", "event.Reason for the service finish provisioning event")

	svcCmd.Flags().BoolVarP(&deleteSvc, "delete", "d", false, "Delete the provided services")

	svcCmd.Flags().BoolVarP(&deleteAll, "delete-all", "D", false, "Delete all services")

	svcCmd.Flags().StringVarP(&svcName, "name", "n", "", "Name of the service to delete")

	svcCmd.Flags().StringVarP(&svcNamespace, "namespace", "", "default", "Namespace of the service to delete")
}
