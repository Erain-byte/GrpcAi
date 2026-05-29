package handler

import (
	"net/http"
	"user/internal/logic"
	"user/internal/svc"
	"user/internal/types"
	"github.com/gin-gonic/gin"
)

type Handler struct {
 logic *logic.Logic
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		logic: logic.NewLogic(svcCtx),
	}
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req types.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if err := h.logic.Register(&req); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Code:    http.StatusOK,
		Message: "register success",
	})
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req types.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	resp, err := h.logic.Login(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Code:    http.StatusOK,
		Message: "login success",
		Data:    resp,
	})
}

// GetUserList 获取用户列表
func (h *Handler) GetUserList(c *gin.Context) {
	var req types.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	resp, err := h.logic.GetUserList(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    resp,
	})
}
