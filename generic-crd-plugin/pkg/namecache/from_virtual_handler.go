package namecache

import (
	"github.com/loft-sh/vcluster-generic-crd-plugin/pkg/config"
	"github.com/loft-sh/vcluster-generic-crd-plugin/pkg/patches"
	"github.com/loft-sh/vcluster-sdk/translate"
	"github.com/pkg/errors"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fromVirtualClusterCacheHandler struct {
	gvk       schema.GroupVersionKind
	mapping   *config.FromVirtualCluster
	nameCache *nameCache
}

func (c *fromVirtualClusterCacheHandler) OnAdd(obj interface{}) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if ok {
		newMappings, err := c.mappingsFromVirtualObject(unstructuredObj, c.mapping)
		if err == nil {
			c.nameCache.exchangeMapping(&virtualObject{
				GVK:         c.gvk,
				VirtualName: unstructuredObj.GetNamespace() + "/" + unstructuredObj.GetName(),
				Mappings:    newMappings,
			})
		}
	}
}

func (c *fromVirtualClusterCacheHandler) OnUpdate(oldObj, newObj interface{}) {
	unstructuredObj, ok := newObj.(*unstructured.Unstructured)
	if ok {
		newMappings, err := c.mappingsFromVirtualObject(unstructuredObj, c.mapping)
		if err == nil {
			c.nameCache.exchangeMapping(&virtualObject{
				GVK:         c.gvk,
				VirtualName: unstructuredObj.GetNamespace() + "/" + unstructuredObj.GetName(),
				Mappings:    newMappings,
			})
		}
	}
}

func (c *fromVirtualClusterCacheHandler) OnDelete(obj interface{}) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if ok {
		c.nameCache.removeMapping(&virtualObject{
			GVK:         c.gvk,
			VirtualName: unstructuredObj.GetNamespace() + "/" + unstructuredObj.GetName(),
		})
	}
}

func (c *fromVirtualClusterCacheHandler) mappingsFromVirtualObject(obj *unstructured.Unstructured, mappingConfig *config.FromVirtualCluster) ([]mapping, error) {
	mappings := []mapping{}
	if mappingConfig.NameMapping.RewriteName == config.RewriteNameTypeFromVirtualToHostNamespace {
		mappings = append(mappings, mapping{
			VirtualName: obj.GetNamespace() + "/" + obj.GetName(),
			HostName:    translate.PhysicalName(obj.GetName(), obj.GetNamespace()),
		})
	}

	for _, p := range mappingConfig.Patches {
		if p.Type != config.PatchTypeRewriteNameFromVirtualToHostNamespace {
			continue
		}

		node, err := patches.NewJSONNode(obj.Object)
		if err != nil {
			return nil, err
		}

		path, err := yamlpath.NewPath(p.Path)
		if err != nil {
			return nil, errors.Wrapf(err, "compile path %s", p.Path)
		}

		matches, err := path.Find(node)
		if err != nil {
			return nil, err
		}

		for _, m := range matches {
			if m.Kind == yaml.ScalarNode {
				mappings = append(mappings, mapping{
					VirtualName: obj.GetNamespace() + "/" + m.Value,
					HostName:    translate.PhysicalName(m.Value, obj.GetNamespace()),
				})
			}
		}
	}

	return mappings, nil
}
