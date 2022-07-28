package configurations

import (
	"github.com/loft-sh/vcluster-sdk/syncer"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	"github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"

	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func New(ctx *context.RegisterContext) syncer.Syncer {
	return &kconfigSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "configuration", &ksvcv1.Configuration{}),
	}
}

type kconfigSyncer struct {
	translator.NamespacedTranslator
}

var _ syncer.Initializer = &kconfigSyncer{}
var _ syncer.UpSyncer = &kconfigSyncer{}

func (k *kconfigSyncer) Init(ctx *context.RegisterContext) error {
	return translate.EnsureCRDFromPhysicalCluster(ctx.Context,
		ctx.PhysicalManager.GetConfig(),
		ctx.VirtualManager.GetConfig(),
		ksvcv1.SchemeGroupVersion.WithKind("Configuration"))
}

// SyncDown defines the action that should be taken by the syncer if a virtual cluster object
// exists, but has no corresponding physical cluster object yet. Typically, the physical cluster
// object would get synced down from the virtual cluster to the host cluster in this scenario.
func (k *kconfigSyncer) SyncDown(ctx *context.SyncContext, vObj client.Object) (ctrl.Result, error) {
	klog.Info("SyncDown called for ", vObj.GetName())

	klog.Infof("Deleting virtual Config Object %s because physical no longer exists", vObj.GetName())
	err := ctx.VirtualClient.Delete(ctx.Context, vObj)
	if err != nil {
		klog.Infof("Error deleting virtual Config object: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (k *kconfigSyncer) Sync(ctx *context.SyncContext, pObj, vObj client.Object) (ctrl.Result, error) {
	klog.Infof("Sync called for Configuration %s : %s", pObj.GetName(), vObj.GetName())

	pConfig := pObj.(*ksvcv1.Configuration)
	vConfig := vObj.(*ksvcv1.Configuration)

	// always treat config values from ksvc as the source of truth
	// hence only sync up the spec
	if !equality.Semantic.DeepEqual(vConfig.Spec, pConfig.Spec) {
		newConfig := vConfig.DeepCopy()
		newConfig.Spec = pConfig.Spec
		klog.Infof("Update virtual kconfig %s:%s, because spec is out of sync", vConfig.Namespace, vConfig.Name)
		err := ctx.VirtualClient.Update(ctx.Context, newConfig)
		if err != nil {
			klog.Errorf("Error updating virtual kconfig spec for %s:%s, %v", vConfig.Namespace, vConfig.Name, err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if !equality.Semantic.DeepEqual(vConfig.Status, pConfig.Status) {
		newConfig := vConfig.DeepCopy()
		newConfig.Status = pConfig.Status
		klog.Infof("Update virtual kconfig %s:%s, because status is out of sync", vConfig.Namespace, vConfig.Name)
		err := ctx.VirtualClient.Status().Update(ctx.Context, newConfig)
		if err != nil {
			klog.Errorf("Error updating virtual kconfig status for %s:%s, %v", vConfig.Namespace, vConfig.Name, err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (k *kconfigSyncer) SyncUp(ctx *context.SyncContext, pObj client.Object) (ctrl.Result, error) {
	klog.Info("SyncUp called for configuration ", pObj.GetName())

	return k.SyncUpCreate(ctx, pObj)
}

func (k *kconfigSyncer) SyncUpCreate(ctx *context.SyncContext, pObj client.Object) (ctrl.Result, error) {
	klog.Infof("SyncUpCreate called for %s:%s", pObj.GetName(), pObj.GetNamespace())
	klog.Info("reverse name should be ", k.PhysicalToVirtual(pObj))

	klog.Info("extracting from indexer")

	parent, err := k.findParentObject(ctx, pObj)
	if err != nil {
		klog.Errorf("no parent found for object %s/%s, %v", pObj.GetNamespace(), pObj.GetName(), err)
		return ctrl.Result{}, err
	}

	newObj := pObj.DeepCopyObject().(client.Object)

	newObj = k.ReverseTranslateMetadata(ctx, newObj, parent)
	// klog.Info(newObj)

	err = ctx.VirtualClient.Create(ctx.Context, newObj)
	if err != nil {
		klog.Errorf("error creating virtual config object %s/%s, %v", newObj.GetNamespace(), newObj.GetName(), err)
		k.NamespacedTranslator.EventRecorder().Eventf(newObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
