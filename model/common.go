package model

type CommonParam struct {
	Operator uint64
}

type CommonParamInterface interface {
	SetOperator(op uint64)
}

func (p *CommonParam) SetOperator(op uint64) {
	p.Operator = op
}

type CompetitionCommonParam struct {
	CommonParam
	CompetitionID uint64
}

type CompetitionCommonParamInterface interface {
	CommonParamInterface
	SetCompetitionID(id uint64)
}

func (p *CompetitionCommonParam) SetCompetitionID(id uint64) {
	p.CompetitionID = id
}

type PresignedURL struct {
	ID  uint64 `json:"id"`
	URL string `json:"url"`
}
