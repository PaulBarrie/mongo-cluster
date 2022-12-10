/*
Copyright 2022.

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

package v1beta1

import (
	"context"
	v1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	DEFAULT_REPLICAS_NUMBER    = 1
	DEFAULT_STORAGE_SIZE       = "1Gi"
	DEFAULT_STORAGE_CLASS      = "local-path"
	DEFAULT_CPU_LIMIT          = "1000m"
	DEFAULT_CPU_REQUEST        = "100m"
	DEFAULT_MEMORY_LIMIT       = "1Gi"
	DEFAULT_MEMORY_REQUEST     = "256Mi"
	DEFAULT_DATABASE           = "mongo"
	DEFAULT_STORAGE_CLASS_NAME = "standard"
)

// log is for logging in this package.
var mongoclusterlog = logf.Log.WithName("mongocluster-resource")
var _manager ctrl.Manager

func (r *MongoCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	//return ctrl.NewWebhookManagedBy(mgr).
	//	For(r).
	//	Complete()
	err := ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
	_manager = mgr
	return err
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-apps-esgi-fr-v1beta1-mongocluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.esgi.fr,resources=mongoclusters,verbs=create;update,versions=v1beta1,name=mmongocluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &MongoCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MongoCluster) Default() {
	mongoclusterlog.Info("default", "name", r.Name)
	if r.Spec.Replicas < 1 {
		mongoclusterlog.Info("No replicas specified, defaulting to %s", DEFAULT_REPLICAS_NUMBER)
		r.Spec.Replicas = DEFAULT_REPLICAS_NUMBER
	}
	if r.Spec.Storage.Size == "" {
		mongoclusterlog.Info("No storage size specified, defaulting to %s", DEFAULT_STORAGE_SIZE)
		r.Spec.Storage.Size = DEFAULT_STORAGE_SIZE
	}
	if r.Spec.Storage.StorageClassName == "" {
		mongoclusterlog.Info("No storage class name specified, defaulting to %s", DEFAULT_STORAGE_CLASS_NAME)
		r.Spec.Storage.StorageClassName = DEFAULT_STORAGE_CLASS
	}
	if r.Spec.Resources.CPU.Request == "" {
		mongoclusterlog.Info("No Request request specified, defaulting to %s", DEFAULT_CPU_REQUEST)
		r.Spec.Resources.CPU.Request = DEFAULT_CPU_REQUEST
	}
	if r.Spec.Resources.CPU.Limit == "" {
		mongoclusterlog.Info("No CPU request specified, defaulting to %s", DEFAULT_CPU_LIMIT)
		r.Spec.Resources.CPU.Limit = DEFAULT_CPU_LIMIT
	}
	if r.Spec.Resources.Memory.Request == "" {
		mongoclusterlog.Info("No memory request specified, defaulting to %s", DEFAULT_MEMORY_REQUEST)
		r.Spec.Resources.Memory.Request = DEFAULT_CPU_LIMIT
	}

	if r.Spec.Resources.Memory.Limit == "" {
		mongoclusterlog.Info("No memory limit specified, defaulting to %s", DEFAULT_MEMORY_LIMIT)
		r.Spec.Resources.Memory.Limit = DEFAULT_MEMORY_LIMIT
	}
	if r.Spec.DatabaseName == "" {
		mongoclusterlog.Info("No database specified, defaulting to %s", DEFAULT_DATABASE)
		r.Spec.DatabaseName = DEFAULT_DATABASE
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-apps-esgi-fr-v1beta1-mongocluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.esgi.fr,resources=mongoclusters,verbs=create;update,versions=v1beta1,name=vmongocluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MongoCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoCluster) ValidateCreate() error {
	mongoclusterlog.Info("validate create", "name", r.Name)

	err := r.validatePasswordSecret()
	if err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MongoCluster) ValidateUpdate(old runtime.Object) error {
	mongoclusterlog.Info("validate update", "name", r.Name)
	err := r.validatePasswordSecret()
	if err != nil {
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MongoCluster) ValidateDelete() error {
	mongoclusterlog.Info("validate delete", "name", r.Name)
	err := r.validatePasswordSecret()
	if err != nil {
		return err
	}
	return nil
}

func (r *MongoCluster) validatePasswordSecret() error {
	if r.Spec.Auth.ExistingSecretName == "" && r.Spec.Auth.Password == "" {
		return error(errors.NewNotFound(v1api.Resource("secret"), r.Spec.Auth.ExistingSecretName))
	} else if r.Spec.Auth.ExistingSecretName != "" && r.Spec.Auth.Password == "" {
		cli := _manager.GetClient()
		secret := &v1api.Secret{}
		ctx := context.Background()
		err := cli.Get(ctx, types.NamespacedName{
			Name:      r.Spec.Auth.ExistingSecretName,
			Namespace: r.Namespace,
		}, secret)
		if err != nil {
			return err
		}
	}
	return nil
}
