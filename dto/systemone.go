package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type SystemOneQuestion struct {
	ID           string            `json:"id,omitempty"`
	Question     string            `json:"question,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Type         string            `json:"type"` // "noul", "choice", "score"
	Options      []string          `json:"options,omitempty"`
	Legend       map[string]string `json:"legend,omitempty"`
	Resolution   string            `json:"resolution,omitempty"`
}

type SystemOneRequest struct {
	Model        string              `json:"model"`
	State        any                 `json:"state"`
	Questions    []SystemOneQuestion `json:"-"`
	RawQuestions json.RawMessage     `json:"questions"`
}

func (r *SystemOneRequest) UnmarshalJSON(data []byte) error {
	type rawReq struct {
		Model     string          `json:"model"`
		State     any             `json:"state"`
		Questions json.RawMessage `json:"questions"`
	}
	var raw rawReq
	if err := common.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Model = raw.Model
	r.State = raw.State
	r.RawQuestions = raw.Questions

	trimmed := bytes.TrimSpace(raw.Questions)
	if len(trimmed) == 0 {
		return nil
	}

	if trimmed[0] == '[' {
		var qList []SystemOneQuestion
		if err := common.Unmarshal(trimmed, &qList); err != nil {
			return err
		}
		r.Questions = qList
	} else if trimmed[0] == '{' {
		var qMap map[string]SystemOneQuestion
		if err := common.Unmarshal(trimmed, &qMap); err != nil {
			return err
		}
		for id, q := range qMap {
			if q.ID == "" {
				q.ID = id
			}
			r.Questions = append(r.Questions, q)
		}
	}
	return nil
}

func (r *SystemOneRequest) MarshalJSON() ([]byte, error) {
	type outReq struct {
		Model     string `json:"model"`
		State     any    `json:"state"`
		Questions any    `json:"questions"`
	}
	out := outReq{
		Model: r.Model,
		State: r.State,
	}
	if len(r.RawQuestions) > 0 {
		out.Questions = r.RawQuestions
	} else {
		out.Questions = r.Questions
	}
	return common.Marshal(out)
}

func (r *SystemOneRequest) IsStream(c *gin.Context) bool {
	return false
}

func (r *SystemOneRequest) GetTokenCountMeta() *types.TokenCountMeta {
	var texts []string
	if r.State != nil {
		switch s := r.State.(type) {
		case string:
			if s != "" {
				texts = append(texts, s)
			}
		default:
			if b, err := common.Marshal(s); err == nil {
				texts = append(texts, string(b))
			}
		}
	}
	for _, q := range r.Questions {
		if q.ID != "" {
			texts = append(texts, q.ID)
		}
		if q.Question != "" {
			texts = append(texts, q.Question)
		}
		if q.Instructions != "" {
			texts = append(texts, q.Instructions)
		}
		if q.Type != "" {
			texts = append(texts, q.Type)
		}
		for _, opt := range q.Options {
			texts = append(texts, opt)
		}
		for k, v := range q.Legend {
			texts = append(texts, fmt.Sprintf("%s: %s", k, v))
		}
		if q.Resolution != "" {
			texts = append(texts, q.Resolution)
		}
	}
	return &types.TokenCountMeta{
		CombineText: strings.Join(texts, "\n"),
	}
}

func (r *SystemOneRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

type SystemOneResponse struct {
	Model   string `json:"model"`
	Usage   Usage  `json:"usage"`
	Answers any    `json:"answers,omitempty"`
}
