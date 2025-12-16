package repository

import "github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"

type RepoHotTopic interface {
	Add(string, *domain.HotTopic) error
	Find(string, string) (domain.HotTopic, error)
	Save(string, *domain.HotTopic) error
	FindAll(string, int64) ([]domain.HotTopic, error)
	FindOpenOnes(string) ([]domain.HotTopic, error)
	FindClosedOnes(string) ([]domain.HotTopic, error)
}

type RepoNotHotTopic interface {
	Save(community string, date int64, items []domain.NotHotTopic) error
	FindAll(string) ([]domain.NotHotTopic, error)
	FindCreatedAt(community string) (int64, error)
}

type RepoTopicsToReview interface {
	NewId() string
	Add(string, *domain.TopicsToReview) error
	Find(string) (domain.TopicsToReview, error)
	SaveSelected(string, *domain.TopicsToReview) error
	FindSelected(string) (domain.TopicsToReview, error)
}
