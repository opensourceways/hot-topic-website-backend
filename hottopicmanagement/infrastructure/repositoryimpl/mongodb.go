package repositoryimpl

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	mongodbCmdOr        = "$or"
	mongodbCmdIn        = "$in"
	mongodbCmdLte       = "$lte"
	mongodbCmdGte       = "$gte"
	mongodbCmdGt	   	= "$gt"
	mongodbCmdElemMatch = "$elemMatch"
)

type anyDoc = map[string]string

type Dao interface {
	IsDocNotExists(error) bool
	IsDocExists(error) bool

	NewDocId() string
	DocIdFilter(s string) (bson.M, error)
	DocIdsFilter(ids []string) (bson.M, error)

	UpdateDoc(filter bson.M, doc bson.M, version int) error
	UpdateDocsWithoutVersion(filter bson.M, doc bson.M) error
	PushArraySingleItem(filter bson.M, field string, doc interface{}, version int) error
	PushArrayMultiItems(filter bson.M, array string, value bson.A, version int) error
	PullArrayMultiItems(filter bson.M, array string, filterOfItem bson.M, version int) error
	UpdateArraySingleItem(filter bson.M, array string, filterOfArray, doc bson.M, version int) error
	PushArraySingleItemAndUpdate(filter bson.M, array string, v interface{}, u bson.M, version int) error
	MoveArrayItem(filter bson.M, from string, filterOfItem bson.M, to string, value bson.M, version int) error

	InsertDoc(doc bson.M) (string, error)
	ReplaceDoc(filter, doc bson.M) (string, error)
	InsertDocIfNotExists(filter, doc bson.M) (string, error)

	DeleteDoc(filter bson.M) error
	DeleteDocs(filter bson.M) error

	GetDoc(filter, project, sort bson.M, result interface{}) error
	GetDocs(filter, project, sort bson.M, limit int64, result interface{}) error
	GetDocAndDelete(filter, project bson.M, result interface{}) error
	GetArrayItem(filter bson.M, array string, filterOfArray, project bson.M, result interface{}) error
}

func genDoc(doc interface{}) (m bson.M, err error) {
	v, err := json.Marshal(doc)
	if err != nil {
		return
	}

	err = json.Unmarshal(v, &m)

	return
}

func childField(fields ...string) string {
	return strings.Join(fields, ".")
}

type daoMap map[string]Dao

func (m daoMap) dao(community string) (Dao, error) {
	v, ok := m[community]
	if !ok {
		return nil, fmt.Errorf("can't find dao for %s", community)
	}

	return v, nil
}
