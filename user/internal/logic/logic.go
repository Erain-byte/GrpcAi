package logic

import (
	"errors"
	"user/pkg"
	"user/internal/svc"
	"user/internal/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Logic struct {
 svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx: svcCtx,
	}
}

// Register 用户注册
func (l *Logic) Register(req *types.RegisterRequest) error {
	// 检查用户名是否已存在
	var count int64
	if err := l.svcCtx.DB.Model(&types.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("username already exists")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 创建用户
	user := types.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   1,
	}

	if err := l.svcCtx.DB.Create(&user).Error; err != nil {
		return err
	}

	return nil
}

// Login 用户登录
func (l *Logic) Login(req *types.LoginRequest) (*types.LoginResponse, error) {
	var user types.User
	if err := l.svcCtx.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid password")
	}

	// 生成token
	token, err := pkg.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &types.LoginResponse{
		Token: token,
	}, nil
}

// GetUserList 获取用户列表
func (l *Logic) GetUserList(req *types.PageRequest) (*types.PageResponse, error) {
	var total int64
	var users []types.User

	// 查询总数
	if err := l.svcCtx.DB.Model(&types.User{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 查询用户列表
	offset := (req.Page - 1) * req.PageSize
	if err := l.svcCtx.DB.Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
		return nil, err
	}

	return &types.PageResponse{
		List:     users,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
