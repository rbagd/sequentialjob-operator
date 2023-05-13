/*
Copyright 2023.

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
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	// appsv1 "k8s.io/apimachineary/pkg/api/apps/v1"

	operatorv1alpha1 "github.com/rbagd/sequentialjob-operator/api/v1alpha1"
)

// SequentialJobReconciler reconciles a SequentialJob object
type SequentialJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.rbagd.eu,resources=sequentialjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.rbagd.eu,resources=sequentialjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.rbagd.eu,resources=sequentialjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SequentialJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SequentialJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	operatorCR := &operatorv1alpha1.SequentialJob{}
	err := r.Get(ctx, req.NamespacedName, operatorCR)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Operator resource object not found.")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Error getting operator resource object")
		return ctrl.Result{}, err
	}

	idx := 0
	for idx, _ = range operatorCR.Spec.Jobs {

		jobItem := &batchv1.Job{}
		err = r.Get(
			ctx,
			types.NamespacedName{
				Namespace: req.Namespace,
				Name:      fmt.Sprintf("%s-%d", req.Name, idx),
			},
			jobItem,
		)

		if err == nil && jobItem.Status.CompletionTime == nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}

		if err != nil && errors.IsNotFound(err) {
			logger.Info("Operator resource object not found.")
			break
		} else if err != nil {
			logger.Error(err, "Error getting operator resource object")
			return ctrl.Result{}, err
		}

	}

	jobStep := batchv1.Job{

		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", req.Name, idx),
			Namespace: req.Namespace,
		},
		Spec: operatorCR.Spec.Jobs[idx],
	}

	err = r.Create(ctx, &jobStep)

	if idx < len(operatorCR.Spec.Jobs)-1 {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SequentialJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.SequentialJob{}).
		Complete(r)
}
