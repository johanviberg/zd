package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTicketContent_SubjectAndBody(t *testing.T) {
	content := "# comment\nMy Subject\nThis is the body\nSecond line"
	subject, body, err := parseTicketContent(content)
	require.NoError(t, err)
	assert.Equal(t, "My Subject", subject)
	assert.Equal(t, "This is the body\nSecond line", body)
}

func TestParseTicketContent_SubjectOnly(t *testing.T) {
	content := "Just a subject"
	subject, body, err := parseTicketContent(content)
	require.NoError(t, err)
	assert.Equal(t, "Just a subject", subject)
	assert.Empty(t, body)
}

func TestParseTicketContent_Empty(t *testing.T) {
	content := "# all comments\n# nothing else\n"
	subject, body, err := parseTicketContent(content)
	require.NoError(t, err)
	assert.Empty(t, subject)
	assert.Empty(t, body)
}

func TestParseTicketContent_WithTemplate(t *testing.T) {
	content := ticketTemplate + "Bug Report\nSomething is broken"
	subject, body, err := parseTicketContent(content)
	require.NoError(t, err)
	assert.Equal(t, "Bug Report", subject)
	assert.Equal(t, "Something is broken", body)
}

func TestParseCommentContent(t *testing.T) {
	content := "# comment\nHello world\nLine 2"
	result := parseCommentContent(content)
	assert.Equal(t, "Hello world\nLine 2", result)
}

func TestParseCommentContent_Empty(t *testing.T) {
	content := "# only comments"
	result := parseCommentContent(content)
	assert.Empty(t, result)
}
