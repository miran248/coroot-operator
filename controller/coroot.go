package controller

import (
	"fmt"
	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *CorootReconciler) corootService(cr *corootv1.Coroot) *corev1.Service {
	ls := Labels(cr, "coroot")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-coroot", cr.Name),
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	port := cr.Spec.Service.Port
	if port == 0 {
		port = 8080
	}
	s.Spec = corev1.ServiceSpec{
		Selector: ls,
		Type:     cr.Spec.Service.Type,
		Ports: []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       port,
				TargetPort: intstr.FromString("http"),
				NodePort:   cr.Spec.Service.NodePort,
			},
		},
	}

	return s
}

func (r *CorootReconciler) corootDeployment(cr *corootv1.Coroot) *appsv1.Deployment {
	ls := Labels(cr, "coroot")
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-coroot",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	refreshInterval := cr.Spec.MetricRefreshInterval.Duration.String()
	if cr.Spec.MetricRefreshInterval.Duration == 0 {
		refreshInterval = corootv1.DefaultMetricRefreshInterval
	}

	env := []corev1.EnvVar{
		{Name: "BOOTSTRAP_REFRESH_INTERVAL", Value: refreshInterval},
		{Name: "BOOTSTRAP_PROMETHEUS_URL", Value: fmt.Sprintf("http://%s-prometheus.%s:9090", cr.Name, cr.Namespace)},
		{Name: "DO_NOT_CHECK_FOR_UPDATES"},
		{Name: "INSTALLATION_TYPE", Value: "k8s-operator"},
	}
	if cr.Spec.UrlBasePath != "" {
		env = append(env, corev1.EnvVar{Name: "URL_BASE_PATH", Value: cr.Spec.UrlBasePath})
	}
	if cr.Spec.PgConnectionString != "" {
		env = append(env, corev1.EnvVar{Name: "PG_CONNECTION_STRING", Value: cr.Spec.PgConnectionString})
	}
	if cr.Spec.CacheTTL.Duration > 0 {
		env = append(env, corev1.EnvVar{Name: "CACHE_TTL", Value: cr.Spec.CacheTTL.Duration.String()})
	}
	if cr.Spec.DisableUsageStatistics {
		env = append(env, corev1.EnvVar{Name: "DISABLE_USAGE_STATISTICS"})
	}
	if cr.Spec.AuthAnonymousRole != "" {
		env = append(env, corev1.EnvVar{Name: "AUTH_ANONYMOUS_ROLE", Value: cr.Spec.AuthAnonymousRole})
	}
	if cr.Spec.AuthBootstrapAdminPassword != "" {
		env = append(env, corev1.EnvVar{Name: "AUTH_BOOTSTRAP_ADMIN_PASSWORD", Value: cr.Spec.AuthBootstrapAdminPassword})
	}

	var image string
	if cr.Spec.EnterpriseEdition != nil {
		image = r.getAppImage(cr, AppCorootEE)
		env = append(env, corev1.EnvVar{Name: "LICENSE_KEY", Value: cr.Spec.EnterpriseEdition.LicenseKey})
	} else {
		image = r.getAppImage(cr, AppCorootCE)
	}

	if cr.Spec.ExternalClickhouse != nil {
		env = append(env,
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_ADDRESS", Value: cr.Spec.ExternalClickhouse.Address},
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_USER", Value: cr.Spec.ExternalClickhouse.User},
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_PASSWORD", Value: cr.Spec.ExternalClickhouse.Password},
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_DATABASE", Value: cr.Spec.ExternalClickhouse.Database},
		)
	} else {
		env = append(env,
			corev1.EnvVar{
				Name:  "BOOTSTRAP_CLICKHOUSE_ADDRESS",
				Value: fmt.Sprintf("%s-clickhouse.%s:9000", cr.Name, cr.Namespace),
			},
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_USER", Value: "default"},
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_PASSWORD", ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-clickhouse", cr.Name),
					},
					Key: "password",
				},
			}},
			corev1.EnvVar{Name: "BOOTSTRAP_CLICKHOUSE_DATABASE", Value: "default"},
		)
	}

	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: ls,
			},
			Spec: corev1.PodSpec{
				SecurityContext: nonRootSecurityContext,
				Affinity:        cr.Spec.Affinity,
				Containers: []corev1.Container{
					{
						Image: image,
						Name:  "coroot",
						Args: []string{
							"--listen=:8080",
							"--data-dir=/data",
						},
						Env: env,
						Ports: []corev1.ContainerPort{
							{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "data", MountPath: "/data"},
						},
						Resources: cr.Spec.Resources,
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromString("http")},
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "data-" + cr.Name + "-coroot",
							},
						},
					},
				},
			},
		},
	}

	return d
}

func (r *CorootReconciler) corootPVC(cr *corootv1.Coroot) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "data-" + cr.Name + "-coroot",
			Namespace: cr.Namespace,
			Labels:    Labels(cr, "coroot"),
		},
	}

	size := cr.Spec.Storage.Size
	if size.IsZero() {
		size, _ = resource.ParseQuantity("10Gi")
	}
	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: size,
			},
		},
		StorageClassName: cr.Spec.Storage.ClassName,
	}

	return pvc
}
