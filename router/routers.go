package router

import (
	"github.com/gin-gonic/gin"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	trace "github.com/gin-contrib/tracing"
	stdopentracing "github.com/opentracing/opentracing-go"

	_ "walm/docs"
	. "walm/pkg/util/log"
	clus "walm/router/api/v1/cluster"
	inst "walm/router/api/v1/instance"
	"walm/router/ex"
	"walm/router/middleware"
)

// @title Walm
// @version 1.0.0
// @description Warp application lifecycle manager.

// @contact.name bing.han
// @contact.url http://transwarp.io
// @contact.email bing.han@transwarp.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host walm.transwarp.io
// @BasePath /walm/api/v1

func InitRouter(oauth bool, runmode string) *gin.Engine {
	r := gin.New()

	r.Use(gin.RecoveryWithWriter(Log.Out))

	//enable swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	gin.SetMode(runmode)
	if runmode == "debug" {
		r.Use(gin.LoggerWithWriter(Log.Out))
	} else {
		//add Prometheus Metric
		p := middleware.NewPrometheus("Walm-gin")
		p.Use(r)
		//add open tracing
		psr := func(spancontext stdopentracing.SpanContext) stdopentracing.StartSpanOption{
			return stdopentracing.ChildOf(spancontext)
			} 
		r.Use(trace.SpanFromHeaders(middleware.Tracer, "Walm", psr, false))
		
	}

	//add Probe for readiness and liveness
	r.GET("/readiness", readinessProbe)
	r.GET("/liveness", livenessProbe)

	//define api group
	apiv1 := r.Group("/walm/api/v1")
	if oauth {
		apiv1.Use(middleware.JWT())
	}
	{
		instance := apiv1.Group("/instance")
		{
			instance.DELETE("/{namespace}/{appName}", inst.DeleteApplication)
			instance.POST("/{chart}", inst.DeployApplication)
			instance.GET("/{namespace}/status/{appname}", inst.ListApplicationsWithStatus)
			instance.GET("/{namespace}/info{appname}", inst.GetApplicationStatusbyName)
			instance.GET("/{namespace}/rollback/{appname}/{version}", inst.RollBackApplication)
			instance.PUT("/{chart}", inst.UpdateApplication)
		}
		cluster := apiv1.Group("/cluster")
		{
			cluster.POST("/{namespace}/{name}", clus.DeployCluster)
			cluster.GET("/{namespace}/{name}", clus.StatusCluster)
			cluster.DELETE("/{namespace}/{name}", clus.DeleteCluster)
		}

	}

	return r
}

func readinessProbe(c *gin.Context) {
	c.JSON(ex.ReturnOK())
}

func livenessProbe(c *gin.Context) {
	c.JSON(ex.ReturnOK())
}