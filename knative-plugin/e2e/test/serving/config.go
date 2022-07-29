package serving

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingclient "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

var _ = ginkgo.Describe("Test config is synced down and applied as expected", func() {
	var (
		f *framework.Framework

		ns string

		vServingClient *servingclient.ServingV1Client
		pServingClient *servingclient.ServingV1Client
	)

	ginkgo.It("Initialize namespace and other base resources", func() {
		f = framework.DefaultFramework

		ns = fmt.Sprintf("e2e-knative-serving-%s", random.RandomString(5))

		var knativeClientErr error
		vServingClient, knativeClientErr = servingclient.NewForConfig(f.VclusterConfig)
		framework.ExpectNoError(knativeClientErr)

		pServingClient, knativeClientErr = servingclient.NewForConfig(f.HostConfig)
		framework.ExpectNoError(knativeClientErr)

		_, err := f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})

		framework.ExpectNoError(err)
	})

	ginkgo.It("Test if config CRD is synced", func() {
		resources, err := f.VclusterClient.DiscoveryClient.ServerResourcesForGroupVersion("serving.knative.dev/v1")
		framework.ExpectNoError(err, "Error encountered while fetching resources for serving.knative.dev/v1")

		var found bool
		for _, resource := range resources.APIResources {
			if resource.Kind == KnativeConfigurationResourceKind {
				found = true
			}
		}

		framework.ExpectNotEqual(found, false, "server does not recognise knative service, crd not synced")
	})

	ginkgo.It("Test create config as a result of ksvc", func() {
		ginkgo.By("Create ksvc", func() {
			_, err := vServingClient.Services(ns).Create(f.Context, &ksvcv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: KnativeServiceName,
				},
				Spec: ksvcv1.ServiceSpec{
					ConfigurationSpec: ksvcv1.ConfigurationSpec{
						Template: ksvcv1.RevisionTemplateSpec{
							Spec: ksvcv1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: KnativeHelloV1Image,
										},
									},
								},
							},
						},
					},
				},
			}, metav1.CreateOptions{})

			framework.ExpectNoError(err, "error creating ksvc in vcluster")
		})

		ginkgo.By("Test if vKsvc creates a pConfig", func() {
			err := wait.PollImmediate(time.Second, framework.PollTimeout, func() (bool, error) {
				_, err := pServingClient.Configurations(
					framework.DefaultVclusterNamespace).
					Get(f.Context,
						translate.PhysicalName(KnativeServiceName, ns),
						metav1.GetOptions{})
				if err != nil {
					if kerrors.IsNotFound(err) {
						return false, nil
					}

					return false, err
				}

				return true, nil
			})

			framework.ExpectNoError(err)
		})
	})

	// this should always be the last spec
	ginkgo.It("Destroy namespace", func() {
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})
})
