package routes

import (
	"github.com/loft-sh/vcluster-sdk/clienthelper"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (r *routeSyncer) findParentObject(ctx *context.SyncContext, obj client.Object) (client.Object, error) {
	klog.Info("extracting from indexer")

	parent := &ksvcv1.Service{}

	owners := obj.GetOwnerReferences()
	for _, owner := range owners {
		err := clienthelper.GetByIndex(ctx.Context, ctx.VirtualClient, parent, translator.IndexByPhysicalName, owner.Name)
		if err != nil {
			klog.Errorf("error while getting by index %v", err)
			return nil, err
		} else {
			klog.Infof("found owner for virtual route %s/%s => ksvc:%s/%s",
				obj.GetNamespace(),
				obj.GetName(),
				parent.GetNamespace(),
				parent.GetName(),
			)

			break
		}
	}

	return parent, nil
}

func (r *routeSyncer) ReverseTranslateMetadata(ctx *context.SyncContext, obj, parent client.Object) client.Object {
	pName := r.PhysicalToVirtual(obj)

	newConfig := obj.(*ksvcv1.Route)
	newConfig.ObjectMeta.Name = pName.Name
	newConfig.ObjectMeta.Namespace = pName.Namespace

	// remove resourceVersion and uid
	newConfig.ObjectMeta.ResourceVersion = ""
	newConfig.ObjectMeta.UID = ""

	var controller, bod *bool
	for _, owner := range newConfig.OwnerReferences {
		if owner.Kind == "Service" {
			controller = owner.Controller
			bod = owner.BlockOwnerDeletion
		}

	}

	parentKsvc := parent.(*ksvcv1.Service)

	newConfig.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         parentKsvc.APIVersion,
			Kind:               parentKsvc.Kind,
			Name:               parentKsvc.GetName(),
			UID:                parentKsvc.GetUID(),
			Controller:         controller,
			BlockOwnerDeletion: bod,
		},
	}

	return newConfig
}
