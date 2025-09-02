package watch

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain/repository"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/watch/forum"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/watch/gitcodeissue"
)

const (
	commentTemp = `你好！这个[链接](%s)的信息可能对你解决本问题有所帮助，请参考，谢谢！`
	gitURL      = `你好！这个\[链接\]\((.*)\)的信息可能对你解决本问题有所帮助，请参考，谢谢！`
	formURL     = `<p>你好！这个<a href="(.*)">链接</a>的信息可能对你解决本问题有所帮助，请参考，谢谢！</p>`
)

func genSolutionComment(solution *domain.DiscussionSource) string {
	return fmt.Sprintf(commentTemp, solution.URL)
}

type solutionComment struct {
	reg *regexp.Regexp
}

func (s solutionComment) ParseURL(comment string) string {
	matches := s.reg.FindStringSubmatch(comment)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

type platformClient interface {
	SholdIgnore(ds *domain.DiscussionSource) (bool, error)
	CountCommentedSolutons(*domain.DiscussionSource) ([]string, error)
	AddSolution(ds *domain.DiscussionSource, comment string) error
}

func newClients(cfg *Config) clients {
	cli := clients{}

	sc := solutionComment{regexp.MustCompile(formURL)}
	for i := range cfg.Forums {
		item := cfg.Forums[i]

		cli[cli.key(item.Community, item.typeDesc())] = forum.NewClient(&item.Detail, sc)
	}

	sc = solutionComment{regexp.MustCompile(gitURL)}
	for i := range cfg.GitCodes {
		item := cfg.GitCodes[i]

		cli[cli.key(item.Community, item.typeDesc())] = gitcodeissue.NewClient(&item.Detail, sc)
	}

	return cli
}

// key is {community}_{discussion source type}
type clients map[string]platformClient

func (cli clients) key(community, t string) string {
	return strings.ToLower(fmt.Sprintf("%s_%s", community, t))
}

func (cli clients) get(community string, ds *domain.DiscussionSource) (platformClient, error) {
	v := cli[cli.key(community, ds.Type)]
	if v == nil {
		return nil, fmt.Errorf("no client for %s and %s", community, ds.Type)
	}

	return v, nil
}

// doneCounter
type doneCounter struct {
	urls      map[string]bool
	expiredAt int64
}

func (c *doneCounter) add(v string) {
	c.urls[v] = true
	c.setExpired()
}

func (c *doneCounter) isTooMany() bool {
	return len(c.urls) >= 3
}

func (c *doneCounter) canDo(v string) bool {
	if c.isTooMany() {
		return false
	}

	return !c.urls[v]
}

func (c *doneCounter) isExpired(now int64) bool {
	return c.expiredAt > 0 && c.expiredAt < now
}

func (c *doneCounter) setExpired() {
	if c.isTooMany() {
		c.expiredAt = expiry(24 * 5)
	}
}

func newdoneCounter(urls []string) doneCounter {
	m := map[string]bool{}
	for _, url := range urls {
		m[url] = true
	}
	v := doneCounter{urls: m}
	v.setExpired()

	return v
}

func expiry(expiry int64) int64 {
	return time.Now().Add(time.Hour * time.Duration(expiry)).Unix()
}

// key is {community}_{discussion id}; value is the num that solution was commented on it
type doneCache map[string]*doneCounter

func (c doneCache) key(community string, ds *domain.DiscussionSource) string {
	return fmt.Sprint("%s_%d", community, ds.Id)
}

func (c doneCache) refresh() {
	now := time.Now().Unix()

	for k, v := range c {
		if v.isExpired(now) {
			delete(c, k)
		}
	}
}

func (c doneCache) get(cli platformClient, community string, ds *domain.DiscussionSource) (*doneCounter, error) {
	k := c.key(community, ds)

	if counter, ok := c[k]; ok {
		return counter, nil
	}

	urls, err := cli.CountCommentedSolutons(ds)
	if err != nil {
		return nil, err
	}

	v := newdoneCounter(urls)
	c[k] = &v

	return &v, nil
}

func newtopicSolutionHandler(repo repository.RepoHotTopic, cfg *Config) *topicSolutionHandler {
	return &topicSolutionHandler{
		repo:    repo,
		cache:   doneCache{},
		clients: newClients(cfg),
	}
}

// topicSolutionHandler
type topicSolutionHandler struct {
	repo    repository.RepoHotTopic
	cache   doneCache
	clients clients
}

func (h *topicSolutionHandler) handle(
	solution *repository.TopicSolutions, needStop func() bool,
) (retry []domain.TopicSolution) {
	for i := range solution.TopicSolutions {
		if needStop() {
			return
		}

		ts := &solution.TopicSolutions[i]

		topic, err := h.repo.Find(solution.Community, ts.TopicId)
		if err != nil {
			logrus.Warnf(
				"find the topic(%s) of community(%s) failed, err:%s",
				ts.TopicId, solution.Community, err.Error(),
			)

			continue
		}

		toRetry := h.handleTopicSolution(solution.Community, &topic, ts.Solutions, needStop)
		if len(toRetry) > 0 {
			retry = append(retry, domain.TopicSolution{
				TopicId:   ts.TopicId,
				Solutions: toRetry,
			})
		}
	}

	h.cache.refresh()

	return
}

func (h *topicSolutionHandler) handleTopicSolution(
	community string, topic *domain.HotTopic,
	solutions []domain.DiscussionSourceSolution,
	needStop func() bool,
) (retry []domain.DiscussionSourceSolution) {
	for i := range solutions {
		if needStop() {
			return
		}

		item := &solutions[i]

		resolvedOne := h.getDiscussionSource(topic, item.ResolvedOne)
		if resolvedOne == nil {
			continue
		}

		toRetry := []int{}
		for _, dsId := range item.RelatedOnes {
			ds := h.getDiscussionSource(topic, dsId)
			if ds == nil {
				continue
			}

			if b, err := h.handleDiscussionSourceSolution(community, resolvedOne, ds); err != nil {
				logrus.Errorf(
					"handle solution(%s) for discussion source() failed, err:%s",
					resolvedOne.URL, ds.URL, err.Error(),
				)
				// can retry
				if b {
					toRetry = append(toRetry, dsId)
				}
			}

			if needStop() {
				return
			}
		}

		if len(toRetry) > 0 {
			retry = append(retry, domain.DiscussionSourceSolution{
				ResolvedOne: item.ResolvedOne,
				RelatedOnes: toRetry,
			})
		}
	}

	return
}

func (h *topicSolutionHandler) getDiscussionSource(topic *domain.HotTopic, dsId int) *domain.DiscussionSource {
	ds := topic.GetDiscussionSource(dsId)
	if ds == nil {
		logrus.Errorf(
			"can't find the DiscussionSource(%d) from topic:%s:%s",
			dsId, topic.Id, topic.Title,
		)
	}

	return ds
}

func (h *topicSolutionHandler) handleDiscussionSourceSolution(
	community string, resolvedOne, ds *domain.DiscussionSource,
) (bool, error) {
	cli, err := h.clients.get(community, ds)
	if err != nil {
		return false, err
	}

	b, err := cli.SholdIgnore(ds)
	if err != nil {
		return true, err
	}
	if b {
		return false, nil
	}

	counter, err := h.cache.get(cli, community, ds)
	if err != nil {
		return true, err
	}

	if !counter.canDo(resolvedOne.URL) {
		return false, nil
	}

	if err = cli.AddSolution(ds, genSolutionComment(resolvedOne)); err != nil {
		return true, err
	}

	logrus.Debugf("add solution: %s to %s", resolvedOne.URL, ds.URL)

	counter.add(resolvedOne.URL)

	return false, nil
}
