package route

import (
	"FileStore/handler"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	router := gin.Default()
	router.Static("/static/", "./static")
	router.GET("/user/signup", handler.SignupGetHandler)
	router.POST("/user/signup", handler.SignupPostHandler)
	router.GET("/user/signin", handler.SigninGetHandler)
	router.POST("/user/signin", handler.SigninPostHandler)
	//加入拦截器校验token
	router.Use(handler.HTTPInterceptor())
	router.POST("/file/downloadurl", handler.DownloadURLHandler)
	router.POST("/user/info", handler.UserInfoHandler)
	router.GET("/file/upload", handler.UploadGetHandler)
	router.POST("/file/upload", handler.UploadPostHandler)

	router.POST("/file/fastupload", handler.TryFastUploadHandler)
	router.POST("/file/meta", handler.GetFileMetaHandler)
	router.POST("/file/query", handler.FileQueryHandler)
	//http.HandleFunc("/file/download", handler.DownloadHandler)
	router.POST("/file/update", handler.FileMetaUpdateHandler)
	//http.HandleFunc("/file/delete", handler.FileDeleteHandler)

	return router

}
