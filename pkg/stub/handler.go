package stub

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/previousnext/solr-operator/pkg/apis/solr/v1alpha1"
	"github.com/prometheus/common/log"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Solr:
		log.With("namespace", o.ObjectMeta.Namespace).Infoln("Received solr provisioning request")
		err := sdk.Create(provisionDeployment(o))
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to provision deployment")
		}
		log.With("namespace", o.ObjectMeta.Namespace).Infoln("Provisioned deployment object")
		err = sdk.Create(provisionService(o))
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to provision service")
		}
		log.With("namespace", o.ObjectMeta.Namespace).Infoln("Provisioned service object")
		// @todo add PVC.
		//err = sdk.Create(provisionPvc(o))
		//if err != nil && !apierrors.IsAlreadyExists(err) {
		//	return errors.Wrap(err, "failed to provision PVC")
		//}
	}
	return nil
}

// provisionDeployment provisions a deployment.
func provisionDeployment(solr *v1alpha1.Solr) *appsv1.Deployment {
	var (
		replicas   = int32(1)
		history    = int32(2)
		repository = "previousnext/solr"
		port       = 8983
		core       = "core1"
		data       = "/opt/solr/data"
	)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("solr-%s", solr.Spec.Name),
			Namespace: solr.ObjectMeta.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(solr, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    solr.Kind,
				}),
			},
			Labels: map[string]string{
				"solr": solr.Spec.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:             &replicas,
			RevisionHistoryLimit: &history,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"solr": solr.Spec.Name,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: solr.Spec.Name,
					Labels: map[string]string{
						"solr": solr.Spec.Name,
					},
					Namespace: solr.ObjectMeta.Namespace,
				},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						// Our Solr containers run as the user "solr".
						// This container will ensure that the permissions are set.
						// Otherwise Solr will fail to boot in the first instance.
						{
							Name:            "permissions",
							Image:           fmt.Sprintf("%s:init", repository),
							ImagePullPolicy: "Always",
							Command: []string{
								"chown",
								"-R",
								"solr:solr",
								data,
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "solr",
							Image: fmt.Sprintf("%s:%s", repository, solr.Spec.Version),
							Ports: []v1.ContainerPort{
								{
									ContainerPort: int32(port),
								},
							},
							Env: []v1.EnvVar{
								{
									Name:  "SOLR_HEAP",
									Value: "256m",
								},
								{
									Name:  "SOLR_CORE",
									Value: core,
								},
							},
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{
									// https://cwiki.apache.org/confluence/display/solr/Ping
									TCPSocket: &v1.TCPSocketAction{
										Port: intstr.FromInt(port),
									},
								},
								InitialDelaySeconds: 300,
								TimeoutSeconds:      10,
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("100m"),
									v1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("100m"),
									v1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "data",
									MountPath: data,
								},
							},
							ImagePullPolicy: "Always",
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "data",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
							// @todo add PVC
							//VolumeSource: v1.VolumeSource{
							//	PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							//		ClaimName: solr.Spec.Name,
							//	},
							//},
						},
					},
				},
			},
		},
	}
}

// provisionService provisions a service.
func provisionService(solr *v1alpha1.Solr) *v1.Service {
	var (
		port = 8983
	)
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("solr-%s", solr.Spec.Name),
			Namespace: solr.ObjectMeta.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(solr, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    solr.Kind,
				}),
			},
			Labels: map[string]string{
				"solr": solr.Spec.Name,
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: int32(port),
				},
			},
			Selector: map[string]string{
				"solr": solr.Spec.Name,
			},
			SessionAffinity: v1.ServiceAffinityNone,
			Type:            v1.ServiceTypeClusterIP,
		},
	}
}
