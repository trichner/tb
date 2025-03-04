package jira

import (
	"fmt"

	"github.com/trichner/tb/pkg/jira/credentials"
	gojira "gopkg.in/andygrunwald/go-jira.v1"
)

type Issue struct {
	Key         string       `json:"key"`
	Summary     string       `json:"summary"`
	Type        string       `json:"type"`
	Assignee    *string      `json:"assignee"`
	Squad       *string      `json:"squad"`
	FixVersion  *string      `json:"fixVersion"`
	Status      *string      `json:"status"`
	Labels      []string     `json:"labels"`
	StoryPoints *int         `json:"storyPoints"`
	IssueLinks  []*IssueLink `json:"issueLinks"`
}

type Version struct {
	Name        string `json:"name"`
	ReleaseDate string `json:"releaseDate"`
	Released    bool   `json:"released"`
}

type IssueLinkDirection string

const (
	IssueLinkDirectionUnknown = IssueLinkDirection("")
	IssueLinkDirectionInward  = IssueLinkDirection("INWARD")
	IssueLinkDirectionOutward = IssueLinkDirection("OUTWARD")
)

type IssueLink struct {
	Direction    IssueLinkDirection `json:"direction"`
	Relation     string             `json:"relation"`
	IssueKey     string             `json:"issueKey"`
	IssueSummary string             `json:"issueSummary"`
}

type JiraService struct {
	client *gojira.Client
}

func NewJiraServiceWithDefaultCredentials(baseUrl string) (*JiraService, error) {
	cred, err := credentials.FindCredentials()
	if err != nil {
		return nil, err
	}

	tp := gojira.BasicAuthTransport{
		Username: cred.Username,
		Password: cred.Token,
	}

	client, err := gojira.NewClient(tp.Client(), baseUrl)
	if err != nil {
		return nil, err
	}

	service := &JiraService{
		client: client,
	}

	return service, nil
}

func NewJiraService(baseUrl string, username string, token string) (*JiraService, error) {
	tp := gojira.BasicAuthTransport{
		Username: username,
		Password: token,
	}

	client, err := gojira.NewClient(tp.Client(), baseUrl)
	if err != nil {
		return nil, err
	}

	service := &JiraService{
		client: client,
	}

	return service, nil
}

func (j *JiraService) GetByKey(key string) (*Issue, error) {
	issues, _, err := j.client.Issue.Search(fmt.Sprintf("key = %s", key), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot find issue %q: %w", key, err)
	}

	if len(issues) == 0 {
		return nil, nil
	}
	issueVos := mapJiraListToVoList(issues)
	if len(issueVos) == 1 {
		return &issueVos[0], nil
	}

	return nil, fmt.Errorf("found multiple issues for the same key: " + key)
}

func (j *JiraService) GetVersion(projectId, version string) (*Version, error) {
	p, _, err := j.client.Project.Get(projectId)
	if err != nil {
		return nil, err
	}

	jiraVersion := findVersion(version, p.Versions)
	if jiraVersion == nil {
		return nil, fmt.Errorf("version %s not found", version)
	}

	v := mapJiraVersion(jiraVersion)
	return v, nil
}

func (j *JiraService) SearchByQuery(query string) ([]Issue, error) {
	issues, _, err := j.client.Issue.Search(query, nil)
	if err != nil {
		return nil, err
	}
	return mapJiraListToVoList(issues), nil
}

func findVersion(version string, versions []gojira.Version) *gojira.Version {
	for _, v := range versions {
		if v.Name == version {
			return &v
		}
	}
	return nil
}

func mapJiraVersion(v *gojira.Version) *Version {
	return &Version{
		Name:        v.Name,
		Released:    v.Released,
		ReleaseDate: v.ReleaseDate,
	}
}

func mapJiraToVo(issue gojira.Issue) Issue {
	var fixVersion *string
	if len(issue.Fields.FixVersions) > 0 {
		fixVersion = &issue.Fields.FixVersions[0].Name
	}

	var status *string
	if issue.Fields.Status != nil {
		status = &issue.Fields.Status.Name
	}

	squad := parseSquadField(issue.Fields)

	return Issue{
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Type:        issue.Fields.Type.Name,
		FixVersion:  fixVersion,
		Assignee:    mapAssignee(issue),
		Squad:       squad,
		Status:      status,
		Labels:      issue.Fields.Labels,
		StoryPoints: parseStoryPointsField(issue.Fields),
		IssueLinks:  mapIssueLinks(issue.Fields.IssueLinks),
	}
}

func mapIssueLinks(links []*gojira.IssueLink) []*IssueLink {
	if links == nil {
		return []*IssueLink{}
	}

	mapped := make([]*IssueLink, 0, len(links))
	for _, l := range links {
		mapped = append(mapped, mapIssueLink(l))
	}
	return mapped
}

func mapIssueLink(link *gojira.IssueLink) *IssueLink {
	if link.InwardIssue != nil {
		return &IssueLink{
			Relation:     link.Type.Inward,
			IssueKey:     link.InwardIssue.Key,
			IssueSummary: link.InwardIssue.Fields.Summary,
			Direction:    IssueLinkDirectionInward,
		}
	}
	if link.OutwardIssue != nil {
		return &IssueLink{
			Relation:     link.Type.Outward,
			IssueKey:     link.OutwardIssue.Key,
			IssueSummary: link.OutwardIssue.Fields.Summary,
			Direction:    IssueLinkDirectionOutward,
		}
	}

	return nil
}

func mapAssignee(issue gojira.Issue) *string {
	if issue.Fields.Assignee == nil {
		return nil
	}
	return &issue.Fields.Assignee.DisplayName
}

func parseSquadField(fields *gojira.IssueFields) *string {
	customSquadField, ok := fields.Unknowns["customfield_10951"]
	if !ok || customSquadField == nil {
		return nil
	}

	squadField, ok := customSquadField.(map[string]interface{})
	if !ok || squadField == nil {
		return nil
	}

	squadNameUntyped, ok := squadField["value"]
	if !ok || squadNameUntyped == nil {
		return nil
	}

	squadName, ok := squadNameUntyped.(string)
	if !ok {
		return nil
	}

	return &squadName
}

func parseStoryPointsField(fields *gojira.IssueFields) *int {
	customStoryPointField, ok := fields.Unknowns["customfield_10004"]

	if !ok || customStoryPointField == nil {
		return nil
	}

	floatStoryPointField, ok := customStoryPointField.(float64)
	if !ok {
		return nil
	}
	intStoryPointField := int(floatStoryPointField)
	return &intStoryPointField
}

func mapJiraListToVoList(issues []gojira.Issue) []Issue {
	jiraIssues := []Issue{}
	for _, issue := range issues {
		jiraIssue := mapJiraToVo(issue)
		jiraIssues = append(jiraIssues, jiraIssue)
	}
	return jiraIssues
}
