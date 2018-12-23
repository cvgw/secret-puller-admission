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

package factory

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	secretPullerImage    = "zendesk/samson_secret_puller:latest"
	initContainerName    = "samson-secret-puller"
	imagePullPolicy      = "IfNotPresent"
	vaultAuthVolumeName  = "vault-auth"
	secretKeysVolumeName = "secret-keys"
	initContainerCPU     = "1000m"
	initContainerMem     = "1G"
	secretVolumeName     = "secrets"
	secretMountPath      = "/secrets"

	vaultAuthSecretName = "vaultauth"

	vaultDefaultAddr = "https://vault:8200"
	secretPrefix     = "secret"
)

type factory struct {
	vaultAddr      string
	vaultVerifyTLS string
}

func New(vaultAddr string, vaultVerifyTLS bool) factory {
	f := factory{
		vaultAddr:      vaultAddr,
		vaultVerifyTLS: fmt.Sprintf("%t", vaultVerifyTLS),
	}

	return f
}

func (f factory) Container() corev1.Container {
	return corev1.Container{
		Name:            initContainerName,
		Image:           secretPullerImage,
		ImagePullPolicy: imagePullPolicy,
		Env: []corev1.EnvVar{
			{
				Name:  "VAULT_ADDR",
				Value: f.vaultAddr,
			},
			{
				Name:  "VAULT_SSL_VERIFY",
				Value: f.vaultVerifyTLS,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			f.VolumeMount(),
			{
				Name:      vaultAuthVolumeName,
				MountPath: "/vault-auth",
			},
			{
				Name:      secretKeysVolumeName,
				MountPath: "/secretkeys",
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(initContainerCPU),
				corev1.ResourceMemory: resource.MustParse(initContainerMem),
			},
		},
	}
}

func (f factory) Volumes() (volumes []corev1.Volume) {
	volumes = append(volumes, corev1.Volume{
		Name: secretVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	})

	volumes = append(volumes, corev1.Volume{
		Name: vaultAuthVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: vaultAuthSecretName,
			},
		},
	})

	volumes = append(volumes, corev1.Volume{
		Name: secretKeysVolumeName,
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{Path: "annotations", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"}},
				},
			},
		},
	})

	return volumes
}

func (f factory) VolumeMount() (mount corev1.VolumeMount) {
	mount.Name = secretVolumeName
	mount.MountPath = secretMountPath

	return mount
}
