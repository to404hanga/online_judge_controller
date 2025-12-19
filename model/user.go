package model

import ojmodel "github.com/to404hanga/online_judge_common/model"

type GetUserListParam struct {
	CommonParam `json:"-"`

	OrderBy string `form:"order_by" binding:"omitempty,oneof=id username realname"` // 排序字段
	Desc    bool   `form:"desc"`                                                    // 是否降序

	Username string              `form:"username"` // 按用户名查询, 前缀匹配
	Realname string              `form:"realname"` // 按真实姓名查询, 全模糊匹配
	Role     *ojmodel.UserRole   `form:"role"`     // 按角色查询, 0: 普通用户, 1: 管理员
	Status   *ojmodel.UserStatus `form:"status"`   // 按状态查询, 0: 正常, 1: 禁用

	Page     int `form:"page" binding:"required,min=1"`               // 分页页码
	PageSize int `form:"page_size" binding:"required,min=10,max=100"` // 分页每页数量
}

type GetUserListResponse struct {
	Total    int            `json:"total"`     // 总记录数
	List     []ojmodel.User `json:"list"`      // 记录列表
	Page     int            `json:"page"`      // 分页页码
	PageSize int            `json:"page_size"` // 分页每页数量
}

type AddUsersToCompetition struct {
	CommonParam `json:"-"`

	CompetitionID uint64   `form:"competition_id" binding:"required"` // 竞赛ID
	UserIDList    []uint64 `json:"user_id_list"`                      // 用户ID, 仅当管理页面选择用户时使用
}

type AddUsersToCompetitionResponse struct {
	InsertSuccess int64 `json:"insert_success"` // 成功插入行数
}

type CompetitionUserListParam struct {
	CommonParam `json:"-"`

	CompetitionID uint64   `json:"competition_id" binding:"required"` // 竞赛ID
	UserIDList    []uint64 `json:"user_id_list" binding:"required"`   // 用户ID
}

type DeleteUserParam struct {
	CommonParam `json:"-"`

	UserID uint64 `json:"user_id" binding:"required"` // 用户ID
}

type UpdateUserParam struct {
	CommonParam `json:"-"`

	UserID uint64 `json:"user_id" binding:"required"` // 用户ID

	Realname string              `json:"realname"`                             // 真实姓名
	Status   *ojmodel.UserStatus `json:"status" binding:"omitempty,oneof=0 1"` // 状态, 0: 正常, 1: 禁用
	Password string              `json:"password"`                             // 密码
}
