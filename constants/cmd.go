package constants

const (
	CreateProblemPath = "/CreateProblem" // 创建题目
	UpdateProblemPath = "/UpdateProblem" // 更新题目
	// GetProblemUploadPresignedURLPath           = "/GetProblemUploadPresignedURL"           // 获取题目上传预签名 URL
	// GetProblemDownloadPresignedURLPath         = "/GetProblemDownloadPresignedURL"         // 获取题目下载预签名 URL
	// GetProblemTestcaseUploadPresignedURLPath   = "/GetProblemTestcaseUploadPresignedURL"   // 获取题目测试用例上传预签名 URL
	// GetProblemTestcaseDownloadPresignedURLPath = "/GetProblemTestcaseDownloadPresignedURL" // 获取题目测试用例下载预签名 URL
	GetProblemListPath        = "/GetProblemList"        // 获取题目列表
	UploadProblemTestcasePath = "/UploadProblemTestcase" // 上传题目测试用例
	GetProblemPath            = "/GetProblem"            // 获取题目
)

const (
	CreateCompetitionPath               = "/CreateCompetition"               // 创建比赛
	UpdateCompetitionPath               = "/UpdateCompetition"               // 更新比赛
	AddCompetitionProblemPath           = "/AddCompetitionProblem"           // 添加比赛题目
	RemoveCompetitionProblemPath        = "/RemoveCompetitionProblem"        // 删除比赛题目
	EnableCompetitionProblemPath        = "/EnableCompetitionProblem"        // 启用比赛题目
	DisableCompetitionProblemPath       = "/DisableCompetitionProblem"       // 禁用比赛题目
	StartCompetitionPath                = "/StartCompetition"                // 开始比赛
	GetCompetitionRankingListPath       = "/GetCompetitionRankingList"       // 获取比赛排名列表
	GetCompetitionFastestSolverListPath = "/GetCompetitionFastestSolverList" // 获取比赛各个题目最快通过提交的用户列表
	ExportCompetitionDataPath           = "/ExportCompetitionData"           // 导出比赛数据
	InitRankingPath                     = "/InitRanking"                     // 初始化比赛排名
	UpdateScorePath                     = "/UpdateScore"                     // 更新比赛用户分数, 仅内部测试用, 后续 release 版本移除
	GetCompetitionListPath              = "/GetCompetitionList"              // 获取比赛列表
	GetCompetitionPath                  = "/GetCompetition"                  // 获取比赛
	UserGetCompetitionListPath          = "/UserGetCompetitionList"          // 用户获取比赛列表
	GetCompetitionProblemListPath       = "/GetCompetitionProblemList"       // 获取比赛题目列表
	UserGetCompetitionProblemListPath   = "/UserGetCompetitionProblemList"   // 用户获取比赛题目列表
	UserGetCompetitionProblemDetailPath = "/UserGetCompetitionProblemDetail" // 用户获取比赛题目详情
)

const (
	SubmitCompetitionProblemPath = "/SubmitCompetitionProblem" // 提交比赛题目
	GetLatestSubmissionPath      = "/GetLatestSubmission"      // 获取最新提交
)

const (
	GetUserListPath               = "/GetUserList"               // 获取用户列表
	DeleteUserPath                = "/DeleteUser"                // 删除用户
	UpdateUserPath                = "/UpdateUser"                // 更新用户
	ResetPasswordPath             = "/ResetPassword"             // 重置用户密码
	UpdatePasswordPath            = "/UpdatePassword"            // 更新用户密码
	GetCompetitionUserListPath    = "/GetCompetitionUserList"    // 获取比赛用户列表
	AddUsersToCompetitionPath     = "/AddUsersToCompetition"     // 添加用户到比赛名单
	EnableUsersInCompetitionPath  = "/EnableUsersInCompetition"  // 允许用户参加比赛
	DisableUsersInCompetitionPath = "/DisableUsersInCompetition" // 禁用用户参加比赛
	CreateUserPath                = "/CreateUser"                // 创建用户
)
