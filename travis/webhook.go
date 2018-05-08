package travis

import "time"

// https://docs.travis-ci.com/user/notifications/#Webhooks-Delivery-Format

const (
	statusSuccess    = 0
	statusNotSuccess = 1
)

type Webhook struct {
	ID                int        `json:"id"`
	Number            string     `json:"number"`
	Type              string     `json:"type"`
	State             string     `json:"state"`
	Status            int        `json:"status"` // status and result are the same
	Result            int        `json:"result"`
	StatusMessage     string     `json:"status_message"` // status_message and result_message are the same
	ResultMessage     string     `json:"result_message"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        time.Time  `json:"finished_at"`
	Duration          int        `json:"duration"`
	BuildURL          string     `json:"build_url"`
	CommitID          int        `json:"commit_id"`
	Commit            string     `json:"commit"`
	BaseCommit        string     `json:"base_commit"`
	HeadCommit        string     `json:"head_commit"`
	Branch            string     `json:"branch"`
	Message           string     `json:"message"`
	CompareURL        string     `json:"compare_url"`
	CommittedAt       time.Time  `json:"committed_at"`
	AuthorName        string     `json:"author_name"`
	AuthorEmail       string     `json:"author_email"`
	CommitterName     string     `json:"committer_name"`
	CommitterEmail    string     `json:"committer_email"`
	PullRequest       bool       `json:"pull_request"`
	PullRequestNumber int        `json:"pull_request_number"`
	PullRequestTitle  string     `json:"pull_request_title"`
	Tag               string     `json:"tag"`
	Repository        Repository `json:"repository"`
}

type Repository struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	OwnerName string `json:"owner_name"`
	URL       string `json:"url"`
}

// IsMatch make sure the webhook is for you...
func (w Webhook) IsMatch(branch string, repo string, owner string) bool {
	return w.Branch == branch &&
		w.Repository.Name == repo &&
		w.Repository.OwnerName == owner
}

func (w Webhook) ShouldDeploy() bool {
	// when travis builds a pull request, Branch is the target branch, not the origin branch
	// source: https://docs.travis-ci.com/user/environment-variables/#Default-Environment-Variables
	return w.Status == statusSuccess && w.Branch == "master" && !w.PullRequest
}

func (w Webhook) DeploySummary() string {
	return w.Commit[:8] + ": " + w.Message
}
