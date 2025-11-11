package watch

import (
	"time"

	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/watch/forum"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/watch/gitcodeissue"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/watch/giteeissue"
)

type Config struct {
	// unit second
	Interval int             `json:"interval"`
	Forums   []ForumConfig   `json:"forums"`
	GitCodes []GitCodeConfig `json:"gitcodes"`
	Gitees   []GiteeConfig   `json:"gitees"`
}

func (cfg *Config) ConfigItems() []interface{} {
	r := make([]interface{}, len(cfg.Forums)+len(cfg.GitCodes))

	for i := range cfg.Forums {
		r[i] = &cfg.Forums[i]
	}

	v := r[len(cfg.Forums):]
	for i := range cfg.GitCodes {
		v[i] = &cfg.GitCodes[i]
	}

	return r
}

func (cfg *Config) SetDefault() {
	if cfg.Interval <= 0 {
		cfg.Interval = 600
	}
}

func (cfg *Config) intervalDuration() time.Duration {
	return time.Second * time.Duration(cfg.Interval)
}

// ForumConfig
type ForumConfig struct {
	Community string       `json:"community"    required:"true"`
	Detail    forum.Config `json:"detail"`
}

func (cfg *ForumConfig) typeDesc() string {
	return "forum"
}

func (cfg *ForumConfig) ConfigItems() []interface{} {
	return []interface{}{&cfg.Detail}
}

// GitCodeConfig
type GitCodeConfig struct {
	Community string              `json:"community"    required:"true"`
	Detail    gitcodeissue.Config `json:"detail"`
}

func (cfg *GitCodeConfig) typeDesc() string {
	return "issue"
}

func (cfg *GitCodeConfig) ConfigItems() []interface{} {
	return []interface{}{&cfg.Detail}
}

type GiteeConfig struct {
	Community string            `json:"community"    required:"true"`
	Detail    giteeissue.Config `json:"detail"`
}

func (cfg *GiteeConfig) typeDesc() string {
	return "issue"
}

func (cfg *GiteeConfig) ConfigItems() []interface{} {
	return []interface{}{&cfg.Detail}
}
