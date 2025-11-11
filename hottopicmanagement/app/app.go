package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/opensourceways/hot-topic-website-backend/common/domain/allerror"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain/repository"
	"github.com/opensourceways/hot-topic-website-backend/utils"
	"github.com/sirupsen/logrus"
)

type AppService interface {
	NewReviews(string, CmdToUploadOptionalTopics) error
	GetHotTopics(community string, since int64) (HotTopicsDTO, error)
	UpdateSelected(string, *CmdToUpdateSelected) error
	ApplyToHotTopic(community string) error
	GetTopicsToReview(community string) (TopicsToReviewDTO, error)
	GetTopicsToPublish(community string) (dto HotTopicsDTO, err error)
	GetWorthlessNotHotTopic(community string) (NotHotTopicsDTO, error)
}

var NoInvokeCommunity = []string{
	"vllm", "unifiedbus", "openeuler", "mindcluster", "mindie", "mindsdk", "mindstudio", "pta",
	"openubmc", "mindspeed", "pytorch", "triton", "sglang", "verl", "tilelang", "sgl",
	"mindspore", "openfuyao", "ascendnpuir", "cannopen",
}

func NewAppService(
	config *Config,
	repoHotTopic repository.RepoHotTopic,
	repoNotHotTopic repository.RepoNotHotTopic,
	repoTopicsToReview repository.RepoTopicsToReview,
) *appService {
	return &appService{
		cfg:                *config,
		repoHotTopic:       repoHotTopic,
		repoNotHotTopic:    repoNotHotTopic,
		repoTopicsToReview: repoTopicsToReview,
	}
}

type appService struct {
	cfg                Config
	repoHotTopic       repository.RepoHotTopic
	repoNotHotTopic    repository.RepoNotHotTopic
	repoTopicsToReview repository.RepoTopicsToReview
}

func (s *appService) reviewFile(community string) string {
	if !s.cfg.SaveToFile {
		return ""
	}

	return filepath.Join(s.cfg.FilePath, fmt.Sprintf("%s_%s.xlsx", community, utils.Date()))
}

func (s *appService) checkInvokeByTime(times []time.Weekday, community string) error {
	if slices.Contains(NoInvokeCommunity, community) {
		return nil
	}
	logrus.Infof("the community is %s", community)
	if !s.cfg.EnableInvokeRestriction {
		return nil
	}

	w := utils.Now().Weekday()

	for _, t := range times {
		if w == t {
			return nil
		}
	}

	desc := make([]string, len(times))
	for i, t := range times {
		desc[i] = t.String()
	}

	return allerror.New(
		allerror.ErrorCodeInvokeTimeRestricted,
		fmt.Sprintf("must invoke on %s", strings.Join(desc, " or ")), nil,
	)

}

func (s *appService) NewReviews(community string, cmd CmdToUploadOptionalTopics) error {
	if err := s.checkInvokeByTime([]time.Weekday{time.Friday}, community); err != nil {
		return err
	}

	cmd.init()

	file, err := newfileToReview(s.reviewFile(community))
	if err != nil {
		return fmt.Errorf("new excel failed, err:%s", err.Error())
	}

	toReview := domain.NewTopicsToReview(utils.GetLastFriday().Unix())

	newOnes, err := s.handleOldTopics(community, cmd, file, &toReview)
	if err != nil {
		return err
	}

	if err := s.handleNewOptionalTopics(community, newOnes, file, &toReview); err != nil {
		return err
	}

	if err := file.saveToFile(); err != nil {
		return err
	}

	return s.repoTopicsToReview.Add(community, &toReview)
}

func (s *appService) handleOldTopics(
	community string, cmd CmdToUploadOptionalTopics, file *fileToReview, tr *domain.TopicsToReview,
) (
	[]*OptionalTopic, error,
) {
	oldTopics, err := s.repoHotTopic.FindOpenOnes(community)
	if err != nil {
		return nil, err
	}
	if len(oldTopics) == 0 {
		newOnes := make([]*OptionalTopic, len(cmd))
		for i := range cmd {
			newOnes[i] = &cmd[i]
		}

		fmt.Println("no old topics")

		return newOnes, nil
	}

	oldTopicsSets := make(map[string]bool, len(oldTopics))
	for i := range oldTopics {
		item := &oldTopics[i]
		oldTopicsSets[item.Title] = true
	}

	oldOnes := make(map[string]*OptionalTopic, len(cmd))
	newOnes := make([]*OptionalTopic, 0, len(cmd))

	for i := range cmd {
		item := &cmd[i]

		if _, ok := oldTopicsSets[item.Title]; ok {
			oldOnes[item.Title] = item
		} else {
			newOnes = append(newOnes, item)
		}
	}

	if n := len(oldTopics); len(oldOnes) != n {
		return nil, fmt.Errorf(
			"the count of old topics is not matched, expect :%d, actual:%d", n, len(oldOnes),
		)
	}

	if err := file.saveLastHotTopics(oldTopics, oldOnes); err != nil {
		return nil, err
	}

	tr.Selected, err = s.toSelected(oldTopics, oldOnes, sheetLastTopics)
	if err != nil {
		return nil, err
	}

	return newOnes, nil
}

func (s *appService) handleNewOptionalTopics(
	community string, newOnes []*OptionalTopic, file *fileToReview, candidate *domain.TopicsToReview,
) error {
	if len(newOnes) == 0 {
		return errors.New("no new topics")
	}

	oldTopics, err := s.repoNotHotTopic.FindAll(community)
	if err != nil {
		return err
	}
	if len(oldTopics) == 0 {
		for i := range newOnes {
			if err := file.saveNewTopic(newOnes[i]); err != nil {
				return err
			}
		}

		for i := range newOnes {
			v := newOnes[i].toTopicToReview("")
			candidate.AddCandidate(sheetNewTopics, &v)
		}

		return nil
	}

	oldTopicsSet := make([]map[int]bool, len(oldTopics))
	for i := range oldTopics {
		oldTopicsSet[i] = oldTopics[i].GetDSSet()
	}

	newTopicsSet := make([]map[int]bool, len(newOnes))
	for i := range newOnes {
		newTopicsSet[i] = newOnes[i].getDSSet()
	}

	new2old, old2new := findRelationsBetweenCategories(newTopicsSet, oldTopicsSet)

	for i, v := range new2old {
		n := len(v)

		cv := newOnes[i].toTopicToReview("")

		if n == 0 {
			if err := file.saveNewTopic(newOnes[i]); err != nil {
				return err
			}

			candidate.AddCandidate(sheetNewTopics, &cv)

			continue
		}

		if j := v[0]; n == 1 && len(old2new[j]) == 1 {
			ns := newTopicsSet[i]
			os := oldTopicsSet[j]

			switch parseRelationshipBetweenSets(ns, os) {
			case setsRelationSame:
				if err := file.saveUnchangedTopic(newOnes[i]); err != nil {
					return err
				}

				candidate.AddCandidate(sheetUnchangedTopics, &cv)

			case setsRelationLeftIncludesRight:
				// must run before file.saveTopicThatAppendToOld to avoid setting Appended
				candidate.AddCandidate(sheetAppendToOld, &cv)

				if err := file.saveTopicThatAppendToOld(newOnes[i], os); err != nil {
					return err
				}

			case setsRelationRightIncludesLeft:
				if err := file.saveTopicThatRemoveFromOld(&oldTopics[j], ns); err != nil {
					return err
				}

			default:
				if err := file.saveTopicThatIntersectWithMultiOlds(newOnes[i], v, oldTopics); err != nil {
					return err
				}

				candidate.AddCandidate(sheetMultiIntersects, &cv)
			}

			continue
		}

		if err := file.saveTopicThatIntersectWithMultiOlds(newOnes[i], v, oldTopics); err != nil {
			return err
		}

		candidate.AddCandidate(sheetMultiIntersects, &cv)

	}

	for j, v := range old2new {
		if len(v) == 0 {
			if err := file.saveTopicThatRemoveFromOld(&oldTopics[j], map[int]bool{}); err != nil {
				return err
			}
		}
	}

	return nil
}
