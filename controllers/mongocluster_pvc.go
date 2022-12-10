package controllers

import (
	"fmt"
	v1 "k8s.io/api/apps/v1"
	v1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
)

func (m *MongoClusterService) getPersistentVolumeClaims() *[]v1api.PersistentVolumeClaim {
	var pvc v1api.PersistentVolumeClaim
	getOne := func(name string) (error, *v1api.PersistentVolumeClaim) {
		if err := m.Reconciler.Client.Get(
			*m.Context,
			types.NamespacedName{Name: name, Namespace: m.Namespace},
			&pvc); err != nil {
			m.Logger.Info(fmt.Sprintf("Error getting PVC %s", name))
			return errors.NewNotFound(v1.Resource("deployment"), ""), nil
		}
		return nil, &pvc
	}
	var pvcs []v1api.PersistentVolumeClaim

	for i := 0; int32(i) < m.AppConfig.Spec.Replicas; i++ {
		resourceName := getResourceGenericName(m.AppConfig.Name, fmt.Sprintf("%d", i))
		err, pvc := getOne(resourceName)
		if err != nil {
			m.Logger.Info(fmt.Sprintf("PVC %s does not exist", resourceName))
		} else {
			pvcs = append(pvcs, *pvc)
		}
	}
	return &pvcs
}

func (m *MongoClusterService) createOrUpdatePersistentVolumeClaim(pvcName string) error {
	actualPVC := &v1api.PersistentVolumeClaim{}
	expectedPVC, err := m.createPersistentVolumeClaim(pvcName, m.AppConfig.Spec.Storage.StorageClassName)
	if err != nil {
		m.Logger.Error(err, "Error creating PVC")
		return err
	}
	err = m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: pvcName, Namespace: m.Namespace}, actualPVC)
	pvcUpToDate := !reflect.DeepEqual((*expectedPVC).Spec, (*actualPVC).Spec) && m.pvcExists(*expectedPVC)

	if err != nil && !errors.IsNotFound(err) {
		m.Logger.Error(err, "Error getting pvc")
		return err
	} else if errors.IsNotFound(err) {
		m.Logger.Info(fmt.Sprintf("Creating pvc %s", expectedPVC.Name))
		err = m.Reconciler.Client.Create(*m.Context, expectedPVC)
		if err != nil {
			m.Logger.Error(err, "Error creating pvc")
			return err
		}
	} else if pvcUpToDate {
		m.Logger.Info(fmt.Sprintf("PVC %s is up to date. Nothing to do.", expectedPVC.Name))
		return nil
	} else {
		m.Logger.Info(fmt.Sprintf("Updating PVC %s", expectedPVC.Name))
		actualPVC.Spec = expectedPVC.Spec
		err = m.Reconciler.Client.Update(*m.Context, actualPVC)
		if err != nil {
			m.Logger.Error(err, "Error updating PVC")
			return err
		}
	}
	m.updateStack(*expectedPVC)
	return nil
}

func (m *MongoClusterService) createPersistentVolumeClaim(name string, storageClassName string) (*v1api.PersistentVolumeClaim, error) {
	quantity, err := resource.ParseQuantity(m.AppConfig.Spec.Storage.Size)
	if err != nil {
		m.Logger.Error(err, "Error parsing quantity")
		return nil, err
	}
	pvc := v1api.PersistentVolumeClaim{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      name,
			Namespace: m.Namespace,
		},
		Spec: v1api.PersistentVolumeClaimSpec{
			AccessModes: []v1api.PersistentVolumeAccessMode{v1api.ReadWriteOnce},
			Resources: v1api.ResourceRequirements{
				Requests: v1api.ResourceList{v1api.ResourceStorage: quantity},
				Limits:   v1api.ResourceList{v1api.ResourceStorage: quantity},
			},
			StorageClassName: &storageClassName,
		},
	}
	return &pvc, nil
}

func (m *MongoClusterService) deletePersistentVolumeClaims(persistentVolumeClaims []v1api.PersistentVolumeClaim) error {
	for _, persistentVolumeClaim := range persistentVolumeClaims {
		if !reflect.DeepEqual(persistentVolumeClaim, v1api.PersistentVolumeClaim{}) {
			break
		}
		err := m.Reconciler.Client.Delete(*m.Context, &persistentVolumeClaim)
		if err != nil {
			m.Logger.Error(err, "Error deleting persistent volume claim")
			return err
		}
	}
	return nil
}

func (m *MongoClusterService) pvcExists(pvc v1api.PersistentVolumeClaim) bool {
	err := m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: pvc.Name, Namespace: m.Namespace}, &pvc)
	if err != nil {
		return false
	}
	return true

}
