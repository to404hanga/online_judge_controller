package event

import "encoding/json"

const SubmissionTopic = "submission_topic"

type SubmissionMessage struct {
	SubmissionID int64 `json:"submission_id"`
}

func (s *SubmissionMessage) Marshal() ([]byte, error) {
	return json.Marshal(s)
}