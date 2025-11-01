package constants

const (
	CreateProblemPath                          = "/CreateProblem"                          // 创建题目
	UpdateProblemPath                          = "/UpdateProblem"                          // 更新题目
	GetProblemUploadPresignedURLPath           = "/GetProblemUploadPresignedURL"           // 获取题目上传预签名 URL
	GetProblemDownloadPresignedURLPath         = "/GetProblemDownloadPresignedURL"         // 获取题目下载预签名 URL
	GetProblemTestcaseUploadPresignedURLPath   = "/GetProblemTestcaseUploadPresignedURL"   // 获取题目测试用例上传预签名 URL
	GetProblemTestcaseDownloadPresignedURLPath = "/GetProblemTestcaseDownloadPresignedURL" // 获取题目测试用例下载预签名 URL
	GetProblemListPath                         = "/GetProblemList"                         // 获取题目列表
)

const (
	CreateCompetitionPath                         = "/CreateCompetition"                         // 创建比赛
	UpdateCompetitionPath                         = "/UpdateCompetition"                         // 更新比赛
	AddCompetitionProblemPath                     = "/AddCompetitionProblem"                     // 添加比赛题目
	RemoveCompetitionProblemPath                  = "/RemoveCompetitionProblem"                  // 删除比赛题目
	EnableCompetitionProblemPath                  = "/EnableCompetitionProblem"                  // 启用比赛题目
	DisableCompetitionProblemPath                 = "/DisableCompetitionProblem"                 // 禁用比赛题目
	StartCompetitionPath                          = "/StartCompetition"                          // 开始比赛
	GetCompetitionProblemListWithPresignedURLPath = "/GetCompetitionProblemListWithPresignedURL" // 获取比赛题目列表（带预签名 URL）
	GetCompetitionRankingListPath                 = "/GetCompetitionRankingList"                 // 获取比赛排名列表
	GetCompetitionFastestSolverListPath           = "/GetCompetitionFastestSolverList"           // 获取比赛各个题目最快通过提交的用户列表
	ExportCompetitionDataPath                     = "/ExportCompetitionData"                     // 导出比赛数据
)

const (
	GetSubmissionUploadPresignedURLPath = "/GetSubmissionUploadPresignedURL" // 获取提交上传预签名 URL
	SubmitCompetitionProblemPath        = "/SubmitCompetitionProblem"        // 提交比赛题目
	// GetSubmissionListPath = "/GetSubmissionList" // 获取提交列表
	GetSubmissionDownloadPresignedURLPath = "/GetSubmissionDownloadPresignedURL" // 获取提交下载预签名 URL
	GetLatestSubmissionPath               = "/GetLatestSubmission"               // 获取最新提交
)
