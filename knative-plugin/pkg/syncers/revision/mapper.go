package revision

import (
	basecontext "context"
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster-sdk/clienthelper"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	IndexByConfiguration = "indexbyconfiguration"
)

func (r *revisionSyncer) RegisterIndices(ctx *context.RegisterContext) error {
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &ksvcv1.Configuration{}, IndexByConfiguration, func(rawObj client.Object) []string {
		return revisionNamesFromConfiguration(rawObj.(*ksvcv1.Configuration))
	})

	if err != nil {
		return err
	}

	return r.NamespacedTranslator.RegisterIndices(ctx)
}

func (r *revisionSyncer) ModifyController(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	builder = builder.Watches(&source.Kind{
		Type: &ksvcv1.Configuration{},
	}, handler.EnqueueRequestsFromMapFunc(mapConfigs))

	return builder, nil
}

func mapConfigs(obj client.Object) []reconcile.Request {
	config, ok := obj.(*ksvcv1.Configuration)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := revisionNamesFromConfiguration(config)
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

func revisionNamesFromConfiguration(config *ksvcv1.Configuration) []string {
	revisions := []string{}

	vNamespace := config.GetNamespace()

	re := regexp.MustCompile(fmt.Sprintf(`(?m)(?P<name>\w+)-x-%s-x-(?P<pNs>\w+)-(?P<rNo>\d+)`, vNamespace))

	if config.Status.LatestCreatedRevisionName != "" {
		matches := re.FindStringSubmatch(config.Status.LatestCreatedRevisionName)
		pNamespace := matches[re.SubexpIndex("pNs")]
		name := matches[re.SubexpIndex("name")]
		revNo := matches[re.SubexpIndex("rNo")]

		revisions = append(revisions, pNamespace+"/"+config.Status.LatestCreatedRevisionName)
		// revisions = append(revisions, vNamespace+"/"+config.Status.LatestCreatedRevisionName)

		revisions = append(revisions, fmt.Sprintf("%s/%s-%s", vNamespace, name, revNo))
		// revisions = append(revisions, pNamespace+"/"+config.Status.LatestReadyRevisionName)
	}

	if config.Status.LatestReadyRevisionName != "" {
		matches := re.FindStringSubmatch(config.Status.LatestReadyRevisionName)
		pNamespace := matches[re.SubexpIndex("pNs")]
		name := matches[re.SubexpIndex("name")]
		revNo := matches[re.SubexpIndex("rNo")]

		revisions = append(revisions, pNamespace+"/"+config.Status.LatestReadyRevisionName)
		// revisions = append(revisions, vNamespace+"/"+config.Status.LatestReadyRevisionName)
		revisions = append(revisions, fmt.Sprintf("%s/%s-%s", vNamespace, name, revNo))
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

	// configList := ksvcv1.ConfigurationList{}
	// listErr := r.virtualClient.List(basecontext.TODO(), &configList)
	// if listErr != nil {
	// 	fmt.Println(listErr)
	// }

	err := clienthelper.GetByIndex(basecontext.TODO(), r.virtualClient, vConfig, IndexByConfiguration, pObj.GetNamespace()+"/"+pObj.GetName())
	if err == nil && vConfig.Name != "" {
		if vConfig.Status.LatestCreatedRevisionName == pObj.GetName() ||
			vConfig.Status.LatestReadyRevisionName == pObj.GetName() {

			vNamespace := vConfig.Namespace
			re := regexp.MustCompile(fmt.Sprintf(`(?m)(?P<name>\w+)-x-%s-x-(?P<pNs>\w+)-(?P<rNo>\d+)`, vNamespace))
			matches := re.FindStringSubmatch(pObj.GetName())
			revName := matches[re.SubexpIndex("name")]
			revNo := matches[re.SubexpIndex("rNo")]

			return types.NamespacedName{
				Name:      fmt.Sprintf("%s-%s", revName, revNo),
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

	re := regexp.MustCompile(`(?m)(?P<revName>[\w|-]+)-(?P<revNo>[\d]+)`)
	matches := re.FindStringSubmatch(req.Name)

	revName := matches[re.SubexpIndex("revName")]
	revNo := matches[re.SubexpIndex("revNo")]

	physicalName := r.NamespacedTranslator.VirtualToPhysical(types.NamespacedName{
		Name:      revName,
		Namespace: req.Namespace,
	}, vObj)

	physicalName.Name += "-" + revNo

	return physicalName
}
