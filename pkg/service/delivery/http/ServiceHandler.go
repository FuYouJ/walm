package http

import (
	"WarpCloud/walm/pkg/k8s"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/http"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	httpUtils "WarpCloud/walm/pkg/util/http"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
)

type ServiceHandler struct {
	k8sCache    k8s.Cache
	k8sOperator k8s.Operator
}

func RegisterServiceHandler(k8sCache k8s.Cache, k8sOperator k8s.Operator) *restful.WebService {
	handler := &ServiceHandler{
		k8sCache:    k8sCache,
		k8sOperator: k8sOperator,
	}

	ws := new(restful.WebService)

	ws.Path(http.ApiV1 + "/service").
		Doc("Kubernetes Service相关操作").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	tags := []string{"service"}

	ws.Route(ws.GET("/{namespace}").To(handler.GetServices).
		Doc("获取Namepace下的所有Service列表").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string")).
		Writes(k8sModel.ServiceList{}).
		Returns(200, "OK", k8sModel.ServiceList{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.GET("/{namespace}/name/{servicename}").To(handler.GetService).
		Doc("获取对应Service的详细信息").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string")).
		Param(ws.PathParameter("servicename", "service名字").DataType("string")).
		Writes(k8sModel.Service{}).
		Returns(200, "OK", k8sModel.Service{}).
		Returns(404, "Not Found", http.ErrorMessageResponse{}).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.DELETE("/{namespace}/name/{servicename}").To(handler.DeleteService).
		Doc("删除一个Service").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string")).
		Param(ws.PathParameter("servicename", "Service名字").DataType("string")).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.POST("/{namespace}").To(handler.CreateService).
		Doc("创建一个Service").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string")).
		Reads(k8sModel.CreateServiceRequestBody{}).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	ws.Route(ws.PUT("/{namespace}").To(handler.UpdateService).
		Doc("更新一个Service").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("namespace", "租户名字").DataType("string")).
		Param(ws.QueryParameter("fullUpdate", "是否全量更新").DataType("boolean").Required(false)).
		Reads(k8sModel.CreateServiceRequestBody{}).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", http.ErrorMessageResponse{}))

	return ws
}

func (handler ServiceHandler) GetServices(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	services, err := handler.k8sCache.ListServices(namespace, "")
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to list services under %s: %s", namespace, err.Error()))
		return
	}
	response.WriteEntity(services)
}

func (handler ServiceHandler) GetService(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("servicename")
	service, err := handler.k8sCache.GetResource(k8sModel.ServiceKind,namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			httpUtils.WriteNotFoundResponse(response, -1, fmt.Sprintf("service %s/%s is not found",namespace, name))
			return
		}
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to get service %s/%s: %s", namespace, name, err.Error()))
		return
	}

	response.WriteEntity(service.(*k8sModel.Service))
}

func (handler ServiceHandler) DeleteService(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	name := request.PathParameter("servicename")
	err := handler.k8sOperator.DeleteService(namespace, name)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to delete secret : %s", err.Error()))
		return
	}
}

func (handler ServiceHandler) CreateService(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	createServiceRequestBody := &k8sModel.CreateServiceRequestBody{}
	err := request.ReadEntity(createServiceRequestBody)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to read request body: %s", err.Error()))
		return
	}

	err = handler.k8sOperator.CreateService(namespace, createServiceRequestBody)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to create service : %s", err.Error()))
		return
	}
}

func (handler ServiceHandler) UpdateService(request *restful.Request, response *restful.Response) {
	namespace := request.PathParameter("namespace")
	createServiceRequestBody := &k8sModel.CreateServiceRequestBody{}
	err := request.ReadEntity(createServiceRequestBody)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to read request body: %s", err.Error()))
		return
	}

	if createServiceRequestBody.ClusterIp != "" {
		httpUtils.WriteErrorResponse(response, - 1, fmt.Sprintf("spec.clusterIP: Invalid value: %s, field is immutable", createServiceRequestBody.ClusterIp))
	}

	fullUpdate, err := httpUtils.GetFullUpdateParam(request)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("query param fullUpdate value is not valid : %s", err.Error()))
		return
	}

	err = handler.k8sOperator.UpdateService(namespace, createServiceRequestBody, fullUpdate)
	if err != nil {
		httpUtils.WriteErrorResponse(response, -1, fmt.Sprintf("failed to update service : %s", err.Error()))
		return
	}
}
