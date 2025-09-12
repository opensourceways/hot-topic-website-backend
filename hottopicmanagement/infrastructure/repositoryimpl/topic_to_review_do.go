package repositoryimpl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
)

const (
	fieldCommunity  = "community"
	fieldCandidates = "candidates"
)

func totopicsToReviewDO(Community string, v *domain.TopicsToReview) topicsToReviewDO {
	candidates := make([]topicToReviewDO, 0, v.CandidatesNum())
	for _, items := range v.Candidates {
		for i := range items {
			candidates = append(candidates, totopicToReviewDO(&items[i]))
		}
	}

	return topicsToReviewDO{
		Candidates:       candidates,
		Date:             v.Date,
		Community:        Community,
		SelectedTopicsDO: toSelectedTopicsDO(v.Selected),
	}
}

func toSelectedTopicsDO(items []domain.TopicToReview) SelectedTopicsDO {
	r := make([]topicToReviewDO, len(items))
	for i := range items {
		r[i] = totopicToReviewDO(&items[i])
	}

	return SelectedTopicsDO{r}
}

// SelectedTopicsDO
type SelectedTopicsDO struct {
	Selected []topicToReviewDO `bson:"selected"    json:"selected"`
}

func (do *SelectedTopicsDO) toDoc() (bson.M, error) {
	return genDoc(do)
}

func (do *SelectedTopicsDO) toSelected() []domain.TopicToReview {
	r := make([]domain.TopicToReview, len(do.Selected))

	for i := range do.Selected {
		r[i] = do.Selected[i].toTopicToReview()
	}

	return r
}

// topicsToReviewDO
type topicsToReviewDO struct {
	Candidates []topicToReviewDO `bson:"candidates"  json:"candidates"`
	Community  string            `bson:"community"   json:"community"`
	Version    int               `bson:"version"     json:"-"`
	Date       int64             `bson:"date"        json:"date"`

	SelectedTopicsDO `bson:",inline"`
}

func (do *topicsToReviewDO) toDoc() (bson.M, error) {
	return genDoc(do)
}

func (do *topicsToReviewDO) toTopicsToReview() domain.TopicsToReview {
	t := domain.NewTopicsToReview(do.Date)
	t.Version = do.Version
	t.Selected = do.SelectedTopicsDO.toSelected()

	for i := range do.Candidates {
		item := do.Candidates[i]
		v := item.toTopicToReview()

		t.AddCandidate(item.Category, &v)
	}

	return t
}

// topicToReviewDO
type topicToReviewDO struct {
	Order             int                          `bson:"order"    json:"order"`
	Title             string                       `bson:"title"    json:"title"`
	Category          string                       `bson:"category" json:"category"`
	Resolved          bool                         `bson:"resolved" json:"resolved"`
	HotTopicId        string                       `bson:"ht_id"    json:"ht_id"`
	DiscussionSources []discussionSourceToReviewDO `bson:"sources"  json:"sources"`
	DSSCount          int                          `bson:"dss_count" json:"dss_count"`
	CommentCount      int                          `bson:"comment_count" json:"comment_count"`
}

func (do *topicToReviewDO) toTopicToReview() domain.TopicToReview {
	r := make([]domain.DiscussionSourceToReview, len(do.DiscussionSources))
	for i := range do.DiscussionSources {
		r[i] = do.DiscussionSources[i].toDiscussionSourceToReview()
	}

	return domain.TopicToReview{
		Order:             do.Order,
		Title:             do.Title,
		Category:          do.Category,
		Resolved:          do.Resolved,
		HotTopicId:        do.HotTopicId,
		DiscussionSources: r,
		DSSCount:          do.DSSCount,
		CommentCount:      do.CommentCount,
	}
}

func totopicToReviewDO(v *domain.TopicToReview) topicToReviewDO {
	r := make([]discussionSourceToReviewDO, len(v.DiscussionSources))
	for i := range v.DiscussionSources {
		r[i] = todiscussionSourceToReviewDO(&v.DiscussionSources[i])
	}

	return topicToReviewDO{
		Order:             v.Order,
		Title:             v.Title,
		Category:          v.Category,
		Resolved:          v.Resolved,
		HotTopicId:        v.HotTopicId,
		DiscussionSources: r,
		DSSCount:          v.DSSCount,
		CommentCount:      v.CommentCount,
	}
}

type DiscussionSourceDO = discussionSourceDO

// discussionSourceToReviewDODO
type discussionSourceToReviewDO struct {
	DiscussionSourceDO `bson:",inline"`

	Closed bool `bson:"closed"    json:"closed"  `
}

func (do *discussionSourceToReviewDO) toDiscussionSourceToReview() domain.DiscussionSourceToReview {
	return domain.DiscussionSourceToReview{
		Closed:           do.Closed,
		DiscussionSource: do.DiscussionSourceDO.toDiscussionSource(),
	}
}

func todiscussionSourceToReviewDO(v *domain.DiscussionSourceToReview) discussionSourceToReviewDO {
	return discussionSourceToReviewDO{
		Closed:             v.Closed,
		DiscussionSourceDO: todiscussionSourceDO(&v.DiscussionSource),
	}
}
