package controllers

import (
	"fmt"
	v1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
)

func (m *MongoClusterService) getServices() *[]v1api.Service {
	var service v1api.Service
	var services []v1api.Service

	getOne := func(name string) (*v1api.Service, error) {
		if err := m.Reconciler.Client.Get(
			*m.Context,
			types.NamespacedName{Name: name, Namespace: m.Namespace},
			&service); err != nil {
			return nil, err
		}
		return &service, nil
	}

	for i := 0; int32(i) < m.AppConfig.Spec.Replicas; i++ {
		serviceName := getResourceGenericName(m.AppConfig.Name, fmt.Sprintf("%d", i))
		service, err := getOne(serviceName)
		if err != nil {
			m.Logger.Info(fmt.Sprintf("Service %s does not exist yet.", serviceName))
		} else {
			services = append(services, *service)
		}
	}
	return &services
}

func (m *MongoClusterService) createOrUpdateService(serviceName string) error {
	actualService := &v1api.Service{}
	expectedService := m.createService(serviceName, serviceName)
	err := m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: expectedService.Name, Namespace: m.Namespace}, actualService)
	serviceUpToDate := !reflect.DeepEqual((*expectedService).Spec, (*actualService).Spec) && m.serviceExists(*expectedService)

	if err != nil && !errors.IsNotFound(err) {
		m.Logger.Error(err, "Error getting service")
		return err
	} else if errors.IsNotFound(err) {
		m.Logger.Info(fmt.Sprintf("Creating service %s", expectedService.Name))
		err = m.Reconciler.Client.Create(*m.Context, expectedService)
		if err != nil {
			m.Logger.Error(err, "Error creating service")
			return err
		}
	} else if serviceUpToDate {
		m.Logger.Info(fmt.Sprintf("Service %s is up to date. Nothing to do.", serviceName))
		return nil
	} else {
		m.Logger.Info(fmt.Sprintf("Updating service %s", expectedService.Name))
		actualService.Spec = expectedService.Spec
		err = m.Reconciler.Client.Update(*m.Context, actualService)
		if err != nil {
			m.Logger.Error(err, "Error updating service")
			return err
		}
	}
	m.updateStack(*expectedService)
	return nil
}

func (m *MongoClusterService) createService(serviceName string, appLabelName string) *v1api.Service {
	return &v1api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: m.Namespace,
		},
		Spec: v1api.ServiceSpec{
			Type:       v1api.ServiceTypeLoadBalancer,
			ClusterIP:  "",
			ClusterIPs: nil,
			Ports: []v1api.ServicePort{
				{
					Port:       MONGO_CONTAINER_PORT,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: MONGO_CONTAINER_PORT},
					Protocol:   v1api.ProtocolTCP,
					NodePort:   getRandomPort(),
				},
			},
			Selector: map[string]string{
				"app": appLabelName,
			},
		},
	}
}
func (m *MongoClusterService) deleteServices(services []v1api.Service) error {
	for _, service := range services {
		if !reflect.DeepEqual(service, v1api.Service{}) || !m.serviceExists(service) {
			break
		}
		err := m.Reconciler.Client.Delete(*m.Context, &service)
		if err != nil {
			m.Logger.Error(err, "Error deleting service")
			return err
		}
	}
	return nil
}

func (m *MongoClusterService) serviceExists(service v1api.Service) bool {
	err := m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: service.Name, Namespace: m.Namespace}, &service)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		logger.Error(err, "Error getting service")
		return false
	}
	return true
}
