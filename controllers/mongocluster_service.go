package controllers

import (
	"context"
	"fmt"
	appsv1beta1 "github.com/PaulBarrie/mongo-cluster/api/v1beta1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	v1api "k8s.io/api/core/v1"
	"reflect"
	"strconv"
)

var mongoMaster string
var mongoGenericPodName string

const (
	MONGODB_DEFAULT_USER               = "admin"
	MONGODB_DEFAULT_PASSWORD           = "mongo_pwd"
	MONGODB_DEFAULT_ROLE               = "root"
	MONGO_DEPLOYMENT_REPLICAS          = 1
	MONGODB_DEFAULT_HOST               = "mongo"
	MONGO_CONTAINER_PORT         int32 = 27017
	MONGO_CONTAINER_NAME               = "mongo"
	MONGO_CONTAINER_IMAGE              = "paulb314/mongo:5.0.6"
	MONGO_KEY_VOLUME_NAME              = "mongo-key"
	MONGO_KEY_MOUNT_PATH               = "/etc/secrets-volume/"
	MONGO_STORAGE_VOLUME_NAME          = "mongo-persistent-storage"
	MONGO_STORAGE_MOUNT_PATH           = "/data"
	MONGO_CONFIGMAP_NAME               = "mongo-configmap"
	MONGO_RESOURCE_FORMAT              = "%s-mongo-%s"
	DEFAULT_PASSWORD_SECRET_NAME       = "mongo-password"
)

var (
	MONGO_KEY_SECRET_DEFAULT_MODE int32 = 0400
)

type MongoClusterService struct {
	AppConfig  *appsv1beta1.MongoCluster
	Namespace  string
	Reconciler *MongoClusterReconciler
	Context    *context.Context
	Stack      *MongoClusterStack
	Logger     logr.Logger
}

type MongoClusterStack struct {
	Deployments            *[]v1.Deployment
	Services               *[]v1api.Service
	PersistentVolumeClaims *[]v1api.PersistentVolumeClaim
	Secret                 *v1api.Secret
}

func (r *MongoClusterReconciler) NewService(context context.Context, appConfig *appsv1beta1.MongoCluster, namespace string) MongoClusterService {
	return MongoClusterService{
		AppConfig:  appConfig,
		Namespace:  namespace,
		Reconciler: r,
		Context:    &context,
		Logger:     logger,
		Stack: &MongoClusterStack{
			Deployments:            &[]v1.Deployment{},
			Services:               &[]v1api.Service{},
			PersistentVolumeClaims: &[]v1api.PersistentVolumeClaim{},
			Secret:                 &v1api.Secret{},
		},
	}
}

func (m *MongoClusterService) CreateOrUpdate() error {
	var err error
	mongoClusterStack, err := m.getStack()
	if err != nil {
		m.Logger.Error(err, "Error getting stack")
	}
	err = m.createOrUpdateSecret()
	if err != nil {
		return err
	}
	givenConfigurations := m.AppConfig.Spec

	for i := 0; int32(i) < givenConfigurations.Replicas; i++ {
		if len(*(mongoClusterStack.PersistentVolumeClaims)) <= i {
			m.Logger.Info(fmt.Sprintf("No persistent volume claim already existing for %d. Create it...", i))
			err = m.createOrUpdatePersistentVolumeClaim(getResourceGenericName(m.AppConfig.Name, strconv.Itoa(i)))
			if err != nil {
				return err
			}
		}
		if len(*(mongoClusterStack.Deployments)) <= i {
			m.Logger.Info(fmt.Sprintf("No deployment already existing for %d. Create it...", i))
			err = m.createOrUpdateDeployment(i, getResourceGenericName(m.AppConfig.Name, strconv.Itoa(i)))
			if err != nil {
				return err
			}
		}
		if len(*(mongoClusterStack.Services)) <= i {
			m.Logger.Info(fmt.Sprintf("No service already existing for %d. Create it...", i))
			err = m.createOrUpdateService(getResourceGenericName(m.AppConfig.Name, strconv.Itoa(i)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MongoClusterService) Delete() error {
	var err error
	mongoClusterStack, err := m.getStack()

	if err != nil || reflect.DeepEqual(*mongoClusterStack, MongoClusterStack{}) {
		m.Logger.Error(err, "Error getting stack")
		return err
	}

	if reflect.DeepEqual(*mongoClusterStack.Deployments, []v1.Deployment{}) {
	} else if err := m.deleteDeployments(*mongoClusterStack.Deployments); err != nil {
		return err
	}

	if reflect.DeepEqual(mongoClusterStack.Services, &[]v1api.Service{}) {
	} else if err := m.deleteServices(*mongoClusterStack.Services); err != nil {
		return err
	}

	if reflect.DeepEqual(mongoClusterStack.PersistentVolumeClaims, &[]v1api.PersistentVolumeClaim{}) {
	} else if err := m.deletePersistentVolumeClaims(*mongoClusterStack.PersistentVolumeClaims); err != nil {
		return err
	}

	if reflect.DeepEqual(mongoClusterStack.Secret, &v1api.Secret{}) {
	} else if err := m.deleteSecret(*mongoClusterStack.Secret); err != nil {
		return err
	}

	return nil
}

func (m *MongoClusterService) updateStack(resource interface{}) {
	if resource == nil {
		m.Logger.Info("The provided resource is null. Nothing to update")
		return
	}
	if reflect.TypeOf(resource) == reflect.TypeOf(v1.Deployment{}) {
		for i := 0; i < len(*m.Stack.Deployments); i++ {
			if (*m.Stack.Deployments)[i].Name == resource.(v1.Deployment).Name {
				(*m.Stack.Deployments)[i] = resource.(v1.Deployment)
				return
			}
		}
		*m.Stack.Deployments = append(*m.Stack.Deployments, resource.(v1.Deployment))
	} else if reflect.TypeOf(resource) == reflect.TypeOf(v1api.Service{}) {
		for i := 0; i < len(*m.Stack.Services); i++ {
			if (*(m.Stack.Services))[i].Name == resource.(v1api.Service).Name {
				(*m.Stack.Services)[i] = resource.(v1api.Service)
				return
			}
		}
		*m.Stack.Services = append(*m.Stack.Services, resource.(v1api.Service))
	} else if reflect.TypeOf(resource) == reflect.TypeOf(v1api.PersistentVolumeClaim{}) {
		for i := 0; i < len(*m.Stack.PersistentVolumeClaims); i++ {
			if (*m.Stack.PersistentVolumeClaims)[i].Name == resource.(v1api.PersistentVolumeClaim).Name {
				(*m.Stack.PersistentVolumeClaims)[i] = resource.(v1api.PersistentVolumeClaim)
				return
			}
		}
		*m.Stack.PersistentVolumeClaims = append(*m.Stack.PersistentVolumeClaims, resource.(v1api.PersistentVolumeClaim))
	} else if reflect.TypeOf(resource) == reflect.TypeOf(v1api.Secret{}) {
		secret := resource.(v1api.Secret)
		m.Stack.Secret = &secret
	} else {
		m.Logger.Info(fmt.Sprintf("Unknown type of resource %s. Nothing to update", reflect.TypeOf(resource)))
	}
}

func (m *MongoClusterService) getStack() (*MongoClusterStack, error) {
	deployments := m.getDeployments()
	services := m.getServices()
	pvcs := m.getPersistentVolumeClaims()

	secret, err := m.createPasswordSecret()
	if err != nil {
		return nil, err
	}

	mongoStack := MongoClusterStack{
		Deployments:            deployments,
		Services:               services,
		PersistentVolumeClaims: pvcs,
		Secret:                 secret,
	}
	return &mongoStack, nil
}
