package editor

import "testing"

func TestParseTicketContent_SubjectAndBody(t *testing.T) {
	content := "# comment\nMy Subject\nThis is the body\nSecond line"
	subject, body, err := parseTicketContent(content)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "My Subject" {
		t.Errorf("subject = %q, want %q", subject, "My Subject")
	}
	if body != "This is the body\nSecond line" {
		t.Errorf("body = %q, want %q", body, "This is the body\nSecond line")
	}
}

func TestParseTicketContent_SubjectOnly(t *testing.T) {
	content := "Just a subject"
	subject, body, err := parseTicketContent(content)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Just a subject" {
		t.Errorf("subject = %q", subject)
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestParseTicketContent_Empty(t *testing.T) {
	content := "# all comments\n# nothing else\n"
	subject, body, err := parseTicketContent(content)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "" || body != "" {
		t.Errorf("expected empty, got subject=%q body=%q", subject, body)
	}
}

func TestParseTicketContent_WithTemplate(t *testing.T) {
	content := ticketTemplate + "Bug Report\nSomething is broken"
	subject, body, err := parseTicketContent(content)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Bug Report" {
		t.Errorf("subject = %q", subject)
	}
	if body != "Something is broken" {
		t.Errorf("body = %q", body)
	}
}

func TestParseCommentContent(t *testing.T) {
	content := "# comment\nHello world\nLine 2"
	result := parseCommentContent(content)
	if result != "Hello world\nLine 2" {
		t.Errorf("result = %q", result)
	}
}

func TestParseCommentContent_Empty(t *testing.T) {
	content := "# only comments"
	result := parseCommentContent(content)
	if result != "" {
		t.Errorf("result = %q, want empty", result)
	}
}
