package app

import (
	"fmt"
	"time"

	"github.com/opensourceways/hot-topic-website-backend/common/domain/allerror"
	"github.com/opensourceways/hot-topic-website-backend/common/domain/repository"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
	"github.com/opensourceways/hot-topic-website-backend/utils"
)

func (s *appService) toSelected(
	oldTopics []domain.HotTopic, current map[string]*OptionalTopic, category string,
) (
	[]domain.TopicToReview, error,
) {
	r := make([]domain.TopicToReview, len(oldTopics))

	for i := range oldTopics {
		old := &oldTopics[i]

		item := current[old.Title]
		item.updateAppended(old.GetDSSet())

		r[i] = item.toTopicToReview(category)

		if err := old.InitReview(&r[i]); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (s *appService) checkInvokeForReview() error {
	times := []time.Weekday{time.Friday, time.Saturday, time.Sunday}

	return s.checkInvokeByTime(times)
}

func (s *appService) GetTopicsToReview(community string) (TopicsToReviewDTO, error) {
	if err := s.checkInvokeForReview(); err != nil {
		return TopicsToReviewDTO{}, err
	}

	v, err := s.repoTopicsToReview.Find(community)
	if err != nil && repository.IsErrorResourceNotFound(err) {
		err = allerror.NewNotFoundError("no review data", err)
	}

	return v, err
}

func (s *appService) UpdateSelected(community string, cmd *CmdToUpdateSelected) error {
	if err := s.checkInvokeForReview(); err != nil {
		return err
	}

	t, err := s.repoTopicsToReview.FindSelected(community)
	if err != nil {
		if repository.IsErrorResourceNotFound(err) {
			err = allerror.NewNotFoundError("can't find old reviewed data", err)
		}

		return err
	}

	if err := t.UpdateSelected(sheetLastTopics, cmd.Selected, s.repoTopicsToReview.NewId); err != nil {
		return err
	}

	// TODO check if all the discussion sources are valid, for example there are from Data Clean

	return s.repoTopicsToReview.SaveSelected(community, &t)
}

func (s *appService) GetHotTopics(community string, date int64) (HotTopicsDTO, error) {
	hts, err := s.repoHotTopic.FindAll(community, date)
	if err != nil {
		return HotTopicsDTO{}, err
	}

	return toHotTopicsDTO(hts, date), nil
}

func (s *appService) getReviews(community string, dateSec int64) (review domain.TopicsToReview, err error) {
	review, err = s.repoTopicsToReview.Find(community)
	if err != nil {
		return
	}

	if !review.IsMatchedReview(dateSec) {
		err = allerror.New(
			allerror.ErrorCodeNoMatchedTopicsToReview,
			fmt.Sprintf(
				"it is not the right review which does not match the time. expect:%d, has:%d",
				dateSec, review.Date,
			),
			nil,
		)
	}

	return
}

func (s *appService) GetTopicsToPublish(community string) (dto HotTopicsDTO, err error) {
	if err := s.checkInvokeForReview(); err != nil {
		return HotTopicsDTO{}, err
	}

	date := utils.GetLastFriday()
	dateSec := date.Unix()

	review, err := s.getReviews(community, dateSec)
	if err != nil {
		return
	}

	hts, err := s.repoHotTopic.FindAll(community, dateSec)
	if err != nil {
		return
	}

	_, news := review.FilterChangedAndNews(hts, date)

	dto = toHotTopicsDTO(append(hts, news...), dateSec)

	return
}

func (s *appService) ApplyToHotTopic(community string) error {
	weeks := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday}
	if err := s.checkInvokeByTime(weeks); err != nil {
		return err
	}

	date := utils.GetLastFriday()
	dateSec := date.Unix()

	review, err := s.getReviews(community, dateSec)
	if err != nil {
		return err
	}

	hts, err := s.repoHotTopic.FindAll(community, dateSec)
	if err != nil {
		return err
	}

	changed, news := review.FilterChangedAndNews(hts, date)

	for i := range changed {
		if err := s.repoHotTopic.Save(community, changed[i]); err != nil {
			return err
		}
	}

	for i := range news {
		if err := s.repoHotTopic.Add(community, &news[i]); err != nil {
			return err
		}
	}

	return s.repoNotHotTopic.Save(community, dateSec, review.GenNotHotTopics())
}

func (s *appService) GetWorthlessNotHotTopic(community string) (NotHotTopicsDTO, error) {
	topics, err := s.repoNotHotTopic.FindAll(community)
	if err != nil {
		return NotHotTopicsDTO{}, nil
	}

	items := make([]*domain.NotHotTopic, 0, len(topics))
	for i := range topics {
		if item := &topics[i]; item.IsWorthless(sheetNewTopics) {
			items = append(items, item)
		}
	}

	return toNotHotTopicsDTO(items), nil
}
