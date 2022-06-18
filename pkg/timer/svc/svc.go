package svc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/nilo19/kube-timer/pkg/tools"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlserializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/util/wait"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	to "k8s.io/utils/pointer"
)

// ServiceTimer is the main struct for the service timer
type ServiceTimer struct {
	DefinitionFile      string
	SvcName             string
	SvcNamespace        string
	StartedEventReason  string
	FinishedEventReason string
	Count               int
	KubeClient          kubernetes.Interface
	DynamicClient       dynamic.Interface
	Mode                tools.ServiceTimerMode

	processingServices map[string]*bool
	finishTimes        chan tools.ObjectFinishTime
	svcNameToTimesMap  map[string]*tools.ObjectFinishTime
	signal             chan struct{}
}

// NewServiceTimer creates a new ServiceTimer
func NewServiceTimer(kubeClient kubernetes.Interface,
	dynamicClient dynamic.Interface,
	definitionFile, svcName, svcNamespace, startedEventReason, finishedEventReason string,
	count int, mode tools.ServiceTimerMode) *ServiceTimer {

	return &ServiceTimer{
		DefinitionFile:      definitionFile,
		SvcName:             svcName,
		SvcNamespace:        svcNamespace,
		StartedEventReason:  startedEventReason,
		FinishedEventReason: finishedEventReason,
		Count:               count,
		KubeClient:          kubeClient,
		DynamicClient:       dynamicClient,
		Mode:                mode,
		processingServices:  make(map[string]*bool),
		finishTimes:         make(chan tools.ObjectFinishTime, count),
		svcNameToTimesMap:   make(map[string]*tools.ObjectFinishTime),
		signal:              make(chan struct{}),
	}
}

// Validate validates the configurations of the service timer
func (st *ServiceTimer) Validate() error {
	if strings.EqualFold(string(st.Mode), string(tools.ServiceTimerModeCreate)) ||
		strings.EqualFold(string(st.Mode), string(tools.ServiceTimerModeCreateAsync)) {
		if st.DefinitionFile == "" {
			return errors.New("definition file is required")
		}
	}

	if strings.EqualFold(string(st.Mode), string(tools.ServiceTimerModeDelete)) && st.SvcName == "" {
		return errors.New("service name is required for delete mode")
	}

	if strings.EqualFold(string(st.Mode), string(tools.ServiceTimerModeDelete)) ||
		strings.EqualFold(string(st.Mode), string(tools.ServiceTimerModeDeleteAll)) {
		if st.StartedEventReason == "" || st.FinishedEventReason == "" {
			return errors.New("started and finished event reasons are required")
		}
	}

	if err := st.validateCount(); err != nil {
		return err
	}

	if err := st.validateMsg(); err != nil {
		return err
	}

	return nil
}

func (st *ServiceTimer) validateCount() error {
	if st.Count <= 0 {
		return fmt.Errorf("provided count %d is invalid", st.Count)
	}

	if st.Count > 1 {
		svcBytes, err := ioutil.ReadFile(st.DefinitionFile)
		if err != nil {
			return fmt.Errorf("error reading service definition file: %w", err)
		}
		if !strings.Contains(string(svcBytes), "generateName") {
			return errors.New("metadata.GenerateName is required for creating multiple services")
		}
	}

	return nil
}

func (st *ServiceTimer) validateMsg() error {
	if st.StartedEventReason != "" && st.FinishedEventReason == "" {
		return errors.New("finished event reason is required")
	}

	if st.StartedEventReason == "" && st.FinishedEventReason != "" {
		return errors.New("started event reason is required")
	}

	return nil
}

// Start starts the service timer
func (st *ServiceTimer) Start(ctx context.Context) error {
	switch st.Mode {
	case tools.ServiceTimerModeCreate:
		if err := st.handleCreate(ctx, false); err != nil {
			return err
		}
	case tools.ServiceTimerModeCreateAsync:
		if err := st.handleCreate(ctx, true); err != nil {
			return err
		}
	case tools.ServiceTimerModeDelete:
		if err := st.handleDelete(ctx, false); err != nil {
			return err
		}
	case tools.ServiceTimerModeDeleteAll:
		if err := st.handleDelete(ctx, true); err != nil {
			return err
		}
	}

	return nil
}

func (st *ServiceTimer) handleCreate(ctx context.Context, isAsync bool) error {
	if st.StartedEventReason != "" {
		st.setupAndRunEventsInformer(st.StartedEventReason, st.FinishedEventReason)
	} else {
		st.setupAndRunServicesInformer()
	}

	log.Printf("Starting %d services creation", st.Count)
	creationTimes := make([]tools.ObjectFinishTime, st.Count)
	for i := 0; i < st.Count; i++ {
		svc, err := st.createServiceFromFile(ctx)
		if err != nil {
			return err
		}
		st.processingServices[svc.Name] = to.BoolPtr(false)
		log.Printf("Created service %s", svc.Name)

		if !isAsync {
			creationTimes[i] = <-st.finishTimes
		}
	}

	if isAsync {
		go func() {
			for i := 0; i < st.Count; i++ {
				creationTimes[i] = <-st.finishTimes
			}
			close(st.finishTimes)
			close(st.signal)
		}()
		<-st.signal
	}

	dumpServiceCreationResults(creationTimes)
	return nil
}

func dumpServiceCreationResults(creationTimes []tools.ObjectFinishTime) {
	tools.Sort(creationTimes)
	maxName, maxTime := tools.GetMaxTime(creationTimes)
	minName, minTime := tools.GetMinTime(creationTimes)
	avgTime := tools.GetAvgTime(creationTimes)
	medTime := tools.GetMedianTime(creationTimes)
	totTime := tools.GetTotalTime(creationTimes)
	log.Printf("Finished creating %d services, max provision time: %s (%s), min provision time: %s (%s), avg provision time: %s, medium provision time: %s, total provision time: %s",
		len(creationTimes),
		maxTime, maxName,
		minTime, minName,
		avgTime,
		medTime,
		totTime)
}

func dumpServiceDeletionResults(deletionTimes []tools.ObjectFinishTime) {
	tools.Sort(deletionTimes)
	maxName, maxTime := tools.GetMaxTime(deletionTimes)
	minName, minTime := tools.GetMinTime(deletionTimes)
	avgTime := tools.GetAvgTime(deletionTimes)
	medTime := tools.GetMedianTime(deletionTimes)
	totTime := tools.GetTotalTime(deletionTimes)
	log.Printf("Finished deleting %d services, max deletion time: %s (%s), min deletion time: %s (%s), avg deletion time: %s, medium deletion time: %s, total deletion time: %s",
		len(deletionTimes),
		maxTime, maxName,
		minTime, minName,
		avgTime,
		medTime,
		totTime)
}

func (st *ServiceTimer) handleDelete(ctx context.Context, isDeleteAll bool) error {
	st.setupAndRunEventsInformer(st.StartedEventReason, st.FinishedEventReason)

	var deletionTimes []tools.ObjectFinishTime
	if !isDeleteAll {
		deletionTimes = make([]tools.ObjectFinishTime, 0)

		log.Printf("Deleting service %s", st.SvcName)
		st.processingServices[st.SvcName] = to.BoolPtr(false)
		if err := st.KubeClient.CoreV1().Services(st.SvcNamespace).Delete(ctx, st.SvcName, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("error deleting service %s: %w", st.SvcName, err)
		}
		deletionTimes = append(deletionTimes, <-st.finishTimes)
	} else {
		allSvc, err := st.KubeClient.CoreV1().Services(st.SvcNamespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("error listing services: %w", err)
		}
		allLBSvc := make([]string, 0)
		for _, svc := range allSvc.Items {
			if svc.Spec.Type == v1.ServiceTypeLoadBalancer {
				allLBSvc = append(allLBSvc, svc.Name)
			}
		}
		log.Printf("Deleting %d services", len(allLBSvc))
		for _, lbSvc := range allLBSvc {
			deletionTimes = make([]tools.ObjectFinishTime, len(allLBSvc))

			log.Printf("Deleting service %s", lbSvc)
			st.processingServices[lbSvc] = to.BoolPtr(false)
			if err := st.KubeClient.CoreV1().Services(st.SvcNamespace).Delete(ctx, lbSvc, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("error deleting service %s: %w", lbSvc, err)
			}
		}

		go func() {
			for i := 0; i < len(allLBSvc); i++ {
				deletionTimes[i] = <-st.finishTimes
			}
			close(st.finishTimes)
			close(st.signal)
		}()
		<-st.signal
	}

	dumpServiceDeletionResults(deletionTimes)

	return nil
}

func (st *ServiceTimer) createServiceFromFile(ctx context.Context) (*v1.Service, error) {
	svcBytes, err := ioutil.ReadFile(st.DefinitionFile)
	if err != nil {
		return nil, fmt.Errorf("error reading service definition file: %w", err)
	}

	var rawObj runtime.RawExtension
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(svcBytes), 100)
	if err := decoder.Decode(&rawObj); err != nil {
		return nil, fmt.Errorf("error decoding service definition to runtime.RawExtension: %w", err)
	}
	obj, _, err := yamlserializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error serializing service definition: %w", err)
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("error converting service definition to unstructured: %w", err)
	}
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	}
	unstructuredService, err := st.DynamicClient.
		//Resource(mapping.Resource).
		Resource(gvr).
		Namespace(unstructuredObj.GetNamespace()).
		Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service: %w", err)
	}

	var svc *v1.Service
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredService.UnstructuredContent(), &svc); err != nil {
		return nil, fmt.Errorf("error converting unstructured service to v1.Service: %w", err)
	}

	return svc, nil
}

func (st *ServiceTimer) setupAndRunServicesInformer() {
	factory := informers.NewSharedInformerFactory(st.KubeClient, 0)
	svcInformer := factory.Core().V1().Services()
	svcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSvc, ok1 := oldObj.(*v1.Service)
			newSvc, ok2 := newObj.(*v1.Service)
			if ok1 && ok2 {
				if (len(oldSvc.Status.LoadBalancer.Ingress) == 0 ||
					oldSvc.Status.LoadBalancer.Ingress[0].IP == "") &&
					(len(newSvc.Status.LoadBalancer.Ingress) > 0 &&
						newSvc.Status.LoadBalancer.Ingress[0].IP != "") {
					t0 := newSvc.ObjectMeta.CreationTimestamp.Time
					t1 := time.Now()
					t := t1.Sub(t0)
					log.Printf("Service %s is ready with external IP %s in %s",
						newSvc.Name, newSvc.Status.LoadBalancer.Ingress[0].IP, t.String())
					st.finishTimes <- tools.ObjectFinishTime{
						Name:     newSvc.Name,
						Started:  t0,
						Finished: t1,
					}
				}
			}
		},
	})
	factory.Start(wait.NeverStop)
}

func (st *ServiceTimer) setupAndRunEventsInformer(started, finished string) {
	factory := informers.NewSharedInformerFactory(st.KubeClient, 0)
	eventsInformer := factory.Core().V1().Events()
	eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event, ok := obj.(*v1.Event)
			if ok && st.processingServices[event.InvolvedObject.Name] != nil && !*st.processingServices[event.InvolvedObject.Name] {
				if strings.EqualFold(event.Reason, started) {
					log.Printf("got started event: %+v", event)
					if st.svcNameToTimesMap[event.InvolvedObject.Name] == nil {
						st.svcNameToTimesMap[event.InvolvedObject.Name] = &tools.ObjectFinishTime{
							Name:    event.InvolvedObject.Name,
							Started: event.CreationTimestamp.Time,
						}
					}
				} else if strings.EqualFold(event.Reason, finished) {
					log.Printf("got finished event: %+v", event)
					st.svcNameToTimesMap[event.InvolvedObject.Name].Finished = event.CreationTimestamp.Time
					st.finishTimes <- *st.svcNameToTimesMap[event.InvolvedObject.Name]
					st.processingServices[event.InvolvedObject.Name] = to.BoolPtr(true)
					log.Printf("Finished in %s for service %s",
						st.svcNameToTimesMap[event.InvolvedObject.Name].Finished.Sub(st.svcNameToTimesMap[event.InvolvedObject.Name].Started).String(),
						event.InvolvedObject.Name)
				}
			}
		},
	})
	factory.Start(wait.NeverStop)
}
