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
	"fmt"
	"log"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

func Add(mgr manager.Manager) error {
	return add(mgr)
}

func add(mgr manager.Manager) error {
	name := "secret-puller-admission-controller-manager"
	namespace := "default"

	svr, err := webhook.NewServer(name, mgr, webhook.ServerOptions{
		CertDir: "/tmp/cert",
		BootstrapOptions: &webhook.BootstrapOptions{
			Service: &webhook.Service{
				Namespace: namespace,
				Name:      fmt.Sprintf("%s-service", name),
				// Selectors should select the pods that runs this webhook server.
				Selectors: map[string]string{
					"control-plane": "controller-manager",
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	wh, err := builder.NewWebhookBuilder().
		Mutating().
		Operations(admissionregistrationv1beta1.Create).
		ForType(&appsv1.Deployment{}).
		Handlers(&secretPullerInjector{}).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		WithManager(mgr).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	if err := svr.Register(wh); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Webhook server started")
	}

	return nil
}
