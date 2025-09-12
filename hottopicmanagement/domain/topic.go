package domain

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/opensourceways/hot-topic-website-backend/utils"
)

const (
	statusNew      = "New"
	statusAppended = "Appended"
	statusResolved = "Resolved"
)

type DiscussionSourceMeta struct {
	Id        	int    	`json:"id"           required:"true"`
	URL       	string 	`json:"url"          required:"true"`
	Type      	string 	`json:"source_type"  required:"true"`
	Title     	string 	`json:"title"        required:"true"`
	SourceId  	string 	`json:"source_id"    required:"true"`
	CreatedAt 	string 	`json:"created_at"   required:"true"`
	Company		string 	`json:"company"`
	CommentNum 	int 	`json:"comment_num"`
}

type DiscussionSource struct {
	DiscussionSourceMeta

	ImportedAt string `json:"imported_at"`
}

func (ds *DiscussionSource) isOldOne() bool {
	return ds.ImportedAt != ""
}

func (ds *DiscussionSource) setImportDate(date string) {
	if !ds.isOldOne() {
		ds.ImportedAt = date
	}
}

func findMaxDate(items []DiscussionSource) (time.Time, error) {
	if len(items) == 0 {
		return time.Time{}, fmt.Errorf("no dss")
	}

	maxTime, err := time.Parse(time.RFC3339, items[0].CreatedAt)
	if err != nil {
		return time.Time{}, err
	}

	items = items[1:]
	for i := range items {
		t, err := time.Parse(time.RFC3339, items[i].CreatedAt)
		if err != nil {
			return time.Time{}, err
		}

		if t.After(maxTime) {
			maxTime = t
		}
	}

	return maxTime, nil
}

type StatusLog struct {
	Status string
	Time   string
}

func (s *StatusLog) resolved() bool {
	return s.Status == statusResolved
}

type TransferLog struct {
	StatusLog
	Order int   // the topic is ordered on the report of that week
	Date  int64 // the date that the report of that week is created
}

func (log *TransferLog) getDate() int64 {
	return log.Date
}

func newAppendedLog(items []DiscussionSource, date string, aWeekAgo time.Time) (log StatusLog) {
	log.Status = statusAppended

	v, err := findMaxDate(items)
	if err != nil {
		logrus.Errorf("find max date failed, err:%v", err)
	}

	if err == nil && v.After(aWeekAgo) {
		log.Time = utils.GetDate(&v)
	} else {
		log.Time = date
	}

	return
}

// HotTopic
type HotTopic struct {
	Id                	string
	Title             	string
	TransferLogs      	[]TransferLog
	DiscussionSources 	[]DiscussionSource
	Version           	int
	DSSCount        	int
	CommentCount      	int
}

func (ht *HotTopic) GetStatus(date int64) TransferLog {
	for i := range ht.TransferLogs {
		if ht.TransferLogs[i].Date == date {
			return ht.TransferLogs[i]
		}
	}

	return TransferLog{}
}

func (ht *HotTopic) UpdatedAt() int64 {
	if n := len(ht.TransferLogs); n > 0 {
		return ht.TransferLogs[n-1].Date
	}

	return 0
}

func (ht *HotTopic) Order() int {
	if n := len(ht.TransferLogs); n > 0 {
		return ht.TransferLogs[n-1].Order
	}

	return 0
}

func (ht *HotTopic) update(r *TopicToReview, date int64, datestr string, aWeekAgo time.Time) bool {
	updatedAt := ht.UpdatedAt()
	if updatedAt == 0 {
		// it is impossible that there aren't old logs
		return false
	}

	if updatedAt == date {
		// for this case, the hot topis of this week are confirmed,
		// so it is just to avoid update the hot topic repeatly
		// because it has been saved.

		logrus.Info("it is repeated to update the hot topic")

		return false
	}

	items, all := r.initAppendedDS(datestr)
	ht.DiscussionSources = all

	log := TransferLog{
		Date:  date,
		Order: r.Order,
	}
	if r.Resolved {
		log.Status = statusResolved
		log.Time = datestr
	} else if len(items) > 0 {
		log.StatusLog = newAppendedLog(items, datestr, aWeekAgo)
	} else {
		log.StatusLog = ht.TransferLogs[len(ht.TransferLogs)-1].StatusLog
	}

	ht.TransferLogs = append(ht.TransferLogs, log)
	ht.Title = r.Title

	return true
}

func (ht *HotTopic) InitReview(t *TopicToReview) error {
	m := t.getDSMap()

	for i := range ht.DiscussionSources {
		item := ht.DiscussionSources[i]

		v, ok := m[item.Id]
		if !ok {
			return fmt.Errorf(
				"missing discussion source(%d) for the reviewing topic(%s)", v, t.Title,
			)
		}
		v.ImportedAt = item.ImportedAt
	}

	t.Order = ht.Order()
	t.HotTopicId = ht.Id

	return nil
}

func (ht *HotTopic) GetDSSet() map[int]bool {
	v := make(map[int]bool, len(ht.DiscussionSources))

	for i := range ht.DiscussionSources {
		v[ht.DiscussionSources[i].Id] = true
	}

	return v
}

func (ht *HotTopic) CreatedAt() int64 {
	if len(ht.TransferLogs) == 0 {
		return 0
	}

	return ht.TransferLogs[0].getDate()
}

func (ht *HotTopic) ResolvedAt() int64 {
	if !ht.isResolved() {
		return 0
	}

	n := len(ht.TransferLogs)

	return ht.TransferLogs[n-1].getDate()
}

func (ht *HotTopic) isResolved() bool {
	n := len(ht.TransferLogs)

	return n > 0 && ht.TransferLogs[n-1].resolved()
}

func (ht *HotTopic) GetDiscussionSource(dsId int) *DiscussionSource {
	for i := range ht.DiscussionSources {
		if item := &ht.DiscussionSources[i]; item.Id == dsId {
			return item
		}
	}

	return nil
}

// DiscussionSourceInfo
type DiscussionSourceInfo struct {
	Id     int    `json:"id"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Closed bool   `json:"closed"`

	removed bool `json:"-"`
}

func (info *DiscussionSourceInfo) Removed() bool {
	return info.removed
}

// NotHotTopic
type NotHotTopic struct {
	Title             string
	Category          string
	DiscussionSources []DiscussionSourceInfo
}

func (nht *NotHotTopic) IsWorthless(newTopicCategory string) bool {
	return nht.Category != "" && nht.Category != newTopicCategory && nht.isClosed()
}

func (nht *NotHotTopic) isClosed() bool {
	for i := range nht.DiscussionSources {
		if !nht.DiscussionSources[i].Closed {
			return false
		}
	}

	return true
}

func (nht *NotHotTopic) GetDSSet() map[int]bool {
	v := make(map[int]bool, len(nht.DiscussionSources))

	for i := range nht.DiscussionSources {
		v[nht.DiscussionSources[i].Id] = true
	}

	return v
}

func (nht *NotHotTopic) UpdateRemoved(dsIdsOfNewTopic map[int]bool) {
	for i := range nht.DiscussionSources {
		item := &nht.DiscussionSources[i]

		if _, ok := dsIdsOfNewTopic[item.Id]; !ok {
			item.removed = true
		}
	}
}

func (nht *NotHotTopic) Sort() []*DiscussionSourceInfo {
	v := make([]*DiscussionSourceInfo, len(nht.DiscussionSources))
	h := 0
	t := len(v) - 1
	for i := range nht.DiscussionSources {
		if item := &nht.DiscussionSources[i]; item.removed {
			v[h] = item
			h++
		} else {
			v[t] = item
			t--
		}
	}

	return v
}
