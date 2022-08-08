package revision

import (
	"fmt"

	"github.com/loft-sh/vcluster-sdk/syncer"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MapperFunc func(obj client.Object) []string

func (r *revisionSyncer) AddReverseMapper(ctx *context.RegisterContext, obj client.Object, indexName string, mapper MapperFunc) error {
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

	return nil
}

func (r *revisionSyncer) GetReverseMapper() syncer.MapperConfig {
	return r.mapperConfig
}

type MapperConfig struct {
	ExtraIndices IndexFunc
	// ControllerModifiers []ControllerModifier
}

type IndexFunc func(ctx *context.RegisterContext) error

type ControllerModifier func(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error)

type ReverseMapper map[syncer.Syncer]MapperConfig
