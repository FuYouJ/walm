package http

import (
	"k8s.io/klog"
	"strconv"
	"github.com/emicklei/go-restful"
	"net/http"
	httpModel "WarpCloud/walm/pkg/models/http"
	"fmt"
)

func WriteErrorResponse(response *restful.Response, code int, errMsg string) error {
	return response.WriteHeaderAndEntity(http.StatusInternalServerError, httpModel.ErrorMessageResponse{code, errMsg})
}

func WriteWarnResponse(response *restful.Response, code int, warnMsg string) error {
	return response.WriteHeaderAndEntity(http.StatusOK, httpModel.WarnMessageResponse{code, warnMsg})
}

func WriteNotFoundResponse(response *restful.Response, code int, errMsg string) error {
	return response.WriteHeaderAndEntity(http.StatusNotFound, httpModel.ErrorMessageResponse{code, errMsg})
}

func GetDeletePvcsQueryParam(request *restful.Request) (deletePvcs bool, err error) {
	deletePvcsStr := request.QueryParameter("deletePvcs")
	if len(deletePvcsStr) > 0 {
		deletePvcs, err = strconv.ParseBool(deletePvcsStr)
		if err != nil {
			klog.Errorf("failed to parse query parameter deletePvcs %s : %s", deletePvcsStr, err.Error())
			return
		}
	}
	return
}

func GetAsyncQueryParam(request *restful.Request) (async bool, err error) {
	asyncStr := request.QueryParameter("async")
	if len(asyncStr) > 0 {
		async, err = strconv.ParseBool(asyncStr)
		if err != nil {
			klog.Errorf("failed to parse query parameter async %s : %s",asyncStr, err.Error())
			return
		}
	}
	return
}

func GetTimeoutSecQueryParam(request *restful.Request) (timeoutSec int64, err error) {
	timeoutStr := request.QueryParameter("timeoutSec")
	if len(timeoutStr) > 0 {
		timeoutSec, err = strconv.ParseInt(timeoutStr, 10, 64)
		if err != nil {
			klog.Errorf("failed to parse query parameter timeoutSec %s : %s", timeoutStr, err.Error())
			return
		}
		if timeoutSec <0 {
			err = fmt.Errorf("query parameter timeoutSec %s should not be less than zero", timeoutStr)
			klog.Error(err.Error())
			return
		}
	}
	return
}

func GetFullUpdateParam(request *restful.Request) (fullUpdate bool, err error) {
	fullUpdateStr := request.QueryParameter("fullUpdate")
	if len(fullUpdateStr) > 0 {
		fullUpdate, err = strconv.ParseBool(fullUpdateStr)
		if err != nil {
			klog.Errorf("failed to parse query parameter fullUpdate %s : %s",fullUpdateStr, err.Error())
			return
		}
	}
	return
}

func GetUpdateConfigMapParam(request *restful.Request) (updateConfigMap bool, err error) {
	updateConfigMapStr := request.QueryParameter("updateConfigMap")
	if len(updateConfigMapStr) > 0 {
		updateConfigMap, err = strconv.ParseBool(updateConfigMapStr)
		if err != nil {
			klog.Errorf("failed to parse query parameter updateConfigMap %s : %s",updateConfigMapStr, err.Error())
			return
		}
	}
	return
}
