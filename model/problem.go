package model

import (
	ojmodel "github.com/to404hanga/online_judge_common/model"
)

type CreateProblemParam struct {
	CommonParam `json:"-"`

	Title          string `json:"title" binding:"required"`
	DescriptionURL string `json:"description_url" binding:"required"`               // 题面的 oss url(去除固定前缀)
	TestcaseZipURL string `json:"testcase_zip_url" binding:"required"`              // 测试用例的压缩包 oss url(去除固定前缀)
	TimeLimit      int    `json:"time_limit" binding:"required,min=50,max=30000"`   // 测试用例的时间限制(单位：豪秒)
	MemoryLimit    int    `json:"memory_limit" binding:"required,min=128,max=1024"` // 测试用例的内存限制(单位：MB)
	Visible        int8   `json:"visible" binding:"required"`                       // 非比赛期间是否可见
}

type UpdateProblemParam struct {
	CommonParam `json:"-"`

	ProblemID      uint64  `json:"problem_id" binding:"required"` // 题目 id
	Title          *string `json:"title"`
	DescriptionURL *string `json:"description_url"`                                   // 题面的 oss url(去除固定前缀)
	TestcaseZipURL *string `json:"testcase_zip_url"`                                  // 测试用例的压缩包 oss url(去除固定前缀)
	Status         *int8   `json:"status" binding:"omitempty,oneof=0 1 2"`            // 题目状态
	TimeLimit      *int    `json:"time_limit" binding:"omitempty,min=50,max=30000"`   // 测试用例的时间限制(单位：豪秒)
	MemoryLimit    *int    `json:"memory_limit" binding:"omitempty,min=128,max=1024"` // 测试用例的内存限制(单位：MB)
	Visible        *int8   `json:"visible" binding:"omitempty,oneof=0 1"`             // 非比赛期间是否可见
}

type GetProblemUploadPresignedURLParam struct {
	CommonParam `json:"-"`

	Hash string `json:"hash" binding:"required"`
}

type GetProblemUploadPresignedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type GetProblemDownloadPresignedURLParam struct {
	CommonParam `json:"-"`

	ProblemID uint64 `json:"problem_id" binding:"required"`
}

type GetProblemDownloadPresignedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type GetProblemTestcaseUploadPresignedURLParam struct {
	CommonParam `json:"-"`

	Hash string `json:"hash" binding:"required"`
}

type GetProblemTestcaseUploadPresignedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type GetProblemTestcaseDownloadPresignedURLParam struct {
	CommonParam `json:"-"`

	ProblemID uint64 `json:"problem_id" binding:"required"`
}

type GetProblemTestcaseDownloadPresignedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type GetProblemListParam struct {
	CommonParam `json:"-"`

	Desc        bool   `json:"desc"`
	Title       string `json:"title"`
	Status      *int8  `json:"status" binding:"omitempty,oneof=0 1 2"`
	Visible     *int8  `json:"visible" binding:"omitempty,oneof=0 1"`
	TimeLimit   *int   `json:"time_limit"`
	MemoryLimit *int   `json:"memory_limit"`

	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type GetProblemListResponse struct {
	List  []ojmodel.Problem `json:"list"`
	Total int               `json:"total"`
}
