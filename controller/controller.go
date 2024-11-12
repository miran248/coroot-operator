package controller

import (
	"context"
	"fmt"
	corootv1 "github.io/coroot/operator/api/v1"
	"golang.org/x/exp/maps"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
	"time"
)

const (
	AppVersionsUpdateInterval = time.Hour
	BusyboxImage              = "busybox:1.36"
)

type CorootReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	instances     map[ctrl.Request]bool
	instancesLock sync.Mutex

	versions     map[App]string
	versionsLock sync.Mutex
}

func NewCorootReconciler(mgr ctrl.Manager) *CorootReconciler {
	r := &CorootReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		instances: map[ctrl.Request]bool{},
		versions:  map[App]string{},
	}

	r.fetchAppVersions()
	go func() {
		for range time.Tick(AppVersionsUpdateInterval) {
			r.fetchAppVersions()
			r.instancesLock.Lock()
			instances := maps.Keys(r.instances)
			r.instancesLock.Unlock()
			for _, i := range instances {
				_, _ = r.Reconcile(nil, i)
			}
		}
	}()

	return r
}

// +kubebuilder:rbac:groups=coroot.com,resources=coroots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coroot.com,resources=coroots/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coroot.com,resources=coroots/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=list;watch;create
// +kubebuilder:rbac:groups="",resources=namespaces;nodes;pods;endpoints;persistentvolumes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;daemonsets;statefulsets;cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses;volumeattachments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *CorootReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cr := &corootv1.Coroot{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Coroot has been deleted")
			r.instancesLock.Lock()
			delete(r.instances, req)
			r.instancesLock.Unlock()
			cr = &corootv1.Coroot{}
			cr.Name = req.Name
			cr.Namespace = req.Namespace
			_ = r.Delete(ctx, r.clusterAgentClusterRoleBinding(cr))
			_ = r.Delete(ctx, r.clusterAgentClusterRole(cr))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	r.instancesLock.Lock()
	r.instances[req] = true
	r.instancesLock.Unlock()

	r.CreateOrUpdateDaemonSet(ctx, cr, r.nodeAgentDaemonSet(cr))

	r.CreateOrUpdateDeployment(ctx, cr, r.clusterAgentDeployment(cr))
	r.CreateOrUpdateClusterRole(ctx, cr, r.clusterAgentClusterRole(cr))
	r.CreateOrUpdateClusterRoleBinding(ctx, cr, r.clusterAgentClusterRoleBinding(cr))

	if cr.Spec.AgentsOnly != nil {
		// TODO: delete
		return ctrl.Result{}, nil
	}

	r.CreateOrUpdatePVC(ctx, cr, r.corootPVC(cr))
	r.CreateOrUpdateDeployment(ctx, cr, r.corootDeployment(cr))
	r.CreateOrUpdateService(ctx, cr, r.corootService(cr))

	r.CreateOrUpdatePVC(ctx, cr, r.prometheusPVC(cr))
	r.CreateOrUpdateDeployment(ctx, cr, r.prometheusDeployment(cr))
	r.CreateOrUpdateService(ctx, cr, r.prometheusService(cr))

	if cr.Spec.ExternalClickhouse == nil {
		r.CreateSecret(ctx, cr, r.clickhouseSecret(cr))

		for _, pvc := range r.clickhouseKeeperPVCs(cr) {
			r.CreateOrUpdatePVC(ctx, cr, pvc)
		}
		r.CreateOrUpdateStatefulSet(ctx, cr, r.clickhouseKeeperStatefulSet(cr))
		r.CreateOrUpdateService(ctx, cr, r.clickhouseKeeperServiceHeadless(cr))

		r.CreateOrUpdateService(ctx, cr, r.clickhouseServiceHeadless(cr))
		for _, pvc := range r.clickhousePVCs(cr) {
			r.CreateOrUpdatePVC(ctx, cr, pvc)
		}
		for _, clickhouse := range r.clickhouseStatefulSets(cr) {
			r.CreateOrUpdateStatefulSet(ctx, cr, clickhouse)
		}
		r.CreateOrUpdateService(ctx, cr, r.clickhouseService(cr))
	} else {
		// TODO: delete
	}

	return ctrl.Result{}, nil
}

func (r *CorootReconciler) CreateOrUpdate(ctx context.Context, cr *corootv1.Coroot, obj client.Object, f controllerutil.MutateFn) {
	logger := log.FromContext(nil, "type", fmt.Sprintf("%T", obj), "name", obj.GetName(), "namespace", obj.GetNamespace())
	_ = ctrl.SetControllerReference(cr, obj, r.Scheme)
	errMsg := "failed to create or update"
	if f == nil {
		f = func() error { return nil }
		errMsg = "failed to create"
	}
	res, err := ctrl.CreateOrUpdate(ctx, r.Client, obj, f)
	if err != nil {
		logger.Error(err, errMsg)
		return
	}
	if res != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("%s", res))
	}
}

func (r *CorootReconciler) CreateSecret(ctx context.Context, cr *corootv1.Coroot, s *corev1.Secret) {
	r.CreateOrUpdate(ctx, cr, s, nil)
}

func (r *CorootReconciler) CreateOrUpdateDeployment(ctx context.Context, cr *corootv1.Coroot, d *appsv1.Deployment) {
	r.CreateOrUpdateServiceAccount(ctx, cr, d.ObjectMeta)
	d.Spec.Template.Spec.ServiceAccountName = d.ObjectMeta.Name
	spec := d.Spec
	r.CreateOrUpdate(ctx, cr, d, func() error {
		return Merge(&d.Spec, spec)
	})
}

func (r *CorootReconciler) CreateOrUpdateDaemonSet(ctx context.Context, cr *corootv1.Coroot, ds *appsv1.DaemonSet) {
	r.CreateOrUpdateServiceAccount(ctx, cr, ds.ObjectMeta)
	ds.Spec.Template.Spec.ServiceAccountName = ds.ObjectMeta.Name
	spec := ds.Spec
	r.CreateOrUpdate(ctx, cr, ds, func() error {
		return Merge(&ds.Spec, spec)
	})
}

func (r *CorootReconciler) CreateOrUpdateStatefulSet(ctx context.Context, cr *corootv1.Coroot, ss *appsv1.StatefulSet) {
	r.CreateOrUpdateServiceAccount(ctx, cr, ss.ObjectMeta)
	ss.Spec.Template.Spec.ServiceAccountName = ss.ObjectMeta.Name
	spec := ss.Spec
	r.CreateOrUpdate(ctx, cr, ss, func() error {
		volumeClaimTemplates := ss.Spec.VolumeClaimTemplates[:]
		err := Merge(&ss.Spec, spec)
		ss.Spec.VolumeClaimTemplates = volumeClaimTemplates
		return err
	})
}

func (r *CorootReconciler) CreateOrUpdatePVC(ctx context.Context, cr *corootv1.Coroot, pvc *corev1.PersistentVolumeClaim) {
	spec := pvc.Spec
	r.CreateOrUpdate(ctx, cr, pvc, func() error {
		return Merge(&pvc.Spec, spec)
	})
}

func (r *CorootReconciler) CreateOrUpdateService(ctx context.Context, cr *corootv1.Coroot, s *corev1.Service) {
	spec := s.Spec
	r.CreateOrUpdate(ctx, cr, s, func() error {
		err := Merge(&s.Spec, spec)
		s.Spec.Ports = spec.Ports
		return err
	})
}

func (r *CorootReconciler) CreateOrUpdateServiceAccount(ctx context.Context, cr *corootv1.Coroot, om metav1.ObjectMeta) {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
		Name:      om.Name,
		Namespace: om.Namespace,
		Labels:    om.Labels,
	}}
	r.CreateOrUpdate(ctx, cr, sa, nil)
}

func (r *CorootReconciler) CreateOrUpdateClusterRole(ctx context.Context, cr *corootv1.Coroot, role *rbacv1.ClusterRole) {
	rules := role.Rules
	r.CreateOrUpdate(ctx, cr, role, func() error {
		role.Rules = rules
		return nil
	})
}

func (r *CorootReconciler) CreateOrUpdateClusterRoleBinding(ctx context.Context, cr *corootv1.Coroot, b *rbacv1.ClusterRoleBinding) {
	r.CreateOrUpdate(ctx, cr, b, nil)
}

func (r *CorootReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corootv1.Coroot{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func Labels(cr *corootv1.Coroot, component string) map[string]string {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return map[string]string{
		"app.kubernetes.io/managed-by": "coroot-operator",
		"app.kubernetes.io/part-of":    cr.Name,
		"app.kubernetes.io/component":  component,
	}
}

var nonRootSecurityContext = &corev1.PodSecurityContext{
	RunAsNonRoot: ptr.To(true),
	RunAsUser:    ptr.To(int64(65534)),
	RunAsGroup:   ptr.To(int64(65534)),
	FSGroup:      ptr.To(int64(65534)),
}
