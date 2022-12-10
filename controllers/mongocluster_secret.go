package controllers

import (
	"context"
	"fmt"
	v1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (m *MongoClusterService) createOrUpdateSecret() error {
	actualSecret := &v1api.Secret{}
	expectedSecret, err := m.createPasswordSecret()
	if err != nil {
		m.Logger.Error(err, "Error creating Password Secret")
		return err
	}
	err = m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: expectedSecret.Name, Namespace: m.Namespace}, actualSecret)
	secretUpToDate := !reflect.DeepEqual((*expectedSecret).Data, (*actualSecret).Data) && m.secretExists(*expectedSecret)

	if err != nil && !errors.IsNotFound(err) {
		m.Logger.Error(err, "Error getting service")
		return err
	} else if errors.IsNotFound(err) {
		m.Logger.Info(fmt.Sprintf("Creating service %s", expectedSecret.Name))
		err = m.Reconciler.Client.Create(*m.Context, expectedSecret)
		if err != nil {
			m.Logger.Error(err, "Error creating service")
			return err
		}
	} else if secretUpToDate {
		m.Logger.Info(fmt.Sprintf("Secret %s is up to date. Nothing to do.", expectedSecret.Name))
		return nil
	} else {
		m.Logger.Info(fmt.Sprintf("Updating service %s", expectedSecret.Name))
		actualSecret.Data = expectedSecret.Data
		err = m.Reconciler.Client.Update(*m.Context, actualSecret)
		if err != nil {
			m.Logger.Error(err, "Error updating service")
			return err
		}
	}
	m.updateStack(expectedSecret)

	return nil
}

func (m *MongoClusterService) createPasswordSecret() (*v1api.Secret, error) {
	var password string
	mongoCluster := m.AppConfig
	apiSecretResult := &v1api.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace, Name: mongoCluster.Spec.Auth.ExistingSecretName}
	if mongoCluster.Spec.Auth.ExistingSecretName != "" {
		err := m.Reconciler.Client.Get(context.Background(), key, apiSecretResult)
		if err != nil {
			m.Logger.Error(err, fmt.Sprintf("The provided password secret does not exist"))
			return nil, err
		}
		return apiSecretResult, nil
	} else if mongoCluster.Spec.Auth.Password != "" {
		password = mongoCluster.Spec.Auth.Password
	} else {
		m.Logger.Info(fmt.Sprintf("No password or secret name provided \n"+
			"INSECURE: take default password [%s]"+
			"", MONGODB_DEFAULT_PASSWORD))
		err := m.Reconciler.Client.Get(
			context.Background(),
			types.NamespacedName{Namespace: m.Namespace, Name: DEFAULT_PASSWORD_SECRET_NAME},
			apiSecretResult)
		if err == nil {
			return apiSecretResult, nil
		}
		password = MONGODB_DEFAULT_PASSWORD
	}
	return &v1api.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      DEFAULT_PASSWORD_SECRET_NAME,
			Namespace: m.Namespace,
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
	}, nil
}

func (m *MongoClusterService) deleteSecret(secret v1api.Secret) error {
	if !reflect.DeepEqual(secret, v1api.Secret{}) {
		return nil
	}
	err := m.Reconciler.Client.Delete(*m.Context, &secret)
	if err != nil {
		m.Logger.Error(err, "Error deleting secret")
		return err
	}
	return nil
}

func (m *MongoClusterService) secretExists(secret v1api.Secret) bool {
	err := m.Reconciler.Client.Get(*m.Context, types.NamespacedName{Name: secret.Name, Namespace: m.Namespace}, &secret)
	if err != nil {
		return false
	}
	return true
}
