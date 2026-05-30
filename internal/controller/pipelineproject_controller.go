/*
Copyright 2026.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
)

// PipelineProjectReconciler reconciles a PipelineProject object: it owns a
// child PipelineCluster CR and a NATS JetStream StatefulSet (with Service +
// ConfigMap) in the same namespace. Phase 1 provisions infrastructure only;
// routes are accepted but inert.
type PipelineProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// NATSImage overrides the default NATS server image. Wired via the chart
	// (features.projects.nats.image+tag) and passed in from main.go.
	NATSImage string
}

// +kubebuilder:rbac:groups=rpc.operator.io,resources=pipelineprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rpc.operator.io,resources=pipelineprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rpc.operator.io,resources=pipelineprojects/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;delete

// Reconcile drives a PipelineProject towards its desired state.
func (r *PipelineProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var project rpcv1alpha1.PipelineProject
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Step 1: Child PipelineCluster CR.
	cluster := &rpcv1alpha1.PipelineCluster{ObjectMeta: metav1.ObjectMeta{
		Name: projectChildClusterName(project.Name), Namespace: project.Namespace,
	}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, cluster, func() error {
		built := buildProjectCluster(&project)
		cluster.Labels = built.Labels
		cluster.Spec = built.Spec
		return controllerutil.SetControllerReference(&project, cluster, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("apply pipelinecluster: %w", err)
	}

	// Step 2: NATS ConfigMap.
	natsReplicas := projectNATSReplicas(&project)
	natsStorage := projectNATSStorage(&project)

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: projectChildNATSName(project.Name), Namespace: project.Namespace,
	}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		built := buildProjectNATSConfigMap(project.Name, natsReplicas)
		cm.Labels = built.Labels
		cm.Data = built.Data
		return controllerutil.SetControllerReference(&project, cm, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("apply nats configmap: %w", err)
	}

	// Step 3: NATS headless Service.
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: projectChildNATSName(project.Name), Namespace: project.Namespace,
	}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		built := buildProjectNATSService(project.Name)
		svc.Labels = built.Labels
		svc.Spec.ClusterIP = built.Spec.ClusterIP
		svc.Spec.Selector = built.Spec.Selector
		svc.Spec.Ports = built.Spec.Ports
		return controllerutil.SetControllerReference(&project, svc, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("apply nats service: %w", err)
	}

	// Step 4: NATS StatefulSet.
	ss := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name: projectChildNATSName(project.Name), Namespace: project.Namespace,
	}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ss, func() error {
		built := buildProjectNATSStatefulSet(project.Name, r.NATSImage, natsReplicas, natsStorage)
		ss.Labels = built.Labels
		// Selector + ServiceName + VolumeClaimTemplates are immutable after creation.
		if ss.CreationTimestamp.IsZero() {
			ss.Spec.Selector = built.Spec.Selector
			ss.Spec.ServiceName = built.Spec.ServiceName
			ss.Spec.VolumeClaimTemplates = built.Spec.VolumeClaimTemplates
		}
		ss.Spec.Replicas = built.Spec.Replicas
		ss.Spec.Template = built.Spec.Template
		return controllerutil.SetControllerReference(&project, ss, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("apply nats statefulset: %w", err)
	}

	// Status: derive phase and child readiness from the children we just (re-)applied.
	clusterChild := rpcv1alpha1.ProjectChildStatus{
		Name:  cluster.Name,
		Ready: cluster.Status.ReadyReplicas,
		Total: cluster.Spec.Replicas,
	}
	natsChild := rpcv1alpha1.ProjectChildStatus{
		Name:  ss.Name,
		Ready: ss.Status.ReadyReplicas,
		Total: natsReplicas,
	}

	phase := deriveProjectPhase(clusterChild, natsChild)

	desiredCond := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "Provisioning",
		Message: fmt.Sprintf("cluster %d/%d, nats %d/%d", clusterChild.Ready, clusterChild.Total, natsChild.Ready, natsChild.Total),
	}
	if phase == rpcv1alpha1.ProjectPhaseReady {
		desiredCond.Status = metav1.ConditionTrue
		desiredCond.Reason = "AllReady"
	} else if phase == rpcv1alpha1.ProjectPhaseDegraded {
		desiredCond.Reason = "ChildDegraded"
	}

	existingCond := apimeta.FindStatusCondition(project.Status.Conditions, "Ready")
	condChanged := existingCond == nil ||
		existingCond.Status != desiredCond.Status ||
		existingCond.Reason != desiredCond.Reason ||
		existingCond.Message != desiredCond.Message

	if condChanged || project.Status.Phase != phase ||
		project.Status.Cluster != clusterChild ||
		project.Status.NATS != natsChild ||
		project.Status.ObservedGeneration != project.Generation {
		project.Status.Phase = phase
		project.Status.Cluster = clusterChild
		project.Status.NATS = natsChild
		project.Status.ObservedGeneration = project.Generation
		apimeta.SetStatusCondition(&project.Status.Conditions, desiredCond)
		if err := r.Status().Update(ctx, &project); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// projectNATSReplicas returns the requested NATS replica count, defaulting to 1.
func projectNATSReplicas(p *rpcv1alpha1.PipelineProject) int32 {
	if p.Spec.NATS == nil || p.Spec.NATS.Replicas == nil {
		return 1
	}
	return *p.Spec.NATS.Replicas
}

// projectNATSStorage returns the requested NATS PVC size, defaulting to natsStorageDefault.
func projectNATSStorage(p *rpcv1alpha1.PipelineProject) resource.Quantity {
	if p.Spec.NATS == nil || p.Spec.NATS.Storage == nil {
		return natsStorageDefault
	}
	return *p.Spec.NATS.Storage
}

// deriveProjectPhase returns the project's phase given the two child states.
// Provisioning: at least one child not yet at desired ready count.
// Ready:       both children fully ready.
// Degraded:    intentionally unused in Phase 1 (no health checks beyond
//              StatefulSet ready replicas yet); Phase 2 will use it when
//              NATS stream provisioning starts emitting failure events.
func deriveProjectPhase(cluster, nats rpcv1alpha1.ProjectChildStatus) rpcv1alpha1.PipelineProjectPhase {
	if cluster.Total > 0 && cluster.Ready >= cluster.Total &&
		nats.Total > 0 && nats.Ready >= nats.Total {
		return rpcv1alpha1.ProjectPhaseReady
	}
	return rpcv1alpha1.ProjectPhaseProvisioning
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rpcv1alpha1.PipelineProject{}).
		Owns(&rpcv1alpha1.PipelineCluster{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Named("pipelineproject").
		Complete(r)
}
