package domain

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/opensourceways/hot-topic-website-backend/common/domain/allerror"
	"github.com/opensourceways/hot-topic-website-backend/utils"
)

// DiscussionSourceToReview
type DiscussionSourceToReview struct {
	Closed bool `json:"source_closed"`

	DiscussionSource
}

func (r *DiscussionSourceToReview) toDiscussionSourceInfo() DiscussionSourceInfo {
	return DiscussionSourceInfo{
		Id:     r.Id,
		URL:    r.URL,
		Title:  r.Title,
		Closed: r.Closed,
	}
}

// TopicToReview
type TopicToReview struct {
	// it is not empty if the topic is the last host topic
	HotTopicId        string                     `json:"ht_id"`
	Order             int                        `json:"order"`
	Title             string                     `json:"title"`
	Category          string                     `json:"category"`
	Resolved          bool                       `json:"resolved"`
	DSSCount          int                        `json:"dss_count"`
	CommentCount      int                        `json:"comment_count"`
	CommenterNum      int                        `json:"commenter_num"`
	DiscussionSources []DiscussionSourceToReview `json:"dss"`
}

func (r *TopicToReview) newHotTopic(dateSec int64, date string) HotTopic {
	dss := make([]DiscussionSource, len(r.DiscussionSources))
	commentCount := 0
	commenterNum := 0
	for i := range r.DiscussionSources {
		item := &r.DiscussionSources[i].DiscussionSource
		item.ImportedAt = date
		commentCount += item.CommentNum
		commenterNum += item.CommenterNum
		dss[i] = *item
	}

	logTime := date
	if t, err := findMaxDate(dss); err == nil {
		logTime = utils.GetDate(&t)
	}

	status := StatusLog{
		Time:   logTime,
		Status: statusNew,
	}
	if r.Resolved {
		status.Status = statusResolved
	}

	return HotTopic{
		Id:    r.HotTopicId,
		Title: r.Title,
		TransferLogs: []TransferLog{
			TransferLog{
				Order:     r.Order,
				Date:      dateSec,
				StatusLog: status,
			},
		},
		DiscussionSources: dss,
		DSSCount:          r.dsNum(),
		CommentCount:      commentCount,
		CommenterNum:      commenterNum,
	}
}

func (r *TopicToReview) newNotHotTopic(selectedDS map[int]bool) (NotHotTopic, bool) {
	v := make([]DiscussionSourceInfo, 0, len(r.DiscussionSources))
	commentCount := 0
	commenterNum := 0

	for i := range r.DiscussionSources {
		if item := &r.DiscussionSources[i]; !selectedDS[item.Id] {
			v = append(v, item.toDiscussionSourceInfo())
			commentCount += item.CommentNum
			commenterNum += item.CommenterNum
		}
	}

	if len(v) == 0 {
		return NotHotTopic{}, false
	}

	return NotHotTopic{
		Title:             r.Title,
		Category:          r.Category,
		DiscussionSources: v,
		DSSCount:          r.dsNum(),
		CommentCount:      r.CommentCount,
		CommenterNum:      r.CommenterNum,
	}, true
}

func (r *TopicToReview) initAppendedDS(date string) (appends []DiscussionSource, all []DiscussionSource) {
	n := r.dsNum()
	all = make([]DiscussionSource, n)
	appends = make([]DiscussionSource, 0, n)

	for i := range r.DiscussionSources {
		item := &r.DiscussionSources[i]
		if !item.isOldOne() {
			item.ImportedAt = date

			appends = append(appends, item.DiscussionSource)
		}

		all[i] = item.DiscussionSource
	}

	return
}

func (r *TopicToReview) dsNum() int {
	return len(r.DiscussionSources)
}

func (t *TopicToReview) GetDSSet() map[int]bool {
	v := make(map[int]bool, len(t.DiscussionSources))

	for i := range t.DiscussionSources {
		v[t.DiscussionSources[i].Id] = true
	}

	return v
}

func (t *TopicToReview) gatherDS(v map[int]bool) {
	for i := range t.DiscussionSources {
		v[t.DiscussionSources[i].Id] = true
	}
}

func (t *TopicToReview) getDSMap() map[int]*DiscussionSourceToReview {
	v := make(map[int]*DiscussionSourceToReview, len(t.DiscussionSources))

	for i := range t.DiscussionSources {
		item := &t.DiscussionSources[i]
		v[item.Id] = item
	}

	return v
}

func (t *TopicToReview) getOldDS() map[int]bool {
	oldOnes := make(map[int]bool, len(t.DiscussionSources))

	for i := range t.DiscussionSources {
		if item := &t.DiscussionSources[i]; item.isOldOne() {
			oldOnes[item.Id] = true
		}
	}

	return oldOnes
}

func (t *TopicToReview) checkForReview(t1 *TopicToReview) error {
	oldOne := t.getOldDS()
	oldOne1 := t1.getOldDS()

	err := allerror.New(allerror.ErrorCodeMissingDS, "missing old ds", nil)

	if len(oldOne) != len(oldOne1) {
		return err
	}

	for k := range oldOne {
		if !oldOne1[k] {
			return err
		}
	}

	return nil
}

// TopicsToReview
type TopicsToReview struct {
	Candidates map[string][]TopicToReview `json:"cadidates"`
	Selected   []TopicToReview            `json:"selected"`
	Version    int                        `json:"-"`
	Date       int64                      `json:"-"`
}

func NewTopicsToReview(date int64) TopicsToReview {
	return TopicsToReview{
		Candidates: map[string][]TopicToReview{},
		Date:       date,
	}
}

func (t *TopicsToReview) IsMatchedReview(date int64) bool {
	return t.Date == date
}

func (t *TopicsToReview) CandidatesNum() int {
	n := 0

	for i := range t.Candidates {
		n += len(t.Candidates[i])
	}

	return n
}

func (t *TopicsToReview) UpdateSelected(lastHotTopic string, items []TopicToReview, newId func() string) error {
	lastTopics := map[string]*TopicToReview{}
	for i := range t.Selected {
		if item := &t.Selected[i]; item.Category == lastHotTopic {
			lastTopics[item.HotTopicId] = item
		}
	}

	n := 0
	for i := range items {
		item := &items[i]

		if old, ok := lastTopics[item.HotTopicId]; ok {
			n++

			if err := old.checkForReview(item); err != nil {
				return err
			}

		} else if item.HotTopicId == "" {
			item.HotTopicId = newId()
		}
	}

	if n != len(lastTopics) {
		return allerror.New(allerror.ErrorCodeMissingHT, "missing some last hot topics", nil)
	}

	t.Selected = items

	return nil
}

func (t *TopicsToReview) AddCandidate(category string, topic *TopicToReview) {
	topic.Category = category

	t.Candidates[category] = append(t.Candidates[category], *topic)
}

func (t *TopicsToReview) GenNotHotTopics() []NotHotTopic {
	selectedDS := map[int]bool{}
	for i := range t.Selected {
		t.Selected[i].gatherDS(selectedDS)
	}

	n := 0
	for _, items := range t.Candidates {
		n += len(items)
	}

	r := make([]NotHotTopic, 0, n)

	for _, items := range t.Candidates {
		for i := range items {
			if v, ok := items[i].newNotHotTopic(selectedDS); ok {
				r = append(r, v)
			}
		}
	}

	logrus.Infof("before there is %d topics, at last there is %d topics", n, len(r))

	return r
}

func (r *TopicsToReview) FilterChangedAndNews(hts []HotTopic, date time.Time) (
	[]*HotTopic, []HotTopic,
) {
	changed := make([]*HotTopic, 0, len(hts))
	news := make([]HotTopic, 0, len(r.Selected))

	htMap := make(map[string]*HotTopic, len(hts))
	for i := range hts {
		item := &hts[i]
		htMap[item.Id] = item
	}

	dateSec := date.Unix()
	dateStr := utils.GetDate(&date)
	aWeekAgo := date.AddDate(0, 0, -7)

	for i := range r.Selected {
		item := &r.Selected[i]

		if ht, ok := htMap[item.HotTopicId]; ok {
			if ht.update(item, dateSec, dateStr, aWeekAgo) {
				changed = append(changed, ht)
			}
		} else {
			news = append(news, item.newHotTopic(dateSec, dateStr))
		}
	}

	return changed, news
}
