package controller

import (
	"fmt"
	corootv1 "github.io/coroot/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *CorootReconciler) clusterAgentServiceAccount(cr *corootv1.Coroot) *corev1.ServiceAccount {
	a := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cluster-agent",
			Namespace: cr.Namespace,
			Labels:    Labels(cr, "coroot-cluster-agent"),
		},
	}
	return a
}

func (r *CorootReconciler) clusterAgentClusterRoleBinding(cr *corootv1.Coroot) *rbacv1.ClusterRoleBinding {
	b := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cr.Name + "-cluster-agent",
			Labels: Labels(cr, "coroot-cluster-agent"),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      cr.Name + "-cluster-agent",
				Namespace: cr.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     cr.Name + "-cluster-agent",
		},
	}
	return b
}

func (r *CorootReconciler) clusterAgentClusterRole(cr *corootv1.Coroot) *rbacv1.ClusterRole {
	verbs := []string{"get", "list", "watch"}
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cr.Name + "-cluster-agent",
			Labels: Labels(cr, "coroot-cluster-agent"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "nodes", "pods", "services", "endpoints", "persistentvolumeclaims", "persistentvolumes"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "replicasets", "daemonsets", "statefulsets", "cronjobs"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs", "jobs"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses", "volumeattachments"},
				Verbs:     verbs,
			},
		},
	}
	return role
}

func (r *CorootReconciler) clusterAgentDeployment(cr *corootv1.Coroot) *appsv1.Deployment {
	ls := Labels(cr, "coroot-cluster-agent")
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cluster-agent",
			Namespace: cr.Namespace,
			Labels:    ls,
		},
	}

	corootUrl := fmt.Sprintf("http://%s-coroot.%s:8080", cr.Name, cr.Namespace)
	if cr.Spec.AgentsOnly != nil {
		corootUrl = cr.Spec.AgentsOnly.CorootURL
	}
	scrapeInterval := cr.Spec.MetricsRefreshInterval.Duration.String()
	if cr.Spec.MetricsRefreshInterval.Duration == 0 {
		scrapeInterval = corootv1.DefaultMetricRefreshInterval
	}
	env := []corev1.EnvVar{
		{Name: "COROOT_URL", Value: corootUrl},
		{Name: "API_KEY", Value: cr.Spec.ApiKey},
		{Name: "METRICS_SCRAPE_INTERVAL", Value: scrapeInterval},
		{Name: "KUBE_STATE_METRICS_ADDRESS", Value: "127.0.0.1:10302"},
	}
	for _, e := range cr.Spec.ClusterAgent.Env {
		env = append(env, e)
	}
	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: ls,
			},
			Spec: corev1.PodSpec{
				SecurityContext:    nonRootSecurityContext,
				ServiceAccountName: cr.Name + "-cluster-agent",
				Affinity:           cr.Spec.ClusterAgent.Affinity,
				Containers: []corev1.Container{
					{
						Image: r.getAppImage(cr, AppClusterAgent),
						Name:  "cluster-agent",
						Args: []string{
							"--listen=127.0.0.1:10301",
							"--metrics-wal-dir=/tmp",
						},
						Resources: cr.Spec.ClusterAgent.Resources,
						VolumeMounts: []corev1.VolumeMount{
							{Name: "tmp", MountPath: "/tmp"},
						},
						Env: env,
					},
					{
						Image: "registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.13.0",
						Name:  "kube-state-metrics",
						Args: []string{
							"--host=127.0.0.1",
							"--port=10302",
							"--resources=namespaces,nodes,daemonsets,deployments,cronjobs,jobs,persistentvolumeclaims,persistentvolumes,pods,replicasets,services,statefulsets,storageclasses,volumeattachments",
							"--metric-labels-allowlist=pods=[*]",
						},
						Resources: corev1.ResourceRequirements{
							Requests: cr.Spec.ClusterAgent.Resources.Requests,
							Limits:   cr.Spec.ClusterAgent.Resources.Limits,
						},
					},
				},
				Volumes: []corev1.Volume{
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

	return d
}
