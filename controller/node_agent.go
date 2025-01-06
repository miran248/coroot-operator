package controller

import (
	"fmt"
	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (r *CorootReconciler) nodeAgentDaemonSet(cr *corootv1.Coroot) *appsv1.DaemonSet {
	ls := Labels(cr, "coroot-node-agent")
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-node-agent",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	collectorEndpoint := fmt.Sprintf("http://%s-coroot.%s:8080", cr.Name, cr.Namespace)
	if cr.Spec.AgentsOnly != nil {
		collectorEndpoint = cr.Spec.AgentsOnly.CorootURL
	}
	scrapeInterval := cr.Spec.MetricsRefreshInterval.Duration.String()
	if cr.Spec.MetricsRefreshInterval.Duration == 0 {
		scrapeInterval = corootv1.DefaultMetricRefreshInterval
	}
	env := []corev1.EnvVar{
		{Name: "COLLECTOR_ENDPOINT", Value: collectorEndpoint},
		{Name: "API_KEY", Value: cr.Spec.ApiKey},
		{Name: "SCRAPE_INTERVAL", Value: scrapeInterval},
	}
	for _, e := range cr.Spec.NodeAgent.Env {
		env = append(env, e)
	}

	resources := cr.Spec.NodeAgent.Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		}
	}
	if resources.Limits == nil {
		resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		}
	}

	tolerations := cr.Spec.NodeAgent.Tolerations
	if len(tolerations) == 0 {
		tolerations = []corev1.Toleration{{Operator: corev1.TolerationOpExists}}
	}

	ds.Spec = appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		UpdateStrategy: cr.Spec.NodeAgent.UpdateStrategy,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ls,
				Annotations: cr.Spec.NodeAgent.PodAnnotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: cr.Name + "-node-agent",
				HostPID:            true,
				Tolerations:        tolerations,
				PriorityClassName:  cr.Spec.NodeAgent.PriorityClassName,
				Affinity:           cr.Spec.NodeAgent.Affinity,
				Containers: []corev1.Container{
					{
						Name:  "node-agent",
						Image: r.getAppImage(cr, AppNodeAgent),
						Args: []string{
							"--cgroupfs-root=/host/sys/fs/cgroup",
						},
						SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
						Env:             env,
						Resources:       resources,
						VolumeMounts: []corev1.VolumeMount{
							{Name: "cgroupfs", MountPath: "/host/sys/fs/cgroup", ReadOnly: true},
							{Name: "tracefs", MountPath: "/sys/kernel/tracing"},
							{Name: "debugfs", MountPath: "/sys/kernel/debug"},
							{Name: "tmp", MountPath: "/tmp"},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "cgroupfs",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/sys/fs/cgroup",
							},
						},
					},
					{
						Name: "tracefs",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/sys/kernel/tracing",
							},
						},
					},
					{
						Name: "debugfs",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/sys/kernel/debug",
							},
						},
					},
					{
						Name: "tmp",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}

	return ds
}
