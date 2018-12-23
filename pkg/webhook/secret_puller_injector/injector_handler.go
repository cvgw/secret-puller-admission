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
	"log"
	"net/http"

	"github.com/cvgw/secret-puller-admission/lib/secret_puller/factory"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

const (
	vaultAddr = "localhost:8200"
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
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	copy := pod.DeepCopy()

	err = a.mutatePodsFn(ctx, copy)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}

	// admission.PatchResponse generates a Response containing patches.
	return admission.PatchResponse(pod, copy)
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

func (a *secretPullerInjector) mutatePodsFn(ctx context.Context, pod *corev1.Pod) error {
	log.Println("I got a request")

	secretPullerFactory := factory.New(vaultAddr, false)

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, secretPullerFactory.Container())
	pod.Spec.Volumes = append(pod.Spec.Volumes, secretPullerFactory.Volumes()...)

	for _, container := range pod.Spec.Containers {
		container.VolumeMounts = append(container.VolumeMounts, secretPullerFactory.VolumeMount())
	}

	return nil
}
