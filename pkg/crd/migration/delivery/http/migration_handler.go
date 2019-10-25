package http

import (
	"WarpCloud/walm/pkg/k8s"
	"WarpCloud/walm/pkg/models/http"
	httpUtils "WarpCloud/walm/pkg/util/http"
	k8sModel "WarpCloud/walm/pkg/models/k8s"

	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
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
		Writes(k8sModel.MigList{}).
		Returns(200, "OK", k8sModel.MigList{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/{namespace}").To(handler.ListMigrationsByNamespace).
		Doc("获取Namespace下的crd迁移信息列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.QueryParameter("labelselector", "节点标签过滤").DataType("string").Required(false)).
		Writes(k8sModel.MigList{}).
		Returns(200, "OK", k8sModel.MigList{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/pod/{namespace}/name/{mig}").To(handler.GetPodMigration).
		Doc("获取pod迁移信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.PathParameter("mig", "crd迁移名称").DataType("string").Required(true)).
		Writes(k8sModel.Mig{}).
		Returns(200, "Ok", k8sModel.Mig{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/migration/pod").To(handler.MigratePod).
		Doc("迁移pod").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(k8sModel.Mig{}).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/node/{namespace}/name/{mig}").To(handler.GetNodeMigration).
		Doc("获取节点迁移信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.PathParameter("mig", "crd迁移名称").DataType("string").Required(true)).
		Writes(k8sModel.MigList{}).
		Returns(200, "Ok", k8sModel.MigList{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/migration/node").To(handler.MigrateNode).
		Doc("迁移节点上所有statefulset管理的pod").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(k8sModel.Mig{}).
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
	migList, err := handler.k8sCache.ListMigrations(namespace, labelSelectorStr)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to list namespace %s migrations: %s",namespace, err.Error()))
		return
	}
	response.WriteEntity(migList)
}

func (handler CrdHandler) GetPodMigration(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	migName := request.PathParameter("mig")
	mig, err := handler.k8sCache.GetResource(k8sModel.MigKind, namespace, migName)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to get migrations %s: %s",migName, err.Error()))
		return
	}
	response.WriteEntity(mig)
}

func (handler CrdHandler) MigratePod(request *restful.Request, response *restful.Response) {

	migParams := new(k8sModel.Mig)
	err := request.ReadEntity(&migParams)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to read mig params : %s", err.Error()))
		return
	}

	if migParams.Spec.Namespace == "" || migParams.Spec.PodName == ""{
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("both spec.namespace and spec.podname must be set"))
		return
	}

	err = handler.k8sOperator.MigratePod(migParams.Spec.Namespace, migParams.Spec.PodName, migParams, false)

	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to migrate pod : %s", err.Error()))
		return
	}
	return
}

func (handler CrdHandler) GetNodeMigration(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	migName := request.PathParameter("mig")
	migList, err := handler.k8sCache.GetNodeMigration(namespace, migName)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to get node migs: %s", err.Error()))
		return
	}
	response.WriteEntity(migList)
}

func (handler CrdHandler) MigrateNode(request *restful.Request, response *restful.Response) {
	migParams := new(k8sModel.Mig)
	err := request.ReadEntity(&migParams)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to read mig params : %s", err.Error()))
		return
	}
	if migParams.SrcHost == "" {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("SrcHost must be set"))
		return
	}
	err = handler.k8sOperator.MigrateNode(migParams)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to migrate node: %s", err.Error()))
		return
	}
	return
}
