package revision

import (
	basecontext "context"
	"strings"

	"github.com/loft-sh/vcluster-sdk/clienthelper"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	IndexByConfiguration = "indexbyconfiguration"
)

// since this function is also only spliting on '/' and generating namespaced names
// we can shift it inside the abstraction as well. Probably as a base implementation
// and allowing the plugin developer to override with their custom version
func mapconfigs(ctx *context.RegisterContext, obj client.Object) []reconcile.Request {
	// map configs
	config, ok := obj.(*ksvcv1.Configuration)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := filterRevisionFromConfiguration(ctx.TargetNamespace, config)
	for _, name := range names {
		if name != "" {
			splitted := strings.Split(name, "/")
			if len(splitted) == 2 {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: splitted[0],
						Name:      splitted[1],
					},
				})
			}
		}
	}

	return requests
}

func filterRevisionFromConfiguration(pNamespace string, obj client.Object) []string {
	revisions := []string{}
	config := obj.(*ksvcv1.Configuration)

	if config.Status.LatestCreatedRevisionName != "" {
		revisions = append(revisions, pNamespace+"/"+config.Status.LatestCreatedRevisionName)
	}

	if config.Status.LatestReadyRevisionName != "" {
		revisions = append(revisions, pNamespace+"/"+config.Status.LatestReadyRevisionName)
	}

	// klog.Infof("create revision for config: %s", config.Name)
	// klog.Infof("%v", revisions)
	// klog.Infof("--------------------------------------------")

	return revisions
}

func (r *revisionSyncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	namespacedName := r.NamespacedTranslator.PhysicalToVirtual(pObj)
	if namespacedName.Name != "" {
		return namespacedName
	}

	namespacedName = r.nameByConfiguration(pObj)
	if namespacedName.Name != "" {
		return namespacedName
	}

	return types.NamespacedName{}
}

func (r *revisionSyncer) nameByConfiguration(pObj client.Object) types.NamespacedName {
	vConfig := &ksvcv1.Configuration{}

	klog.Infof("getting name by configuration for revision %s/%s", pObj.GetNamespace(), pObj.GetName())

	err := clienthelper.GetByIndex(basecontext.TODO(), r.virtualClient, vConfig, IndexByConfiguration, pObj.GetNamespace()+"/"+pObj.GetName())
	if err == nil && vConfig.Name != "" {
		if vConfig.Status.LatestCreatedRevisionName == pObj.GetName() ||
			vConfig.Status.LatestReadyRevisionName == pObj.GetName() {

			klog.Infof("matched for config: %s", vConfig.Name)
			vNamespace := vConfig.Namespace

			// last five digits are revision suffix
			revName := pObj.GetName()
			revSuffix := revName[len(revName)-6:]

			// a better approach will be to have a function like translate.SafeTrimN or something similar
			// that makes sure to hash the string if longer than N chars. This would be helpful in cases
			// where we know last X chars are always to stay and hashing can be applied on the left out
			// prefix part. For eg. in this case, we know for sure that knative controllers trim the excess
			// name with hash strategy and always append the last 6 digits of rev num after that (-12345)
			var name string
			if len(vConfig.Name) > 57 {
				name = vConfig.Name[:57] + revSuffix
				// safeVConfigName := translate.SafeConcatName(vConfig.Name[:57], vConfig.Name[57:])
			} else {
				name = vConfig.Name + revSuffix
			}

			klog.Infof("safe concat name should be %s", name)

			key := types.NamespacedName{
				Name:      name,
				Namespace: vNamespace,
			}

			// register this in the namecache
			r.nameCache[key] = types.NamespacedName{
				Namespace: pObj.GetNamespace(),
				Name:      pObj.GetName(),
			}

			return key
		}
	}

	return types.NamespacedName{}
}

func (r *revisionSyncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	// revision name would be of the form
	// <name>-<xxxxx> (5-digit revision number)
	// the corresponding physical name should be of the form
	// <name>-x-<virtual_namespace>-x-<physical_namespace>-<xxxxx>
	// example: default/hello-00001
	// should translate to vcluster/hello-x-default-x-vcluster-00001

	// lookup in the nameCache
	return r.nameCache[req]
}
