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

package controllers

import (
	"context"
	appsv1beta1 "github.com/PaulBarrie/mongo-cluster/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	finalizerName = "mongocluster.finalizers.esgi.fr"
)

// MongoClusterReconciler reconciles a MongoCluster object
type MongoClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var logger = logf.Log.WithName("controller_mongocluster")

//+kubebuilder:rbac:groups=apps.esgi.fr,resources=mongoclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.esgi.fr,resources=mongoclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.esgi.fr,resources=mongoclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=deployments;services;secrets;persistentvolumeclaims;configmaps,verbs=get;list;create;update;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MongoCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MongoClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var mongoCluster appsv1beta1.MongoCluster
	logger.WithValues("Namespace", req.NamespacedName)

	if err := r.Get(ctx, req.NamespacedName, &mongoCluster); err != nil {
		logger.Error(err, "unable to fetch MongoCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	mongoService := r.NewService(ctx, &mongoCluster, req.NamespacedName.Namespace)

	//https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html
	isUnderDeletion := !(mongoCluster.ObjectMeta.DeletionTimestamp.IsZero())
	thereIsFinalizer := controllerutil.ContainsFinalizer(&mongoCluster, finalizerName)
	if isUnderDeletion {
		if thereIsFinalizer {
			// Remove resources
			if err := mongoService.Delete(); err != nil {
				return ctrl.Result{}, err
			}
			// Remove finalizer
			controllerutil.RemoveFinalizer(&mongoCluster, finalizerName)
			if err := r.Update(ctx, &mongoCluster); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if !thereIsFinalizer {
			controllerutil.AddFinalizer(&mongoCluster, finalizerName)
			if err := r.Update(ctx, &mongoCluster); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	if err := mongoService.CreateOrUpdate(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta1.MongoCluster{}).
		Complete(r)
}
