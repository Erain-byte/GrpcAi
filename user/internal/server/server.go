package server

import (
	"user/internal/handler"
	"user/internal/middleware"
	"user/internal/svc"
	"user/internal/types"
	"github.com/gin-gonic/gin"
)

func NewServer(svcCtx *svc.ServiceContext) *gin.Engine {
	r := gin.Default()

	// 中间件
	r.Use(middleware.Recovery())

	// 初始化处理器
	h := handler.NewHandler(svcCtx)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, types.Response{
			Code:    200,
			Message: "ok",
		})
	})

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 用户相关路由（公开接口）
		user := v1.Group("/user")
		{
			// 注册
			user.POST("/register", h.Register)
			// 登录
			user.POST("/login", h.Login)
		}

		// 用户相关路由（需要认证的接口）
		userAuth := v1.Group("/user").Use(middleware.JWTAuth())
		{
			// 获取用户列表
			userAuth.GET("/list", h.GetUserList)
		}
	}

	return r
}
