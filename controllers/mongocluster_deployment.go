package controllers

import (
	"fmt"
	v1 "k8s.io/api/apps/v1"
	v1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

func (m *MongoClusterService) getDeployments() *[]v1.Deployment {
	var deploy v1.Deployment
	getOne := func(name string) (error, *v1.Deployment) {
		if err := m.Reconciler.Client.Get(
			*m.Context,
			types.NamespacedName{Name: name, Namespace: m.Namespace},
			&deploy); err != nil {
			return errors.NewNotFound(v1.Resource("deployment"), ""), nil
		}
		return nil, &deploy
	}
	var deployments []v1.Deployment

	for i := 0; int32(i) < m.AppConfig.Spec.Replicas; i++ {
		deploymentName := getResourceGenericName(m.AppConfig.Name, fmt.Sprintf("%d", i))
		err, deploy := getOne(deploymentName)
		if err != nil {
			m.Logger.Info(fmt.Sprintf("Deployment %s does not exist", deploymentName))
		} else {
			deployments = append(deployments, *deploy)
		}
	}
	return &deployments
}

func (m *MongoClusterService) createOrUpdateDeployment(id int, deploymentName string) error {
	actualDeployment := &v1.Deployment{}
	expectedDeployment, err := m.createDeployment(id, deploymentName)

	if err != nil {
		m.Logger.Error(err, "Error creating Deployment")
		return err
	}
	err = m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: deploymentName, Namespace: m.Namespace}, actualDeployment)
	deploymentUpToDate := !reflect.DeepEqual((*expectedDeployment).Spec, (*actualDeployment).Spec) && m.deploymentExists(*expectedDeployment)

	if err != nil && !errors.IsNotFound(err) {
		m.Logger.Error(err, "Error getting deployment")
		return err
	} else if errors.IsNotFound(err) {
		m.Logger.Info(fmt.Sprintf("Creating deployment %s", expectedDeployment.Name))
		err = m.Reconciler.Client.Create(*m.Context, expectedDeployment)
		if err != nil {
			m.Logger.Error(err, "Error creating deployment")
			return err
		}
	} else if deploymentUpToDate {
		m.Logger.Info(fmt.Sprintf("Deployment %s is up to date. Nothing to do.", expectedDeployment.Name))
		return nil
	} else {
		m.Logger.Info(fmt.Sprintf("Updating Deployment %s", expectedDeployment.Name))
		actualDeployment.Spec = expectedDeployment.Spec
		err = m.Reconciler.Client.Update(*m.Context, actualDeployment)
		if err != nil {
			m.Logger.Error(err, "Error updating Deployment")
			return err
		}
	}
	m.updateStack(*expectedDeployment)
	return nil
}

func (m *MongoClusterService) createDeployment(id int, name string) (*v1.Deployment, error) {
	envVars, err := m.createPodEnvVariables(id)
	if err != nil {
		m.Logger.Error(err, "Error getting mongo pod env variables")
		return nil, err
	}
	replicas := int32(MONGO_DEPLOYMENT_REPLICAS)
	return &v1.Deployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      name,
			Namespace: m.Namespace,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Replicas: &replicas,
			Template: v1api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: v1api.PodSpec{
					Containers: []v1api.Container{
						{
							Name:  MONGO_CONTAINER_NAME,
							Image: MONGO_CONTAINER_IMAGE,
							Env:   envVars,
							Ports: []v1api.ContainerPort{
								{
									ContainerPort: MONGO_CONTAINER_PORT,
								},
							},
							Command: []string{"/bin/bash"},
							Args:    []string{"/scripts/run.sh"},
							VolumeMounts: []v1api.VolumeMount{
								{
									Name:      MONGO_KEY_VOLUME_NAME,
									MountPath: MONGO_KEY_MOUNT_PATH,
									ReadOnly:  true,
								},
								{
									Name:      MONGO_STORAGE_VOLUME_NAME,
									MountPath: MONGO_STORAGE_MOUNT_PATH,
								},
							},
						},
					},
					Volumes: []v1api.Volume{
						{
							Name: MONGO_KEY_VOLUME_NAME,
							VolumeSource: v1api.VolumeSource{
								Secret: &v1api.SecretVolumeSource{
									SecretName:  DEFAULT_PASSWORD_SECRET_NAME,
									DefaultMode: &MONGO_KEY_SECRET_DEFAULT_MODE,
								},
							},
						},
						{
							Name: MONGO_STORAGE_VOLUME_NAME,
							VolumeSource: v1api.VolumeSource{
								PersistentVolumeClaim: &v1api.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (m *MongoClusterService) createPodEnvVariables(replicaId int) ([]v1api.EnvVar, error) {
	passwordSecret, err := m.createPasswordSecret()
	if err != nil {
		m.Logger.Error(err, "Error creating password secret")
		return nil, err
	}
	clusterMembers := getClusterMembers(m.AppConfig.Name, int(m.AppConfig.Spec.Replicas))
	if clusterMembers == "" {
		return nil, errors.NewInternalError(fmt.Errorf("error creating cluster members"))
	}
	return []v1api.EnvVar{
		{
			Name:  "MONGODB_USERNAME",
			Value: MONGODB_DEFAULT_USER,
		},
		{
			Name: "MONGODB_PASSWORD",
			ValueFrom: &v1api.EnvVarSource{
				SecretKeyRef: &v1api.SecretKeySelector{
					LocalObjectReference: v1api.LocalObjectReference{
						Name: passwordSecret.Name,
					},
					Key: "password",
				},
			},
		},
		{
			Name:  "MONGODB_DBNAME",
			Value: m.AppConfig.Spec.DatabaseName,
		},
		{
			Name:  "MONGODB_ROLE",
			Value: MONGODB_DEFAULT_ROLE,
		},
		{
			Name:  "CLUSTER_MEMBERS",
			Value: clusterMembers,
		},
		{
			Name:  "MONGODB_REPLICA_ID",
			Value: strconv.Itoa(replicaId),
		},
		{
			Name:  "HOST",
			Value: fmt.Sprintf("%s-%d", MONGODB_DEFAULT_HOST, replicaId),
		},
		{
			Name:  "DEBIAN_FRONTEND",
			Value: "noninteractive",
		},
		{
			Name:  "DEBCONF_NONINTERACTIVE_SEEN",
			Value: "true",
		},
	}, nil
}

func (m *MongoClusterService) deleteDeployments(deployments []v1.Deployment) error {
	for _, deployment := range deployments {
		if !reflect.DeepEqual(deployment, v1.Deployment{}) {
			break
		}
		err := m.Reconciler.Client.Delete(*m.Context, &deployment)
		if err != nil {
			m.Logger.Error(err, "Error deleting deployment")
			return err
		}
	}
	return nil
}

func (m *MongoClusterService) deploymentExists(deployment v1.Deployment) bool {
	err := m.Reconciler.Client.Get(*m.Context, types.NamespacedName{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
	}, &deployment)
	if err != nil && errors.IsNotFound(err) {
		return false
	} else if err != nil {
		m.Logger.Error(err, "Error getting deployment")
		return false
	}
	return true

}
