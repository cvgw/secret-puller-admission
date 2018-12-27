/*
Copyright 2018 Cole Wippern.

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

package secret_puller_injector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cvgw/secret-puller-admission/lib/secret_puller/factory"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

const (
	vaultAddrVar       = "VAULT_ADDR"
	injectorAnnotation = "secret-puller-injector.admission"
)

type secretPullerInjector struct {
	client  client.Client
	decoder types.Decoder
}

// secretPullerInjector implements admission.Handler.
var _ admission.Handler = &secretPullerInjector{}

// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=services,verbs=get;list;watch;create;update;patch;delete
func (a *secretPullerInjector) Handle(ctx context.Context, req types.Request) types.Response {
	log.Println("Request received")
	resource := &appsv1.Deployment{}

	err := a.decoder.Decode(req, resource)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	annotations := resource.Spec.Template.Annotations

	copy := resource.DeepCopy()
	if annotations != nil && annotations[injectorAnnotation] == "true" {
		log.Println("Resource has required annotation")
		err = a.mutateResourceFn(ctx, copy)
		if err != nil {
			return admission.ErrorResponse(http.StatusInternalServerError, err)
		}
	}

	// admission.PatchResponse generates a Response containing patches.
	return admission.PatchResponse(resource, copy)
}

// secretPullerInjector implements inject.Client.
var _ inject.Client = &secretPullerInjector{}

// InjectClient injects the client into the secretPullerInjector
func (a *secretPullerInjector) InjectClient(c client.Client) error {
	a.client = c
	return nil
}

// secretPullerInjector implements inject.Decoder.
var _ inject.Decoder = &secretPullerInjector{}

// InjectDecoder injects the decoder into the secretPullerInjector
func (a *secretPullerInjector) InjectDecoder(d types.Decoder) error {
	a.decoder = d
	return nil
}

func (a *secretPullerInjector) mutateResourceFn(ctx context.Context, resource *appsv1.Deployment) error {
	log.Println("I got a request")

	spec := resource.Spec.Template.Spec

	vaultAddr := os.Getenv(vaultAddrVar)
	if vaultAddr == "" {
		return errors.New(fmt.Sprintf("%s cannot be blank", vaultAddrVar))
	}

	secretPullerFactory := factory.New(vaultAddr, false)

	spec.InitContainers = append(spec.InitContainers, secretPullerFactory.Container())
	spec.Volumes = append(spec.Volumes, secretPullerFactory.Volumes()...)

	mutatedContainers := make([]corev1.Container, 0)
	for _, container := range spec.Containers {
		container.VolumeMounts = append(container.VolumeMounts, secretPullerFactory.VolumeMount())
		mutatedContainers = append(mutatedContainers, container)
	}

	spec.Containers = mutatedContainers

	resource.Spec.Template.Spec = spec

	return nil
}
