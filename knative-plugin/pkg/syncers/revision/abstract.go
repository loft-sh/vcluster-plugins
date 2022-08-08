package revision

import (
	"fmt"

	"github.com/loft-sh/vcluster-sdk/syncer"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type MapperFunc func(obj client.Object) []string
type Enqueuer func(obj client.Object) []reconcile.Request

func (r *revisionSyncer) AddReverseMapper(
	ctx *context.RegisterContext,
	obj client.Object,
	indexName string,
	mapper MapperFunc,
	enqueuer Enqueuer) error {

	// RegisterIndices
	r.mapperConfig.ExtraIndices = func(ctx *context.RegisterContext) error {
		fmt.Println(">>>>>>>>>> Calling indexer function <<<<<<<<")
		err := ctx.VirtualManager.GetFieldIndexer().
			IndexField(ctx.Context, obj, indexName, client.IndexerFunc(mapper))
		if err != nil {
			return err
		}

		return nil
	}

	// add watcher
	r.mapperConfig.ExtraWatchers = append(r.mapperConfig.ExtraWatchers,
		func(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
			builder = builder.Watches(&source.Kind{
				Type: obj,
			}, handler.EnqueueRequestsFromMapFunc(handler.MapFunc(enqueuer)))

			return builder, nil
		})

	return nil
}

func (r *revisionSyncer) GetReverseMapper() syncer.MapperConfig {
	return r.mapperConfig
}

func (r *revisionSyncer) GetWatchers() []syncer.Watchers {
	return r.mapperConfig.ExtraWatchers
}

type MapperConfig struct {
	ExtraIndices IndexFunc
	// ControllerModifiers []ControllerModifier
}

type IndexFunc func(ctx *context.RegisterContext) error

type ControllerModifier func(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error)

type ReverseMapper map[syncer.Syncer]MapperConfig
