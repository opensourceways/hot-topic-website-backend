package watchhottopic

import (
	"slices"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/opensourceways/hot-topic-website-backend/common/domain/allerror"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/app"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain/repository"
	"github.com/opensourceways/hot-topic-website-backend/utils"
)

var NoInvokeCommunity = []string{
	"vllm", "unifiedbus", "openeuler", "mindcluster", "mindie", "mindsdk", "mindstudio", "pta",
	"openubmc", "mindspeed", "pytorch", "triton", "sglang", "verl", "tilelang", "sgl",
	"mindspore", "openfuyao", "ascendnpuir", "cannopen",
}

func newdoneCache() doneCache {
	return doneCache{
		communities: map[string]bool{},
	}
}

// doneCache
type doneCache struct {
	date        int64
	communities map[string]bool
}

func (c *doneCache) isDone(community string) bool {
	return c.communities[community]
}

func (c *doneCache) add(community string) {
	c.communities[community] = true
}

func (c *doneCache) refresh(date int64) {
	if c.date != date {
		c.date = date
		c.communities = map[string]bool{}
	}
}

// newHandler
func newHandler(
	app app.AppService,
	repo repository.RepoNotHotTopic,
	communities []string,
) *handler {
	return &handler{
		app:         app,
		repo:        repo,
		cache:       newdoneCache(),
		communities: communities,
	}
}

// handler
type handler struct {
	app         app.AppService
	repo        repository.RepoNotHotTopic
	cache       doneCache
	communities []string
}

func (h *handler) handle(needStop func() bool) {
	date := utils.GetLastFriday().Unix()

	h.cache.refresh(date)

	for _, community := range h.communities {
		if needStop() {
			print("stop handle")
			return
		}

		if h.cache.isDone(community) {
			print("skip community:", community)
			continue
		}

		if b, err := h.isDone(community, date, needStop); b {
			print("is done, community:", community)
			h.cache.add(community)
			continue

		} else if err != nil {
			logrus.Errorf("check if is is done failed, community:%s, err:%s", community, err.Error())

			continue
		}
		logrus.Infof("apply hot topic, community:%s", community)
		err := h.doApply(community, needStop)
		logrus.Infof("apply hot topic, community:%s, err:%v", community, err)

		if err == nil {
			h.cache.add(community)

			continue
		}

		if allerror.IsError(err, allerror.ErrorCodeInvokeTimeRestricted) {
			return
		}
	}
}

func (h *handler) isDone(community string, date int64, needStop func() bool) (bool, error) {
	if slices.Contains(NoInvokeCommunity, community) {
		return false, nil
	}
	v, err := h.findUpdatingTime(community, needStop)
	if err != nil {
		return false, err
	}

	return v == date, nil
}

func (h *handler) findUpdatingTime(community string, needStop func() bool) (v int64, err error) {
	if v, err = h.repo.FindCreatedAt(community); err == nil {
		return v, err
	}

	for i := 0; i < 2; i++ {
		if needStop() {
			return
		}

		time.Sleep(10 * time.Second)

		if v, err = h.repo.FindCreatedAt(community); err == nil {
			return v, err
		}
	}

	return
}

func (h *handler) doApply(community string, needStop func() bool) (err error) {
	if err = h.app.ApplyToHotTopic(community); err == nil {
		return
	}

	if allerror.IsError(err, allerror.ErrorCodeInvokeTimeRestricted) {
		return
	}

	for i := 0; i < 2; i++ {
		if needStop() {
			return
		}

		time.Sleep(10 * time.Second)

		if err = h.app.ApplyToHotTopic(community); err == nil {
			return
		}
	}

	return
}
