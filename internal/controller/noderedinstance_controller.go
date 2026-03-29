package controller

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	niotav1alpha1 "github.com/godenja/niota-node-red-operator/api/v1alpha1"
)

//go:embed assets/settings.js
var settingsJS string

const (
	nodeRedPort = int32(1880)
	dataPath    = "/data"
)

// NodeRedInstanceReconciler reconciles NodeRedInstance objects.
//
// +kubebuilder:rbac:groups=niota.io,resources=noderedinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=niota.io,resources=noderedinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=niota.io,resources=noderedinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=traefik.io,resources=ingressroutes,verbs=get;list;watch;create;update;patch;delete
type NodeRedInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile implements reconcile.Reconciler.
func (r *NodeRedInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instance := &niotav1alpha1.NodeRedInstance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling NodeRedInstance", "id", instance.Spec.ID)

	steps := []struct {
		name string
		fn   func(context.Context, *niotav1alpha1.NodeRedInstance) error
	}{
		{"OAuthSecret", r.reconcileOAuthSecret},
		{"CredentialSecret", r.reconcileCredentialSecret},
		{"PVC", r.reconcilePVC},
		{"ConfigMap", r.reconcileConfigMap},
		{"Deployment", r.reconcileDeployment},
		{"Service", r.reconcileService},
		{"IngressRoute", r.reconcileIngressRoute},
	}

	for _, step := range steps {
		if err := step.fn(ctx, instance); err != nil {
			logger.Error(err, "reconcile step failed", "step", step.name)
			return ctrl.Result{}, fmt.Errorf("%s: %w", step.name, err)
		}
	}

	// Update status
	patch := client.MergeFrom(instance.DeepCopy())
	instance.Status.Ready = true
	if err := r.Status().Patch(ctx, instance, patch); err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	logger.Info("NodeRedInstance reconciled successfully", "id", instance.Spec.ID)
	return ctrl.Result{}, nil
}

// reconcileOAuthSecret creates or updates the Secret that holds the OAuth2
// client secret sourced from the CR spec.
func (r *NodeRedInstanceReconciler) reconcileOAuthSecret(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Spec.ID + "-oauth",
			Namespace: instance.Namespace,
			Labels:    labelsFor(instance),
		},
		StringData: map[string]string{
			"clientSecret": instance.Spec.OAuth.ClientSecret,
		},
	}
	if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Secret{}
	err := r.Get(ctx, namespacedName(secret), existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, secret)
	}
	if err != nil {
		return err
	}

	// Keep the secret in sync when the CR value changes.
	patch := client.MergeFrom(existing.DeepCopy())
	existing.StringData = secret.StringData
	return r.Patch(ctx, existing, patch)
}

// reconcileCredentialSecret creates the Node-RED credential encryption Secret.
// It is written once and never modified again (Kubernetes immutable Secret).
// This ensures the key is stable across CR updates (e.g. image changes).
// When spec.credentialSecret is set, that value is used (migration path);
// otherwise a random 32-byte key is generated on first creation.
func (r *NodeRedInstanceReconciler) reconcileCredentialSecret(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	name := instance.Spec.ID + "-credential"

	// If the secret already exists, never touch it — it is intentionally immutable.
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: instance.Namespace}, existing)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	credValue := instance.Spec.CredentialSecret
	if credValue == "" {
		b := make([]byte, 32)
		if _, genErr := rand.Read(b); genErr != nil {
			return fmt.Errorf("generating credential secret: %w", genErr)
		}
		credValue = hex.EncodeToString(b)
	}

	immutable := true
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labelsFor(instance),
		},
		Immutable: &immutable,
		StringData: map[string]string{
			"credentialSecret": credValue,
		},
	}
	if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, secret)
}

// reconcilePVC ensures the data PVC exists.
func (r *NodeRedInstanceReconciler) reconcilePVC(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	storageSize := instance.Spec.StorageSize
	if storageSize == "" {
		storageSize = "1Gi"
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Spec.ID,
			Namespace: instance.Namespace,
			Labels:    labelsFor(instance),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}
	if instance.Spec.StorageClass != nil {
		pvc.Spec.StorageClassName = instance.Spec.StorageClass
	}

	if err := controllerutil.SetControllerReference(instance, pvc, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, namespacedName(pvc), existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, pvc)
	}
	// PVCs are immutable after creation; only create, never update.
	return err
}

// reconcileConfigMap creates or updates the settings.js ConfigMap.
func (r *NodeRedInstanceReconciler) reconcileConfigMap(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Spec.ID + "-settings",
			Namespace: instance.Namespace,
			Labels:    labelsFor(instance),
		},
		Data: map[string]string{
			"settings.js": settingsJS,
		},
	}

	if err := controllerutil.SetControllerReference(instance, cm, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, namespacedName(cm), existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, cm)
	}
	if err != nil {
		return err
	}

	patch := client.MergeFrom(existing.DeepCopy())
	existing.Data = cm.Data
	return r.Patch(ctx, existing, patch)
}

// reconcileDeployment creates or updates the Node-RED Deployment.
func (r *NodeRedInstanceReconciler) reconcileDeployment(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	labels := labelsFor(instance)

	image := instance.Spec.Image.Repository + ":" + func() string {
		if instance.Spec.Image.Tag != "" {
			return instance.Spec.Image.Tag
		}
		return "latest"
	}()

	var pullSecrets []corev1.LocalObjectReference
	if instance.Spec.Image.PullSecretName != "" {
		pullSecrets = []corev1.LocalObjectReference{{Name: instance.Spec.Image.PullSecretName}}
	}

	replicas := int32(1)
	envVars := []corev1.EnvVar{
		{Name: "NODE_TLS_REJECT_UNAUTHORIZED", Value: "0"},
		{Name: "NIOTA_COOKIENAME", Value: "njwt_prod"},
		{Name: "VIRTUAL_PATH", Value: "/" + instance.Spec.ID},
		{Name: "OAUTH_CLIENT_ID", Value: instance.Spec.OAuth.ClientID},
		{
			Name: "OAUTH_CLIENT_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.ID + "-oauth"},
					Key: "clientSecret",
				},
			},
		},
		{
			Name: "NODE_RED_CREDENTIAL_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.ID + "-credential"},
					Key: "credentialSecret",
				},
			},
		},
		{Name: "OAUTH_AUTH_URL", Value: instance.Spec.OAuth.AuthURL},
		{Name: "OAUTH_TOKEN_URL", Value: instance.Spec.OAuth.TokenURL},
		{
			Name:  "OAUTH_CALLBACK_URL",
			Value: fmt.Sprintf("https://%s/%s/auth/strategy/callback", instance.Spec.Domain, instance.Spec.ID),
		},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Spec.ID,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ImagePullSecrets: pullSecrets,
					Containers: []corev1.Container{
						{
							Name:  "node-red",
							Image: image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: nodeRedPort, Protocol: corev1.ProtocolTCP},
							},
							Env: envVars,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: dataPath},
								// subPath mount creates a specific file; it coexists
								// with the PVC mount at the parent directory.
								{
									Name:      "settings",
									MountPath: dataPath + "/settings.js",
									SubPath:   "settings.js",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: instance.Spec.ID,
								},
							},
						},
						{
							Name: "settings",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.Spec.ID + "-settings",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, deploy, r.Scheme); err != nil {
		return err
	}

	existing := &appsv1.Deployment{}
	err := r.Get(ctx, namespacedName(deploy), existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, deploy)
	}
	if err != nil {
		return err
	}

	patch := client.MergeFrom(existing.DeepCopy())
	existing.Spec.Template.Spec.Containers = deploy.Spec.Template.Spec.Containers
	existing.Spec.Template.Spec.ImagePullSecrets = deploy.Spec.Template.Spec.ImagePullSecrets
	existing.Spec.Template.Spec.Volumes = deploy.Spec.Template.Spec.Volumes
	return r.Patch(ctx, existing, patch)
}

// reconcileService creates the ClusterIP Service if it does not exist.
func (r *NodeRedInstanceReconciler) reconcileService(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Spec.ID,
			Namespace: instance.Namespace,
			Labels:    labelsFor(instance),
		},
		Spec: corev1.ServiceSpec{
			Selector: labelsFor(instance),
			Ports: []corev1.ServicePort{
				{Port: nodeRedPort, Protocol: corev1.ProtocolTCP},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, svc, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Service{}
	err := r.Get(ctx, namespacedName(svc), existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, svc)
	}
	// Service selector / ports are stable; only create, skip updates.
	return err
}

// reconcileIngressRoute creates or updates the Traefik IngressRoute.
// The resource is handled as unstructured to avoid a hard dependency on
// Traefik's client-go package.
func (r *NodeRedInstanceReconciler) reconcileIngressRoute(ctx context.Context, instance *niotav1alpha1.NodeRedInstance) error {
	matchRule := fmt.Sprintf("Host(`%s`) && PathPrefix(`/%s`)", instance.Spec.Domain, instance.Spec.ID)

	ingressClass := instance.Spec.IngressClass
	if ingressClass == "" {
		ingressClass = "traefik"
	}

	desired := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "IngressRoute",
			"metadata": map[string]interface{}{
				"name":      instance.Spec.ID,
				"namespace": instance.Namespace,
				"labels":    strMapToInterface(labelsFor(instance)),
				"annotations": map[string]interface{}{
					"kubernetes.io/ingress.class": ingressClass,
				},
			},
			"spec": map[string]interface{}{
				"entryPoints": []interface{}{"websecure"},
				"routes": []interface{}{
					map[string]interface{}{
						"match":  matchRule,
						"kind":   "Rule",
						"syntax": "v2",
						"services": []interface{}{
							map[string]interface{}{
								"name": instance.Spec.ID,
								"port": nodeRedPort,
							},
						},
					},
				},
				"tls": map[string]interface{}{
					"secretName": instance.Spec.TLSSecretName,
				},
			},
		},
	}

	// Set owner reference so the IngressRoute is garbage-collected with the CR.
	if err := controllerutil.SetControllerReference(instance, desired, r.Scheme); err != nil {
		return err
	}

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "traefik.io",
		Version: "v1alpha1",
		Kind:    "IngressRoute",
	})

	err := r.Get(ctx, types.NamespacedName{Name: instance.Spec.ID, Namespace: instance.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	patch := client.MergeFrom(existing.DeepCopy())
	existing.Object["spec"] = desired.Object["spec"]
	existing.Object["metadata"].(map[string]interface{})["annotations"] =
		desired.Object["metadata"].(map[string]interface{})["annotations"]
	return r.Patch(ctx, existing, patch)
}

// SetupWithManager registers the controller with the manager.
func (r *NodeRedInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&niotav1alpha1.NodeRedInstance{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// --- helpers -----------------------------------------------------------------

func labelsFor(instance *niotav1alpha1.NodeRedInstance) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "node-red",
		"app.kubernetes.io/instance":   instance.Spec.ID,
		"app.kubernetes.io/managed-by": "niota-node-red-operator",
	}
}

func strMapToInterface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func namespacedName(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
}