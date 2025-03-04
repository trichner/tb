package jira

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	credentials2 "github.com/trichner/tb/pkg/jira/credentials"
)

func TestNewJiraService(t *testing.T) {
	t.Skip("integration test")

	creds, err := credentials2.FindCredentials()
	assert.NoError(t, err)

	svc, err := NewJiraService(creds.Username, creds.Token)
	assert.NoError(t, err)

	issue, err := svc.GetByKey("ARC-119")
	assert.NoError(t, err)

	fmt.Printf("%+v\n", issue)
	fmt.Printf("assignee: %s\n", *issue.Assignee)
}
