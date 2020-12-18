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

package v1

import (
	"errors"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clusterversionlog = logf.Log.WithName("clusterversion-resource")

func (r *ClusterVersion) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-multicluster-ops-io-v1-clusterversion,mutating=true,failurePolicy=fail,groups=multicluster-ops.io,resources=clusterversions,verbs=create;update,versions=v1,name=mclusterversion.kb.io

var _ webhook.Defaulter = &ClusterVersion{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClusterVersion) Default() {
	clusterversionlog.Info("default", "name", r.Name)
	if r.Spec.RequiredAvailableCount <= 0 {
		r.Spec.RequiredAvailableCount = 1
	}
}

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-multicluster-ops-io-v1-clusterversion,mutating=false,failurePolicy=fail,groups=multicluster-ops.io,resources=clusterversions,versions=v1,name=vclusterversion.kb.io

var _ webhook.Validator = &ClusterVersion{}

func (r *ClusterVersion) validateDuplicate() *field.Error {
	path := field.NewPath("spec").Child("clusters")
	for _, c1 := range r.Spec.Clusters {
		duplicated := 0
		for _, c2 := range r.Spec.Clusters {
			if c1.ID == c2.ID {
				duplicated += 1
			}
		}
		if duplicated > 1 {
			return field.Invalid(path, c1.ID, "duplicate cluster id")
		}
	}
	return nil
}

func (r *ClusterVersion) validateClusters() error {
	errList := field.ErrorList{}
	if err := r.validateDuplicate(); err != nil {
		errList = append(errList, err)
		return apierr.NewInvalid(schema.GroupKind{
			Group: "multicluster-ops.io",
			Kind:  "ClusterVersion",
		}, r.Name, errList)
	}
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterVersion) ValidateCreate() error {
	clusterversionlog.Info("validate create", "name", r.Name)
	return r.validateClusters()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterVersion) ValidateUpdate(old runtime.Object) error {
	clusterversionlog.Info("validate update", "name", r.Name)
	return r.validateClusters()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterVersion) ValidateDelete() error {
	clusterversionlog.Info("validate delete", "name", r.Name)
	if r.Status.OperationID != "" {
		return errors.New("mustn't delete while operation is running")
	}
	return nil
}
