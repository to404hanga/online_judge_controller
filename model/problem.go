package model

import (
	ojmodel "github.com/to404hanga/online_judge_common/model"
)

type CreateProblemParam struct {
	CommonParam `json:"-"`

	Title           string `json:"title" binding:"required"`
	Description     string `json:"description" binding:"required"`                   // 题面的描述
	DescriptionHash string `header:"X-Description-Hash" binding:"required"`          // 题面的描述哈希值, 字母为小写
	TimeLimit       int    `json:"time_limit" binding:"required,min=50,max=30000"`   // 测试用例的时间限制(单位：豪秒)
	MemoryLimit     int    `json:"memory_limit" binding:"required,min=128,max=1024"` // 测试用例的内存限制(单位：MB)
	Visible         *int8  `json:"visible" binding:"required,oneof=0 1"`             // 非比赛期间是否可见
}

type UpdateProblemParam struct {
	CommonParam `json:"-"`

	ProblemID       uint64  `json:"problem_id" binding:"required"` // 题目 id
	Title           *string `json:"title"`
	Description     *string `json:"description"`                                       // 题面的描述
	DescriptionHash *string `header:"X-Description-Hash"`                              // 题面的描述哈希值, 字母为小写
	Status          *int8   `json:"status" binding:"omitempty,oneof=0 1 2"`            // 题目状态
	TimeLimit       *int    `json:"time_limit" binding:"omitempty,min=50,max=30000"`   // 测试用例的时间限制(单位：豪秒)
	MemoryLimit     *int    `json:"memory_limit" binding:"omitempty,min=128,max=1024"` // 测试用例的内存限制(单位：MB)
	Visible         *int8   `json:"visible" binding:"omitempty,oneof=0 1"`             // 非比赛期间是否可见
}

type GetProblemListParam struct {
	CommonParam `json:"-"`

	Desc        bool   `form:"desc"`
	OrderBy     string `form:"order_by" binding:"omitempty,oneof=id created_at updated_at"`
	Title       string `form:"title"`
	Status      *int8  `form:"status" binding:"omitempty,oneof=0 1 2"`
	Visible     *int8  `form:"visible" binding:"omitempty,oneof=0 1"`
	TimeLimit   *int   `form:"time_limit"`
	MemoryLimit *int   `form:"memory_limit"`

	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=10,max=100"`
}

type GetProblemListResponse struct {
	List  []ojmodel.Problem `json:"list"`
	Total int               `json:"total"`
}

type UploadProblemTestcaseParam struct {
	CommonParam `json:"-"`

	ProblemID uint64 `json:"problem_id" binding:"required"`
}

type GetProblemParam struct {
	CompetitionCommonParam `json:"-"`

	ProblemID uint64 `json:"problem_id" binding:"required"`
}

type GetProblemResponse struct {
	*ojmodel.Problem `json:",inline"`
}
