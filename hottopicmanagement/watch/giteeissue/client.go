package giteeissue

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/opensourceways/hot-topic-website-backend/hottopicmanagement/domain"
)

type Config struct {
	Token string `json:"token"`
}

type solutionComment interface {
	ParseURL(comment string) string
}

type clientImpl struct {
	httpClient *http.Client
	token      string
	solutionComment
}

func NewClient(cfg *Config, sc solutionComment) *clientImpl {
	return &clientImpl{
		httpClient:      &http.Client{},
		token:           cfg.Token,
		solutionComment: sc,
	}
}

type issueInfo struct {
	owner string
	repo  string
	num   string
}

func parseIssue(ds *domain.DiscussionSource) (issueInfo, error) {
	v := strings.Split(strings.TrimSpace(ds.URL), "/")
	n := len(v) - 1
	if n < 3 {
		return issueInfo{}, errors.New("invalid ds url")
	}

	return issueInfo{
		owner: v[n-3],
		repo:  v[n-2],
		num:   v[n],
	}, nil
}

func (impl *clientImpl) makeRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+impl.token)
	req.Header.Set("Content-Type", "application/json")

	return impl.httpClient.Do(req)
}

func (impl *clientImpl) SholdIgnore(ds *domain.DiscussionSource) (bool, error) {
	return false, nil
}

func (impl *clientImpl) CountCommentedSolutons(ds *domain.DiscussionSource) ([]string, error) {
	issue, err := parseIssue(ds)
	if err != nil {
		return nil, err
	}

	urls := []string{}
	for page := 1; ; page++ {
		url := "https://gitee.com/api/v5/repos/" + issue.owner + "/" + issue.repo + "/issues/" + issue.num + "/comments?page=" + strconv.Itoa(page) + "&per_page=100"
		resp, err := impl.makeRequest(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to get comments: " + resp.Status)
		}

		var comments []struct {
			Body *string `json:"body"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
			return nil, err
		}

		if len(comments) == 0 {
			break
		}

		for _, c := range comments {
			if c.Body != nil {
				if v := impl.ParseURL(*c.Body); v != "" {
					urls = append(urls, v)
				}
			}
		}
	}

	return urls, nil
}

func (impl *clientImpl) AddSolution(ds *domain.DiscussionSource, comment string) error {
	issue, err := parseIssue(ds)
	if err != nil {
		return err
	}

	url := "https://gitee.com/api/v5/repos/" + issue.owner + "/" + issue.repo + "/issues/" + issue.num + "/comments"
	body := struct {
		Body string `json:"body"`
	}{comment}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := impl.makeRequest(context.Background(), http.MethodPost, url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.New("failed to add comment: " + resp.Status)
	}

	return nil
}
