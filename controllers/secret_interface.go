/*
Copyright 2022 - Janis Meybohm, Wikimedia Foundation Inc.

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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNotImplemented = errors.New("not implemented")

// SecretsClient implements partly the corev1.SecretsInterface for the helm secret storage driver to be happy
type SecretsClient struct {
	client    client.Client
	Namespace string
}

// NewSecretsClient returns a new SecretsClient
func NewSecretsClient(client client.Client, namespace string) *SecretsClient {
	return &SecretsClient{client: client, Namespace: namespace}
}

func (s *SecretsClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Secret, error) {
	var secret v1.Secret
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: s.Namespace,
	}
	err := s.client.Get(ctx, namespacedName, &secret, &client.GetOptions{Raw: &opts})
	return &secret, err
}
func (s *SecretsClient) Create(ctx context.Context, secret *v1.Secret, opts metav1.CreateOptions) (*v1.Secret, error) {
	secret.Namespace = s.Namespace
	err := s.client.Create(ctx, secret, &client.CreateOptions{Raw: &opts})
	return nil, err
}
func (s *SecretsClient) Update(ctx context.Context, secret *v1.Secret, opts metav1.UpdateOptions) (*v1.Secret, error) {
	secret.Namespace = s.Namespace
	err := s.client.Update(ctx, secret, &client.UpdateOptions{Raw: &opts})
	return nil, err
}
func (s *SecretsClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	secret, err := s.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return s.client.Delete(ctx, secret, &client.DeleteOptions{Raw: &opts})
}
func (s *SecretsClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.SecretList, error) {
	var secrets v1.SecretList
	err := s.client.List(ctx, &secrets, &client.ListOptions{Raw: &opts})
	if err != nil {
		return nil, err
	}
	return &secrets, nil
}
func (s *SecretsClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return ErrNotImplemented
}
func (s *SecretsClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, ErrNotImplemented
}
func (s *SecretsClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Secret, err error) {
	return nil, ErrNotImplemented
}
func (s *SecretsClient) Apply(ctx context.Context, secret *corev1.SecretApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Secret, err error) {
	return nil, ErrNotImplemented
}
