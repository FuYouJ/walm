package http

import (
	"WarpCloud/walm/pkg/models/common"
	errorModel "WarpCloud/walm/pkg/models/error"
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/release/mocks"
	"WarpCloud/walm/test/e2e/framework"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"path/filepath"
	"testing"
)

func TestReleaseHandler_DeleteRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}
	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DeleteRelease", "testns", "testname", false, false, int64(0)).Return(nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DeleteRelease", "testns", "testname", false, false, int64(0)).Return(errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DeleteRelease", "testns", "testname", true, true, int64(60)).Return(nil)
			},
			queryUrl:   "?deletePvcs=true&async=true&timeoutSec=60",
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?deletePvcs=notvalid&async=true&timeoutSec=60",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?deletePvcs=true&async=notvalid&timeoutSec=60",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?deletePvcs=true&async=true&timeoutSec=notvalid",
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname" + test.queryUrl

		httpRequest, _ := http.NewRequest("DELETE", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, httpWriter.Code, test.statusCode)
	}
}

func TestReleaseHandler_InstallRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		body       interface{}
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil), false, int64(0)).Return(nil)
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil), false, int64(0)).Return(errors.New(""))
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil), true, int64(60)).Return(nil)
			},
			queryUrl:   "?async=true&timeoutSec=60",
			body:       release.ReleaseRequestV2{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=notvalid&timeoutSec=60",
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=true&timeoutSec=notvalid",
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns" + test.queryUrl

		bodyBytes, err := json.Marshal(test.body)
		assert.IsType(t, nil, err)

		httpRequest, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_InstallReleaseWithChart(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	currentFilePath, err := framework.GetCurrentFilePath()
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		initMock    func()
		chartPath   string
		body        string
		releaseName string
		statusCode  int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  currentFilePath,
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything, false, int64(0)).Return(nil)
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything, false, int64(0)).Return(errors.New(""))
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/withchart"

		httpRequest, _ := http.NewRequest("POST", url, nil)
		httpRequest.Header.Set("Content-Type", "multipart/form-data")

		chartBytes := []byte{}
		var err error
		if test.chartPath != "" {
			chartBytes, err = ioutil.ReadFile(test.chartPath)
			if err != nil {
				t.Fatal(err.Error())
			}
		}

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		{

			part, err := w.CreateFormFile("chart", "my-chart.tgz")
			if err != nil {
				t.Fatalf("CreateFormFile: %v", err)
			}
			part.Write(chartBytes)

			err = w.WriteField("release", test.releaseName)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.releaseName))

			err = w.WriteField("body", test.body)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.body))

			err = w.Close()
			if err != nil {
				t.Fatalf("Close: %v", err)
			}
		}

		r := multipart.NewReader(&b, w.Boundary())
		httpRequest.MultipartForm, err = r.ReadForm(0)
		if err != nil {
			t.Fatal(err.Error())
		}

		httpRequest.Form = neturl.Values(map[string][]string{"body": {test.body}, "release": {test.releaseName}})

		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)

		mockUseCase.AssertExpectations(t)
	}
}

func TestReleaseHandler_UpgradeRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		body       interface{}
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil), false, int64(0)).Return(nil)
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil), false, int64(0)).Return(errors.New(""))
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil), true, int64(60)).Return(nil)
			},
			queryUrl:   "?async=true&timeoutSec=60",
			body:       release.ReleaseRequestV2{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=notvalid&timeoutSec=60",
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=true&timeoutSec=notvalid",
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns" + test.queryUrl

		bodyBytes, err := json.Marshal(test.body)
		assert.IsType(t, nil, err)

		httpRequest, _ := http.NewRequest("PUT", url, bytes.NewBuffer(bodyBytes))
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_UpgradeReleaseWithChart(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	currentFilePath, err := framework.GetCurrentFilePath()
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		initMock    func()
		chartPath   string
		body        string
		releaseName string
		statusCode  int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  currentFilePath,
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything, false, int64(0)).Return(nil)
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("InstallUpgradeRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything, false, int64(0)).Return(errors.New(""))
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/withchart"

		httpRequest, _ := http.NewRequest("PUT", url, nil)
		httpRequest.Header.Set("Content-Type", "multipart/form-data")

		chartBytes := []byte{}
		var err error
		if test.chartPath != "" {
			chartBytes, err = ioutil.ReadFile(test.chartPath)
			if err != nil {
				t.Fatal(err.Error())
			}
		}

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		{

			part, err := w.CreateFormFile("chart", "my-chart.tgz")
			if err != nil {
				t.Fatalf("CreateFormFile: %v", err)
			}
			part.Write(chartBytes)

			err = w.WriteField("release", test.releaseName)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.releaseName))

			err = w.WriteField("body", test.body)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.body))

			err = w.Close()
			if err != nil {
				t.Fatalf("Close: %v", err)
			}
		}

		r := multipart.NewReader(&b, w.Boundary())
		httpRequest.MultipartForm, err = r.ReadForm(0)
		if err != nil {
			t.Fatal(err.Error())
		}

		httpRequest.Form = neturl.Values(map[string][]string{"body": {test.body}, "release": {test.releaseName}})

		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)

		mockUseCase.AssertExpectations(t)
	}
}

func TestReleaseHandler_ListRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "", "").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "", "").Return(nil, nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "", "test=true").Return(nil, errors.New(""))
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "", "test=true").Return(nil, nil)
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/" + test.queryUrl

		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_ListReleaseByNamespace(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "testns", "").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "testns", "").Return(nil, nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "testns", "test=true").Return(nil, errors.New(""))
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "testns", "test=true").Return(nil, nil)
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns" + test.queryUrl

		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_GetRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetRelease", "testns", "testname").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetRelease", "testns", "testname").Return(nil, errorModel.NotFoundError{})
			},
			statusCode: 404,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetRelease", "testns", "testname").Return(nil, nil)
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname"

		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_DryRunRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		body       interface{}
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DryRunRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil)).Return(nil, nil)
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DryRunRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil)).Return(nil, errors.New(""))
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/dryrun"

		bodyBytes, err := json.Marshal(test.body)
		assert.IsType(t, nil, err)

		httpRequest, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_DryRunReleaseWithChart(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	currentFilePath, err := framework.GetCurrentFilePath()
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		initMock    func()
		chartPath   string
		body        string
		releaseName string
		statusCode  int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  currentFilePath,
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DryRunRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything).Return(nil, nil)
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("DryRunRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything).Return(nil, errors.New(""))
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/dryrun/withchart"

		httpRequest, _ := http.NewRequest("POST", url, nil)
		httpRequest.Header.Set("Content-Type", "multipart/form-data")

		chartBytes := []byte{}
		var err error
		if test.chartPath != "" {
			chartBytes, err = ioutil.ReadFile(test.chartPath)
			if err != nil {
				t.Fatal(err.Error())
			}
		}

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		{

			part, err := w.CreateFormFile("chart", "my-chart.tgz")
			if err != nil {
				t.Fatalf("CreateFormFile: %v", err)
			}
			part.Write(chartBytes)

			err = w.WriteField("release", test.releaseName)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.releaseName))

			err = w.WriteField("body", test.body)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.body))

			err = w.Close()
			if err != nil {
				t.Fatalf("Close: %v", err)
			}
		}

		r := multipart.NewReader(&b, w.Boundary())
		httpRequest.MultipartForm, err = r.ReadForm(0)
		if err != nil {
			t.Fatal(err.Error())
		}

		httpRequest.Form = neturl.Values(map[string][]string{"body": {test.body}, "release": {test.releaseName}})

		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)

		mockUseCase.AssertExpectations(t)
	}
}

func TestReleaseHandler_ComputeResourcesByDryRunRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		body       interface{}
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ComputeResourcesByDryRunRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil)).Return(nil, nil)
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ComputeResourcesByDryRunRelease", "testns", &release.ReleaseRequestV2{}, ([]*common.BufferedFile)(nil)).Return(nil, errors.New(""))
			},
			body:       release.ReleaseRequestV2{},
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/dryrun/resources"

		bodyBytes, err := json.Marshal(test.body)
		assert.IsType(t, nil, err)

		httpRequest, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_ComputeResourcesByDryRunReleaseWithChart(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	currentFilePath, err := framework.GetCurrentFilePath()
	if err != nil {
		t.Fatal(err.Error())
	}

	tests := []struct {
		initMock    func()
		chartPath   string
		body        string
		releaseName string
		statusCode  int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  currentFilePath,
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			chartPath:  filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ComputeResourcesByDryRunRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything).Return(nil, nil)
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ComputeResourcesByDryRunRelease", "testns",
					&release.ReleaseRequestV2{ReleaseRequest: release.ReleaseRequest{Name: "testname"}},
					mock.Anything).Return(nil, errors.New(""))
			},
			chartPath:   filepath.Join(filepath.Dir(currentFilePath), "../../../../test/resources/helm/tomcat-0.2.0.tgz"),
			body:        "{}",
			releaseName: "testname",
			statusCode:  500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/dryrun/withchart/resources"

		httpRequest, _ := http.NewRequest("POST", url, nil)
		httpRequest.Header.Set("Content-Type", "multipart/form-data")

		chartBytes := []byte{}
		var err error
		if test.chartPath != "" {
			chartBytes, err = ioutil.ReadFile(test.chartPath)
			if err != nil {
				t.Fatal(err.Error())
			}
		}

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		{

			part, err := w.CreateFormFile("chart", "my-chart.tgz")
			if err != nil {
				t.Fatalf("CreateFormFile: %v", err)
			}
			part.Write(chartBytes)

			err = w.WriteField("release", test.releaseName)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.releaseName))

			err = w.WriteField("body", test.body)
			if err != nil {
				t.Fatalf("WriteField: %v", err)
			}
			part.Write([]byte(test.body))

			err = w.Close()
			if err != nil {
				t.Fatalf("Close: %v", err)
			}
		}

		r := multipart.NewReader(&b, w.Boundary())
		httpRequest.MultipartForm, err = r.ReadForm(0)
		if err != nil {
			t.Fatal(err.Error())
		}

		httpRequest.Form = neturl.Values(map[string][]string{"body": {test.body}, "release": {test.releaseName}})

		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)

		mockUseCase.AssertExpectations(t)
	}
}

func TestReleaseHandler_PauseRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("PauseOrRecoverRelease", "testns", "testname", false, int64(0), true).Return(nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("PauseOrRecoverRelease", "testns", "testname", false, int64(0), true).Return(errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("PauseOrRecoverRelease", "testns", "testname", true, int64(60), true).Return(nil)
			},
			queryUrl:   "?async=true&timeoutSec=60",
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=notvalid&timeoutSec=60",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=true&timeoutSec=notvalid",
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname/pause" + test.queryUrl

		httpRequest, _ := http.NewRequest("POST", url, nil)
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_RecoverRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("PauseOrRecoverRelease", "testns", "testname", false, int64(0), false).Return(nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("PauseOrRecoverRelease", "testns", "testname", false, int64(0), false).Return(errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("PauseOrRecoverRelease", "testns", "testname", true, int64(60), false).Return(nil)
			},
			queryUrl:   "?async=true&timeoutSec=60",
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=notvalid&timeoutSec=60",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
			},
			queryUrl:   "?async=true&timeoutSec=notvalid",
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname/recover" + test.queryUrl

		httpRequest, _ := http.NewRequest("POST", url, nil)
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_RestartRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}
	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("RestartRelease", "testns", "testname").Return(nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("RestartRelease", "testns", "testname").Return(errors.New(""))
			},
			statusCode: 500,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname/restart" + test.queryUrl

		httpRequest, _ := http.NewRequest("POST", url, nil)
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_UpdateReleaseIngress(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		body       interface{}
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("UpdateReleaseIngress", "test-ns", "test-name", "test-ingress", &k8s.IngressRequestBody{}).Return(nil)
			},
			body:       k8s.IngressRequestBody{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("UpdateReleaseIngress", "test-ns", "test-name", "test-ingress", &k8s.IngressRequestBody{}).Return(errors.New(""))
			},
			body:       k8s.IngressRequestBody{},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("UpdateReleaseIngress", "test-ns", "test-name", "test-ingress", &k8s.IngressRequestBody{
					Host:        "happy.k8s.io",
					Annotations: map[string]string{"test1": "test2"},
					Path:        "/*",
				}).Return(nil)
			},
			body: k8s.IngressRequestBody{
				Host:        "happy.k8s.io",
				Annotations: map[string]string{"test1": "test2"},
				Path:        "/*",
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/test-ns" + "/name" + "/test-name" + "/ingresses" + "/test-ingress"

		bodyBytes, err := json.Marshal(test.body)
		assert.IsType(t, nil, err)

		httpRequest, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_UpdateReleaseConfigMap(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		body       interface{}
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
			},
			body:       "notvalid",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("UpdateReleaseConfigMap", "test-ns", "test-name", "test-configmap", &k8s.ConfigMapRequestBody{}).Return(nil)
			},
			body:       k8s.ConfigMapRequestBody{},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("UpdateReleaseConfigMap", "test-ns", "test-name", "test-configmap", &k8s.ConfigMapRequestBody{}).Return(errors.New(""))
			},
			body:       k8s.ConfigMapRequestBody{},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("UpdateReleaseConfigMap", "test-ns", "test-name", "test-configmap", &k8s.ConfigMapRequestBody{
					Data: map[string]string{"test1": "test2"},
				}).Return(nil)
			},
			body: k8s.ConfigMapRequestBody{
				Data: map[string]string{"test1": "test2"},
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/test-ns" + "/name" + "/test-name" + "/configmaps" + "/test-configmap"

		bodyBytes, err := json.Marshal(test.body)
		assert.IsType(t, nil, err)

		httpRequest, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_GetReleaseEvents(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetReleaseEvents", "testns", "testname").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetReleaseEvents", "testns", "testname").Return(nil, errorModel.NotFoundError{})
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetReleaseEvents", "testns", "testname").Return(nil, nil)
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname/events"
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_GetBackUpRelease(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetBackUpRelease", "testns", "testname").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetBackUpRelease", "testns", "testname").Return(nil, errorModel.NotFoundError{})
			},
			statusCode: 404,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetBackUpRelease", "testns", "testname").Return(nil, nil)
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/name/testname/backup"
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_ListBackUpReleaseByNamespace(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListBackUpReleases", "testns").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListBackUpReleases", "testns").Return(nil, errorModel.NotFoundError{})
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListBackUpReleases", "testns").Return(nil, nil)
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/testns/backup"
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_ListBackUpReleases(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListBackUpReleases", "").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListBackUpReleases", "").Return(nil, errorModel.NotFoundError{})
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListBackUpReleases", "").Return(nil, nil)
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/backup"
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_GetReleaseConfig(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetRelease", "testns", "testname").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetRelease", "testns", "testname").Return(nil, errorModel.NotFoundError{})
			},
			statusCode: 404,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("GetRelease", "testns", "testname").Return(&release.ReleaseInfoV2{}, nil)
			},
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/config/testns/name/testname"
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_ListReleaseConfig(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "", "").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "", "").Return([]*release.ReleaseInfoV2{}, nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "", "test=true").Return(nil, errors.New(""))
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "", "test=true").Return([]*release.ReleaseInfoV2{}, nil)
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/config/" + test.queryUrl
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}

func TestReleaseHandler_ListReleaseConfigByNamespace(t *testing.T) {
	var mockUseCase *mocks.UseCase
	var mockReleaseHandler ReleaseHandler

	container := restful.NewContainer()
	container.Add(RegisterReleaseHandler(&mockReleaseHandler))

	refreshMockUseCase := func() {
		mockUseCase = &mocks.UseCase{}
		mockReleaseHandler.usecase = mockUseCase
	}

	tests := []struct {
		initMock   func()
		queryUrl   string
		statusCode int
	}{
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "testns", "").Return(nil, errors.New(""))
			},
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleases", "testns", "").Return([]*release.ReleaseInfoV2{}, nil)
			},
			statusCode: 200,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "testns", "test=true").Return(nil, errors.New(""))
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 500,
		},
		{
			initMock: func() {
				refreshMockUseCase()
				mockUseCase.On("ListReleasesByLabels", "testns", "test=true").Return([]*release.ReleaseInfoV2{}, nil)
			},
			queryUrl:   "?labelselector=test=true",
			statusCode: 200,
		},
	}

	for _, test := range tests {
		test.initMock()
		url := releaseRootPath + "/config/testns/" + test.queryUrl
		httpRequest, _ := http.NewRequest("GET", url, nil)
		httpWriter := httptest.NewRecorder()
		container.ServeHTTP(httpWriter, httpRequest)
		assert.Equal(t, test.statusCode, httpWriter.Code)
	}
}
