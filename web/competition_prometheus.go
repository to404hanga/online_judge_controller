package web

import "github.com/prometheus/client_golang/prometheus"

var (
	startCompetitionRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "start_competition_requests_total",
			Help:      "StartCompetition requests total.",
		},
		[]string{"code", "reason"},
	)
	startCompetitionDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "start_competition_duration_seconds",
			Help:      "StartCompetition duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
	getCompetitionRankingListRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "get_competition_ranking_list_requests_total",
			Help:      "GetCompetitionRankingList requests total.",
		},
		[]string{"code", "reason"},
	)
	getCompetitionRankingListDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "get_competition_ranking_list_duration_seconds",
			Help:      "GetCompetitionRankingList duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
	userGetCompetitionListRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "user_get_competition_list_requests_total",
			Help:      "UserGetCompetitionList requests total.",
		},
		[]string{"code", "reason"},
	)
	userGetCompetitionListDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "user_get_competition_list_duration_seconds",
			Help:      "UserGetCompetitionList duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
	userGetCompetitionProblemListRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "user_get_competition_problem_list_requests_total",
			Help:      "UserGetCompetitionProblemList requests total.",
		},
		[]string{"code", "reason"},
	)
	userGetCompetitionProblemListDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "user_get_competition_problem_list_duration_seconds",
			Help:      "UserGetCompetitionProblemList duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
	userGetCompetitionProblemDetailRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "user_get_competition_problem_detail_requests_total",
			Help:      "UserGetCompetitionProblemDetail requests total.",
		},
		[]string{"code", "reason"},
	)
	userGetCompetitionProblemDetailDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "user_get_competition_problem_detail_duration_seconds",
			Help:      "UserGetCompetitionProblemDetail duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
	checkUserCompetitionProblemAcceptedRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "check_user_competition_problem_accepted_requests_total",
			Help:      "CheckUserCompetitionProblemAccepted requests total.",
		},
		[]string{"code", "reason"},
	)
	checkUserCompetitionProblemAcceptedDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "check_user_competition_problem_accepted_duration_seconds",
			Help:      "CheckUserCompetitionProblemAccepted duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
	timeEventConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "time_event_connections_total",
			Help:      "TimeEventHandler connections total.",
		},
		[]string{"reason"},
	)
	timeEventConnectionDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "time_event_connection_duration_seconds",
			Help:      "TimeEventHandler connection duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"reason"},
	)
	timeEventActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "online_judge_controller",
			Subsystem: "competition",
			Name:      "time_event_active_connections",
			Help:      "TimeEventHandler active connections.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		startCompetitionRequestsTotal,
		startCompetitionDurationSeconds,
		getCompetitionRankingListRequestsTotal,
		getCompetitionRankingListDurationSeconds,
		userGetCompetitionListRequestsTotal,
		userGetCompetitionListDurationSeconds,
		userGetCompetitionProblemListRequestsTotal,
		userGetCompetitionProblemListDurationSeconds,
		userGetCompetitionProblemDetailRequestsTotal,
		userGetCompetitionProblemDetailDurationSeconds,
		checkUserCompetitionProblemAcceptedRequestsTotal,
		checkUserCompetitionProblemAcceptedDurationSeconds,
		timeEventConnectionsTotal,
		timeEventConnectionDurationSeconds,
		timeEventActiveConnections,
	)
}
