package http

import (
	"WarpCloud/walm/pkg/k8s"
	"WarpCloud/walm/pkg/models/http"
	httpUtils "WarpCloud/walm/pkg/util/http"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	tosv1beta1 "github.com/migration/pkg/apis/tos/v1beta1"
)

type CrdHandler struct {
	k8sCache    k8s.Cache
	k8sOperator k8s.Operator
}



func RegisterCrdHandler(k8sCache k8s.Cache, k8sOperator k8s.Operator) *restful.WebService {
	handler := &CrdHandler{
		k8sOperator: k8sOperator,
		k8sCache:    k8sCache,
	}

	ws := new(restful.WebService)

	ws.Path(http.ApiV1+"/crd").
		Doc("Kubernetes Custom Resource Definition相关操作").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	tags := []string{"crd"}


	// crd crd
	ws.Route(ws.GET("/migration").To(handler.ListMigrations).
		Doc("获取所有crd迁移信息列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("labelselector", "节点标签过滤").DataType("string").Required(false)).
		Writes([]tosv1beta1.Mig{}).
		Returns(200, "OK", []tosv1beta1.Mig{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/{namespace}").To(handler.ListMigrationsByNamespace).
		Doc("获取Namespace下的crd迁移信息列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.QueryParameter("labelselector", "节点标签过滤").DataType("string").Required(false)).
		Writes([]tosv1beta1.Mig{}).
		Returns(200, "OK", []tosv1beta1.Mig{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/{namespace}/{mig}").To(handler.GetMigration).
		Doc("获取特定的crd迁移信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.PathParameter("mig", "crd迁移名称").DataType("string").Required(true)).
		Writes(tosv1beta1.Mig{}).
		Returns(200, "Ok", tosv1beta1.Mig{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/migration/pod/{namespace}/name/{pod}/mig/{mig}").To(handler.MigratePod).
		Doc("迁移Pod").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.PathParameter("pod", "pod名字").DataType("string").Required(true)).
		Param(ws.PathParameter("mig", "迁移操作名称").DataType("string").Required(true)).
		Param(ws.QueryParameter("migNamespace", "迁移信息所在Namespace").DataType("string").Required(false)).
		Param(ws.QueryParameter("destHost", "迁移的目标node节点名称").DataType("string").Required(false)).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/migration/node/{srcHost}/mig/{mig}").To(handler.MigrateNode).
		Doc("迁移节点上所有statefulset管理的pod").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("srcHost", "要迁移的node节点名称").DataType("string").Required(true)).
		Param(ws.PathParameter("mig", "迁移操作名称").DataType("string").Required(true)).
		Param(ws.QueryParameter("migNamespace", "迁移信息所在Namespace").DataType("string").Required(false)).
		Param(ws.QueryParameter("destHost", "迁移的目标node节点名称").DataType("string").Required(false)).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	return ws

}


func (handler CrdHandler) ListMigrations(request *restful.Request, response *restful.Response) {

	labelSelectorStr := request.QueryParameter("labelselector")
	migs, err := handler.k8sCache.ListMigrations("", labelSelectorStr)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to list migrations: %s", err.Error()))
		return
	}
	response.WriteEntity(migs)
}

func (handler CrdHandler) ListMigrationsByNamespace(request *restful.Request, response *restful.Response) {
	labelSelectorStr := request.QueryParameter("labelselector")
	namespace := request.PathParameter("namespace")
	migs, err := handler.k8sCache.ListMigrations(namespace, labelSelectorStr)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to list namespace %s migrations: %s",namespace, err.Error()))
		return
	}
	response.WriteEntity(migs)
}

func (handler CrdHandler) GetMigration(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	migName := request.PathParameter("mig")
	mig, err := handler.k8sCache.GetMigration(namespace, migName)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to get migrations %s: %s",migName, err.Error()))
		return
	}
	response.WriteEntity(mig)
}

func (handler CrdHandler) MigratePod(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")
	migName := request.PathParameter("mig")
	migNamespace := request.QueryParameter("migNamespace")
	destHost := request.QueryParameter("destHost")

	err := handler.k8sOperator.MigratePod(namespace, podName, migName, migNamespace, destHost)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to migrate pod : %s", err.Error()))
		return
	}
	return
}

func (handler CrdHandler) MigrateNode(request *restful.Request, response *restful.Response) {
	srcHost := request.PathParameter("srcHost")
	migName := request.PathParameter("mig")
	migNamespace := request.QueryParameter("migNamespace")
	destHost := request.QueryParameter("destHost")

	err := handler.k8sOperator.MigrateNode(srcHost, destHost, migName, migNamespace)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to migrate node: %s", err.Error()))
		return
	}
	return
}

