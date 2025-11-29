package repositoryimpl

import (
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/opensourceways/hot-topic-website-backend/common/domain/repository"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
)

func NewTopicToReview(dao Dao) *topicToReview {
	return &topicToReview{
		dao: dao,
	}
}

type topicToReview struct {
	dao Dao
}

func (impl *topicToReview) NewId() string {
	// avoid generate id at same time
	time.Sleep(time.Microsecond)

	return impl.dao.NewDocId()
}

func (impl *topicToReview) filter(community string) bson.M {
	return bson.M{fieldCommunity: community}
}

func (impl *topicToReview) Add(community string, v *domain.TopicsToReview) error {
	do := totopicsToReviewDO(community, v)
	doc, err := do.toDoc()
	if err != nil {
		return err
	}
	doc[fieldVersion] = 0

	_, err = impl.dao.ReplaceDoc(impl.filter(community), doc)

	return err
}

func (impl *topicToReview) Find(community string) (domain.TopicsToReview, error) {
	var do topicsToReviewDO

	if err := impl.dao.GetDoc(impl.filter(community), nil, nil, &do); err != nil {
		if impl.dao.IsDocNotExists(err) {
			err = repository.NewErrorResourceNotFound(err)
		}

		return domain.TopicsToReview{}, err
	}

	return do.toTopicsToReview(), nil
}

func (impl *topicToReview) FindSelected(community string) (domain.TopicsToReview, error) {
	var do topicsToReviewDO

	if err := impl.dao.GetDoc(impl.filter(community), bson.M{fieldCandidates: 0}, nil, &do); err != nil {
		if impl.dao.IsDocNotExists(err) {
			err = repository.NewErrorResourceNotFound(err)
		}

		return domain.TopicsToReview{}, err
	}

	return do.toTopicsToReview(), nil
}

func (impl *topicToReview) SaveSelected(community string, v *domain.TopicsToReview) error {
	do := toSelectedTopicsDO(v.Selected)
	doc, err := do.toDoc()
	logrus.Infof("dao:%v, err:%v", impl.dao, err)
	if err != nil {
		return err
	}

	return impl.dao.UpdateDoc(impl.filter(community), doc, v.Version)
}
