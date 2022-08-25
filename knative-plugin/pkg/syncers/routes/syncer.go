package routes

import (
	"github.com/loft-sh/vcluster-sdk/syncer"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	"github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/klog"
	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *context.RegisterContext) syncer.Syncer {
	return &routeSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "route", &ksvcv1.Route{}),
	}
}

type routeSyncer struct {
	translator.NamespacedTranslator
}

var _ syncer.Initializer = &routeSyncer{}

func (r *routeSyncer) Init(ctx *context.RegisterContext) error {
	return translate.EnsureCRDFromPhysicalCluster(ctx.Context,
		ctx.PhysicalManager.GetConfig(),
		ctx.VirtualManager.GetConfig(),
		ksvcv1.SchemeGroupVersion.WithKind("Route"))
}

func (r *routeSyncer) SyncDown(ctx *context.SyncContext, vObj client.Object) (ctrl.Result, error) {
	klog.Info("SyncDown called for route ", vObj.GetName())

	klog.Infof("Deleting virtual route Object %s because physical no longer exists", vObj.GetName())
	err := ctx.VirtualClient.Delete(ctx.Context, vObj)
	if err != nil {
		klog.Infof("Error deleting virtual route object: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *routeSyncer) Sync(ctx *context.SyncContext, pObj, vObj client.Object) (ctrl.Result, error) {
	klog.Infof("Sync called for Route %s : %s", pObj.GetName(), vObj.GetName())

	pRoute := pObj.(*ksvcv1.Route)
	vRoute := vObj.(*ksvcv1.Route)

	// always treat config values from ksvc as the source of truth
	// hence only sync up the spec
	if !equality.Semantic.DeepEqual(vRoute.Spec, pRoute.Spec) {
		newConfig := vRoute.DeepCopy()
		newConfig.Spec = pRoute.Spec
		klog.Infof("Update virtual route %s:%s, because spec is out of sync", vRoute.Namespace, vRoute.Name)
		err := ctx.VirtualClient.Update(ctx.Context, newConfig)
		if err != nil {
			klog.Errorf("Error updating virtual route spec for %s:%s, %v", vRoute.Namespace, vRoute.Name, err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if !equality.Semantic.DeepEqual(vRoute.Status, pRoute.Status) {
		newConfig := vRoute.DeepCopy()
		newConfig.Status = pRoute.Status
		klog.Infof("Update virtual route %s:%s, because status is out of sync", vRoute.Namespace, vRoute.Name)
		err := ctx.VirtualClient.Status().Update(ctx.Context, newConfig)
		if err != nil {
			klog.Errorf("Error updating virtual route status for %s:%s, %v", vRoute.Namespace, vRoute.Name, err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *routeSyncer) SyncUp(ctx *context.SyncContext, pObj client.Object) (ctrl.Result, error) {
	klog.Info("SyncUp called for route ", pObj.GetName())

	return r.SyncUpCreate(ctx, pObj)
}

func (r *routeSyncer) SyncUpCreate(ctx *context.SyncContext, pObj client.Object) (ctrl.Result, error) {
	klog.Infof("SyncUpCreate called for %s:%s", pObj.GetName(), pObj.GetNamespace())
	klog.Info("reverse name should be ", r.PhysicalToVirtual(pObj))

	klog.Info("extracting from indexer")

	parent, err := r.findParentObject(ctx, pObj)
	if err != nil {
		klog.Errorf("no parent found for object %s/%s, %v", pObj.GetNamespace(), pObj.GetName(), err)
		return ctrl.Result{}, err
	}

	newObj := pObj.DeepCopyObject().(client.Object)

	newObj = r.ReverseTranslateMetadata(ctx, newObj, parent)

	err = ctx.VirtualClient.Create(ctx.Context, newObj)
	if err != nil {
		klog.Errorf("error creating virtual route object %s/%s, %v", newObj.GetNamespace(), newObj.GetName(), err)
		r.NamespacedTranslator.EventRecorder().Eventf(newObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
