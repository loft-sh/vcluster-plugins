package revision

import (
	basecontext "context"
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster-sdk/clienthelper"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/translate"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	IndexByConfiguration = "indexbyconfiguration"
)

// func (r *revisionSyncer) RegisterIndices(ctx *context.RegisterContext) error {
// 	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &ksvcv1.Configuration{}, IndexByConfiguration, func(rawObj client.Object) []string {
// 		return revisionNamesFromConfiguration(ctx.TargetNamespace, rawObj.(*ksvcv1.Configuration))
// 	})

// 	if err != nil {
// 		return err
// 	}

// 	return r.NamespacedTranslator.RegisterIndices(ctx)
// }

// func (r *revisionSyncer) ModifyController(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
// 	builder = builder.Watches(&source.Kind{
// 		Type: &ksvcv1.Configuration{},
// 	}, handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
// 		return mapconfigs(ctx, obj)
// 	}))

// 	return builder, nil
// }

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

	err := clienthelper.GetByIndex(basecontext.TODO(), r.virtualClient, vConfig, IndexByConfiguration, pObj.GetNamespace()+"/"+pObj.GetName())
	if err == nil && vConfig.Name != "" {
		if vConfig.Status.LatestCreatedRevisionName == pObj.GetName() ||
			vConfig.Status.LatestReadyRevisionName == pObj.GetName() {

			vNamespace := vConfig.Namespace
			pConfigName := translate.PhysicalName(vConfig.Name, vConfig.Namespace)
			revSuffix := strings.TrimPrefix(pObj.GetName(), pConfigName)

			return types.NamespacedName{
				Name:      vConfig.Name + revSuffix,
				Namespace: vNamespace,
			}
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

	fmt.Println("********", req, "*********")
	fmt.Println(vObj)
	fmt.Println(strings.Repeat("*", 100))

	re := regexp.MustCompile(`(?m)(?P<revName>[\w|-]+)-(?P<revNo>[\d]+)`)
	matches := re.FindStringSubmatch(req.Name)

	revName := matches[re.SubexpIndex("revName")]
	revNo := matches[re.SubexpIndex("revNo")]

	physicalName := r.NamespacedTranslator.VirtualToPhysical(types.NamespacedName{
		Name:      revName,
		Namespace: req.Namespace,
	}, vObj)

	physicalName.Name += "-" + revNo

	fmt.Println("physical name:", physicalName)
	fmt.Println(strings.Repeat("=", 100))
	return physicalName
}
