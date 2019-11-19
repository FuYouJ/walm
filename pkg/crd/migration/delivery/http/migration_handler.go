package http

import (
	"WarpCloud/walm/pkg/k8s"
	"WarpCloud/walm/pkg/models/http"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/setting"
	httpUtils "WarpCloud/walm/pkg/util/http"

	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
)

type CrdHandler struct {
	k8sCache    k8s.Cache
	k8sOperator k8s.Operator
	//migDisableFlag bool
}

type CommonHandler struct {
	innerFunc restful.RouteFunction
}

func (commonHandler CommonHandler) handle(request *restful.Request, response *restful.Response) {
	if setting.Config.CrdConfig != nil && setting.Config.CrdConfig.EnableMigrationCRD {
	} else {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("migration not enabled, check for config"))
		return
	}
	commonHandler.innerFunc(request, response)
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
	ws.Route(ws.GET("/migration").To(CommonHandler{handler.ListMigrations}.handle).
		Doc("获取所有crd迁移信息列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("labelselector", "节点标签过滤").DataType("string").Required(false)).
		Writes(k8sModel.MigList{}).
		Returns(200, "OK", k8sModel.MigList{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/pod/{namespace}/name/{pod}").To(CommonHandler{handler.GetPodMigration}.handle).
		Doc("获取pod迁移信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.PathParameter("pod", "pod名称").DataType("string").Required(true)).
		Returns(200, "Ok", k8sModel.Mig{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.DELETE("/migration/pod/{namespace}/name/{pod}").To(CommonHandler{handler.DeletePodMigration}.handle).
		Doc("删除pod迁移信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Param(ws.PathParameter("pod", "pod名称").DataType("string").Required(true)).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/migration/pod/{namespace}").To(CommonHandler{handler.MigratePod}.handle).
		Doc("迁移pod").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string").Required(true)).
		Reads(k8sModel.PodMigRequest{}).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/migration/node/{node}").To(CommonHandler{handler.GetNodeMigration}.handle).
		Doc("获取node迁移信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("node", "node名字").DataType("string").Required(true)).
		Writes(k8sModel.MigList{}).
		Returns(200, "Ok", k8sModel.MigList{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/migration/node").To(CommonHandler{handler.MigrateNode}.handle).
		Doc("迁移node(所有statefulset管理的pod)").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(k8sModel.NodeMigRequest{}).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	return ws

}

func (handler CrdHandler) ListMigrations(request *restful.Request, response *restful.Response) {

	labelSelectorStr := request.QueryParameter("labelselector")
	migs, err := handler.k8sCache.ListMigrations(labelSelectorStr)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to list migrations: %s", err.Error()))
		return
	}
	response.WriteEntity(migs)
}

func (handler CrdHandler) GetPodMigration(request *restful.Request, response *restful.Response) {

	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")
	name := "mig" + "-" +  namespace + "-" + podName
	mig, err := handler.k8sCache.GetResource(k8sModel.MigKind, "default", name)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to get migration: %s", err.Error()))
		return
	}
	response.WriteEntity(mig)
}


func (handler CrdHandler) DeletePodMigration(request *restful.Request, response *restful.Response) {

	namespace := request.PathParameter("namespace")
	podName := request.PathParameter("pod")

	err := handler.k8sOperator.DeletePodMigration(namespace, podName)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to delete migration: %s", err.Error()))
		return
	}
	return
}

func (handler CrdHandler) MigratePod(request *restful.Request, response *restful.Response) {

	namespace := request.PathParameter("namespace")
	podMig := &k8sModel.PodMigRequest{}
	err := request.ReadEntity(podMig)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to read request body: %s", err.Error()))
		return
	}
	if podMig.PodName == "" {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("name not set in request body"))

	}

	mig := &k8sModel.Mig{
		Meta:     k8sModel.Meta{
			Namespace: "default",
			Name: "mig" + "-" + namespace + "-" + podMig.PodName,
		},
		Labels: podMig.Labels,
		Spec:     k8sModel.MigSpec{
			Namespace: namespace,
			PodName: podMig.PodName,
		},
		DestHost: podMig.DestNode,
	}

	err = handler.k8sOperator.MigratePod(mig)

	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to migrate pod : %s", err.Error()))
		return
	}
	return
}

func (handler CrdHandler) GetNodeMigration(request *restful.Request, response *restful.Response) {

	node := request.PathParameter("node")
	migList, err := handler.k8sCache.GetNodeMigration(node)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to get node migs: %s", err.Error()))
		return
	}
	response.WriteEntity(migList)
}

func (handler CrdHandler) MigrateNode(request *restful.Request, response *restful.Response) {

	nodeMig := &k8sModel.NodeMigRequest{}
	err := request.ReadEntity(nodeMig)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to read request body: %s", err.Error()))
		return
	}
	if nodeMig.NodeName == "" {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("name not set in request body"))

	}
	err = handler.k8sOperator.MigrateNode(nodeMig.NodeName, nodeMig.DestNode)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to migrate node: %s", err.Error()))
		return
	}
	return
}
