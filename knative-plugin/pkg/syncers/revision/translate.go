package revision

import (
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *revisionSyncer) ReverseTranslateMetadata(ctx *context.SyncContext, obj, parent client.Object) client.Object {
	rev := obj.(*ksvcv1.Revision)

	// remove resourceVersion and uid
	rev.ObjectMeta.ResourceVersion = ""
	rev.ObjectMeta.UID = ""

	// reset owner references
	// TODO: find and set correct owner references
	rev.OwnerReferences = []metav1.OwnerReference{}

	return rev
}
