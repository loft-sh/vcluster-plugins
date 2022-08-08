package revision

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/loft-sh/vcluster-sdk/clienthelper"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"k8s.io/apimachinery/pkg/runtime"
	ksvcv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	r := gin.Default()
	r.GET("/revision/indexer", func(c *gin.Context) {

	})
}

func (r *revisionSyncer) revisionIndexer(c *gin.Context) {
	registerContext, ok := c.Keys[REGISTER_CONTEXT].(*context.RegisterContext)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "type assertion for register context failed",
		})

		return
	}

	indexer := registerContext.VirtualManager.GetFieldIndexer()
	cache := registerContext.VirtualManager.GetCache()
	informer, err := cache.GetInformer(c.Request.Context(), &ksvcv1.Configuration{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	cl := &ksvcv1.ConfigurationList{}
	matchFields := client.MatchingFields{"indexbyconfiguration": "vcluster/hello-x-default-x-vcluster-00001"}
	err = registerContext.VirtualManager.GetClient().List(c.Request.Context(), cl, matchFields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	rev := &ksvcv1.Revision{}

	c.JSON(http.StatusOK, gin.H{
		"message":     "success",
		"indexer":     indexer,
		"informer":    informer,
		"configList":  cl,
		"matchFields": matchFields,
		"kind":        getKindFromObj(rev, r.virtualClient.Scheme()),
	})
}

func getKindFromObj(obj runtime.Object, scheme *runtime.Scheme) string {
	gvk, err := clienthelper.GVKFrom(obj, scheme)
	if err != nil {
		panic(err)
	}

	kind := gvk.Kind
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>", kind)
	return kind
}
