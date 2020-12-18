/*
Copyright 2020 taisho6339.

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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	opsv1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
	"github.com/taisho6339/multicluster-upgrade-operator/pkg/ops"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	reasonOperationFailed    = "OperationFailed"
	reasonClusterUnavailable = "ClusterUnavailable"
	defaultRequeueDuration   = time.Second * 60
)

type operationFunc func() (ctrl.Result, error)

// ClusterVersionReconciler reconciles a ClusterVersion object
type ClusterVersionReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Operator ops.Operator
}

// +kubebuilder:rbac:groups=multicluster-ops.io,resources=clusterversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multicluster-ops.io,resources=clusterversions/status,verbs=get;update;patch

func (r *ClusterVersionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	obj := &opsv1.ClusterVersion{}
	log := r.Log.WithValues("multicluster", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get multi cluster")
		return ctrl.Result{}, nil
	}
	// Actual Operations
	if obj.Status.OperationID != "" {
		return r.reconcileOperationStatus(ctx, obj, log)
	}
	return r.reconcileClusterVersion(ctx, obj, log)
}

func (r *ClusterVersionReconciler) reconcileOperationStatus(ctx context.Context, obj *opsv1.ClusterVersion, log logr.Logger) (ctrl.Result, error) {
	status, err := r.Operator.GetOperationStatus(ctx, *obj)
	if err != nil {
		log.Error(err, "failed to get operation status")
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
	}

	switch status {
	case ops.OperationStatusRunning:
		return ctrl.Result{}, nil
	case ops.OperationStatusDone:
		log.Info(fmt.Sprintf("(operation_id %s, operation_type %s) is done.", obj.Status.OperationID, obj.Status.OperationType))
		addSuccessOperation(obj.Status.OperationType)
		obj.Status.ResetStatus()
		return r.updateStatus(ctx, obj, log)
	case ops.OperationStatusFailed:
		opID := obj.Status.OperationID
		opType := obj.Status.OperationType
		// report as an error
		err := errors.New("operation failed")
		log.Error(err, fmt.Sprintf("operation_id %s failed. this operation type is %s", opID, opType))
		r.Recorder.Eventf(obj, corev1.EventTypeWarning, reasonOperationFailed, "operation_type: %s, operation_id: %s", opType, opID)
		addFailedOperation(obj.Status.OperationType)
		obj.Status.ResetStatus()
		return r.updateStatus(ctx, obj, log)
	case ops.OperationStatusUnknown:
		opID := obj.Status.OperationID
		opType := obj.Status.OperationType
		// report as an error
		err := errors.New("operation status is unknown")
		log.Error(err, fmt.Sprintf("operation_id %s, operation_type %s", opID, opType))
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ClusterVersionReconciler) reconcileClusterVersion(ctx context.Context, obj *opsv1.ClusterVersion, log logr.Logger) (ctrl.Result, error) {
	for _, cluster := range obj.Spec.Clusters {
		cv, err := r.Operator.GetClusterVersion(ctx, *obj, cluster)
		if err != nil {
			log.Error(err, "get cluster version")
			return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
		}
		if cv.Master.Version != cluster.Version {
			return r.reconcileMasterVersion(ctx, obj, cluster, log)
		}
		for _, pool := range cv.NodePools {
			if pool.Version != cluster.Version {
				return r.reconcileNodePoolVersion(ctx, obj, cluster, pool.NodePoolID, log)
			}
		}
		cs, err := r.Operator.GetClusterStatus(ctx, *obj, cluster)
		if err != nil {
			log.Error(err, "failed to get cluster status")
			return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
		}
		if !cs.Available {
			err := fmt.Errorf("cluster %s is unavailable", cluster.ID)
			log.Error(err, err.Error())
			r.Recorder.Event(obj, corev1.EventTypeWarning, reasonClusterUnavailable, err.Error())
			return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
		}
		if cs.Type == ops.ClusterStatusServiceOut {
			return r.serviceIn(ctx, obj, cluster, log)
		}
	}
	return ctrl.Result{}, nil
}

func (r *ClusterVersionReconciler) withServiceOut(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, log logr.Logger, op operationFunc) (ctrl.Result, error) {
	cs, err := r.Operator.GetClusterStatus(ctx, *obj, cluster)
	if err != nil {
		log.Error(err, "failed to get cluster status")
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
	}
	if cs.Type == ops.ClusterStatusServiceOut {
		return op()
	}
	if r.canServiceOut(ctx, obj, cluster) {
		return r.serviceOut(ctx, obj, cluster, log)
	}
	// report as an warning event
	msg := fmt.Sprintf("can't service out. currently available clusters less than required available count: %d", obj.Spec.RequiredAvailableCount)
	r.Recorder.Event(obj, corev1.EventTypeWarning, reasonClusterUnavailable, msg)
	return ctrl.Result{}, nil
}

func (r *ClusterVersionReconciler) canServiceOut(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster) bool {
	availableCount := 0
	for _, c := range obj.Spec.Clusters {
		if c.ID == cluster.ID {
			continue
		}
		cs, err := r.Operator.GetClusterStatus(ctx, *obj, c)
		if err == nil && cs.Available {
			availableCount += 1
		}
		if availableCount >= obj.Spec.RequiredAvailableCount {
			return true
		}
	}
	return false
}

func (r *ClusterVersionReconciler) reconcileMasterVersion(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, log logr.Logger) (ctrl.Result, error) {
	return r.withServiceOut(ctx, obj, cluster, log, func() (result ctrl.Result, err error) {
		return r.upgradeMaster(ctx, obj, cluster, log)
	})
}

func (r *ClusterVersionReconciler) reconcileNodePoolVersion(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, nodePoolID string, log logr.Logger) (ctrl.Result, error) {
	return r.withServiceOut(ctx, obj, cluster, log, func() (result ctrl.Result, err error) {
		return r.upgradeNodePool(ctx, obj, cluster, nodePoolID, log)
	})
}

func (r *ClusterVersionReconciler) serviceIn(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, log logr.Logger) (ctrl.Result, error) {
	result, err := r.Operator.ServiceIn(ctx, *obj, cluster)
	if err != nil {
		log.Error(err, "failed to service in")
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
	}
	obj.Status.ClusterID = cluster.ID
	obj.Status.OperationID = result.OperationID
	obj.Status.OperationType = result.OperationType
	return r.updateStatus(ctx, obj, log)
}

func (r *ClusterVersionReconciler) serviceOut(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, log logr.Logger) (ctrl.Result, error) {
	result, err := r.Operator.ServiceOut(ctx, *obj, cluster)
	if err != nil {
		log.Error(err, "failed to service out")
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
	}
	obj.Status.ClusterID = cluster.ID
	obj.Status.OperationID = result.OperationID
	obj.Status.OperationType = result.OperationType
	return r.updateStatus(ctx, obj, log)
}

func (r *ClusterVersionReconciler) upgradeMaster(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, log logr.Logger) (ctrl.Result, error) {
	result, err := r.Operator.UpgradeMaster(ctx, *obj, cluster)
	if err != nil {
		log.Error(err, "failed to upgrade master")
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
	}
	obj.Status.ClusterID = cluster.ID
	obj.Status.OperationID = result.OperationID
	obj.Status.OperationType = result.OperationType
	return r.updateStatus(ctx, obj, log)
}

func (r *ClusterVersionReconciler) upgradeNodePool(ctx context.Context, obj *opsv1.ClusterVersion, cluster opsv1.Cluster, nodePoolID string, log logr.Logger) (ctrl.Result, error) {
	result, err := r.Operator.UpgradeNodePool(ctx, *obj, cluster, nodePoolID)
	if err != nil {
		log.Error(err, "failed to upgrade node pool")
		return ctrl.Result{RequeueAfter: defaultRequeueDuration}, err
	}
	obj.Status.ClusterID = cluster.ID
	obj.Status.OperationID = result.OperationID
	obj.Status.OperationType = result.OperationType
	return r.updateStatus(ctx, obj, log)
}

func (r *ClusterVersionReconciler) updateStatus(ctx context.Context, obj *opsv1.ClusterVersion, log logr.Logger) (ctrl.Result, error) {
	if err := r.Client.Status().Update(ctx, obj); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opsv1.ClusterVersion{}).
		Complete(r)
}
