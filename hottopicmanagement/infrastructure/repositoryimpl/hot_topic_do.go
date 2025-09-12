package repositoryimpl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
)

const (
	fieldId        = "id"
	fieldOrder     = "order"
	fieldTitle     = "title"
	fieldVersion   = "version"
	fieldClosedAt  = "closed_at"
	fieldCreatedAt = "created_at"
)

func tohotTopicDO(v *domain.HotTopic) hotTopicDO {
	return hotTopicDO{
		Id:                v.Id,
		Title:             v.Title,
		DiscussionSources: todiscussionSourceDOs(v.DiscussionSources),
		TransferLogs:      totransferLogDOs(v.TransferLogs),
		ClosedAt:          v.ResolvedAt(),
		CreatedAt:         v.CreatedAt(),
	}
}

func todiscussionSourceDOs(items []domain.DiscussionSource) []discussionSourceDO {
	r := make([]discussionSourceDO, len(items))

	for i := range items {
		r[i] = todiscussionSourceDO(&items[i])
	}

	return r
}

func totransferLogDOs(items []domain.TransferLog) []transferLogDO {
	r := make([]transferLogDO, len(items))

	for i := range items {
		r[i] = totransferLogDO(&items[i])
	}

	return r
}

// hotTopicDO
type hotTopicDO struct {
	Id                string               `bson:"id"             json:"id"`
	Title             string               `bson:"title"          json:"title"`
	DiscussionSources []discussionSourceDO `bson:"sources"        json:"sources"`
	TransferLogs      []transferLogDO      `bson:"logs"           json:"logs"`
	ClosedAt          int64                `bson:"closed_at"      json:"closed_at"`
	CreatedAt         int64                `bson:"created_at"     json:"created_at"`
	Version           int                  `bson:"version"        json:"-"`
}

func (do *hotTopicDO) toDoc() (bson.M, error) {
	return genDoc(do)
}

func (do *hotTopicDO) index() string {
	return do.Id
}

func (do *hotTopicDO) toHotTopic() domain.HotTopic {
	return domain.HotTopic{
		Id:                do.index(),
		Title:             do.Title,
		DiscussionSources: do.toDiscussionSources(),
		TransferLogs:      do.toTransferLogs(),
		Version:           do.Version,
	}
}

func (do *hotTopicDO) toDiscussionSources() []domain.DiscussionSource {
	r := make([]domain.DiscussionSource, len(do.DiscussionSources))

	for i := range do.DiscussionSources {
		r[i] = do.DiscussionSources[i].toDiscussionSource()
	}

	return r
}

func (do *hotTopicDO) toTransferLogs() []domain.TransferLog {
	r := make([]domain.TransferLog, len(do.TransferLogs))

	for i := range do.TransferLogs {
		r[i] = do.TransferLogs[i].toTransferLog()
	}

	return r
}

// discussionSourceDO
type discussionSourceDO struct {
	Id         int    `bson:"id"           json:"id"`
	URL        string `bson:"url"          json:"url"`
	Type       string `bson:"type"         json:"type"`
	Title      string `bson:"title"        json:"title"`
	SourceId   string `bson:"source_id"    json:"source_id"`
	CreatedAt  string `bson:"created_at"   json:"created_at"`
	ImportedAt string `bson:"imported_at"  json:"imported_at"`
	Company    string `bson:"company"      json:"company"`
	CommentNum int    `bson:"comment_num"  json:"comment_num"`
}

func (do *discussionSourceDO) toDiscussionSource() domain.DiscussionSource {
	return domain.DiscussionSource{
		DiscussionSourceMeta: domain.DiscussionSourceMeta{
			Id:        do.Id,
			URL:       do.URL,
			Type:      do.Type,
			Title:     do.Title,
			SourceId:  do.SourceId,
			CreatedAt: do.CreatedAt,
			Company:   do.Company,
			CommentNum:do.CommentNum,
		},
		ImportedAt: do.ImportedAt,
	}
}

func todiscussionSourceDO(v *domain.DiscussionSource) discussionSourceDO {
	return discussionSourceDO{
		Id:         v.Id,
		URL:        v.URL,
		Type:       v.Type,
		Title:      v.Title,
		SourceId:   v.SourceId,
		CreatedAt:  v.CreatedAt,
		ImportedAt: v.ImportedAt,
		Company:    v.Company,
		CommentNum: v.CommentNum,
	}
}

// transferLogDO
type transferLogDO struct {
	Date   int64  `bson:"date"   json:"date"`
	Time   string `bson:"time"   json:"time"`
	Status string `bson:"status" json:"status"`
	Order  int    `bson:"order"  json:"order"`
}

func (do *transferLogDO) toTransferLog() domain.TransferLog {
	return domain.TransferLog{
		StatusLog: domain.StatusLog{
			Time:   do.Time,
			Status: do.Status,
		},
		Date:  do.Date,
		Order: do.Order,
	}
}

func totransferLogDO(v *domain.TransferLog) transferLogDO {
	return transferLogDO{
		Date:   v.Date,
		Time:   v.Time,
		Status: v.Status,
		Order:  v.Order,
	}
}
