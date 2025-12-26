package web

import "github.com/prometheus/client_golang/prometheus"

var (
	submitCompetitionProblemRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "submission",
			Name:      "submit_competition_problem_requests_total",
			Help:      "SubmitCompetitionProblem requests total.",
		},
		[]string{"code", "reason", "language"},
	)
	submitCompetitionProblemDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "submission",
			Name:      "submit_competition_problem_duration_seconds",
			Help:      "SubmitCompetitionProblem duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason", "language"},
	)
	getLatestSubmissionRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "submission",
			Name:      "get_latest_submission_requests_total",
			Help:      "GetLatestSubmission requests total.",
		},
		[]string{"code", "reason"},
	)
	getLatestSubmissionDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "submission",
			Name:      "get_latest_submission_duration_seconds",
			Help:      "GetLatestSubmission duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason"},
	)
)

func init() {
	prometheus.MustRegister(
		submitCompetitionProblemRequestsTotal,
		submitCompetitionProblemDurationSeconds,
		getLatestSubmissionRequestsTotal,
		getLatestSubmissionDurationSeconds,
	)
}
