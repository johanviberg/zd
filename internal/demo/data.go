package demo

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/johanviberg/zd/internal/types"
)

type Store struct {
	mu       sync.RWMutex
	Tickets  map[int64]types.Ticket
	Comments map[int64][]types.Comment
	Audits   map[int64][]types.Audit
	Users    []types.User
	nextID   int64
}

func NewStore() *Store {
	s := &Store{
		Tickets:  make(map[int64]types.Ticket),
		Comments: make(map[int64][]types.Comment),
		Audits:   make(map[int64][]types.Audit),
	}
	r := rand.New(rand.NewSource(42))
	s.generateUsers()
	s.generateTickets(r)
	return s
}

// nextID must be called while holding s.mu.Lock().
func (s *Store) nextIDLocked() int64 {
	s.nextID++
	return s.nextID
}

func (s *Store) generateUsers() {
	s.Users = []types.User{
		{ID: 1001, Name: "Sarah Chen", Email: "sarah.chen@example.com", Role: "agent", Active: true},
		{ID: 1002, Name: "Emma Johansson", Email: "emma.johansson@example.com", Role: "agent", Active: true},
		{ID: 1003, Name: "James O'Brien", Email: "james.obrien@example.com", Role: "agent", Active: true},
		{ID: 1004, Name: "Priya Patel", Email: "priya.patel@example.com", Role: "agent", Active: true},
		{ID: 2001, Name: "Marcus Rivera", Email: "marcus.rivera@customer.com", Role: "end-user", Active: true},
		{ID: 2002, Name: "Lisa Tanaka", Email: "lisa.tanaka@customer.com", Role: "end-user", Active: true},
		{ID: 2003, Name: "David Kim", Email: "david.kim@customer.com", Role: "end-user", Active: true},
		{ID: 2004, Name: "Anna Kowalski", Email: "anna.kowalski@customer.com", Role: "end-user", Active: true},
		{ID: 2005, Name: "Robert Johnson", Email: "robert.johnson@customer.com", Role: "end-user", Active: true},
		{ID: 2006, Name: "Fatima Al-Hassan", Email: "fatima.alhassan@customer.com", Role: "end-user", Active: true},
	}
}

var agentIDs = []int64{1001, 1002, 1003, 1004}
var endUserIDs = []int64{2001, 2002, 2003, 2004, 2005, 2006}

type category struct {
	name     string
	tags     []string
	subjects []string
	descs    []string
}

var categories = []category{
	{
		name: "billing",
		tags: []string{"billing", "payment", "invoice"},
		subjects: []string{
			"Incorrect charge on my account",
			"Need a refund for duplicate payment",
			"Invoice not received for last month",
			"Billing cycle change request",
			"Credit card update failed",
			"Unexpected price increase on renewal",
			"Pro-rated refund for downgrade",
			"Tax exemption certificate submission",
			"Payment method declined repeatedly",
			"Annual vs monthly billing question",
			"Coupon code not applying at checkout",
			"Disputed charge on credit card statement",
			"Requesting billing history export",
			"Auto-renewal cancellation request",
			"Currency conversion issue on invoice",
			"Need updated W-9 form",
			"Payment confirmation not received",
			"Subscription overlap after plan change",
			"Requesting custom enterprise pricing",
			"Volume discount eligibility question",
		},
		descs: []string{
			"I noticed an extra charge of $49.99 on my latest statement that I don't recognize.",
			"I was billed twice this month. Please issue a refund for the duplicate payment.",
			"I haven't received my invoice for the billing period ending last month.",
			"I'd like to switch from monthly to annual billing to save on costs.",
			"Every time I try to update my credit card, I get an error message.",
		},
	},
	{
		name: "bug",
		tags: []string{"bug", "defect", "issue"},
		subjects: []string{
			"App crashes on login after update",
			"Dashboard shows incorrect data",
			"Export to CSV produces empty file",
			"Notification emails arriving late",
			"Search returns no results for existing items",
			"Mobile app freezes on ticket list",
			"Date picker shows wrong timezone",
			"Attachment upload fails over 5MB",
			"Two-factor auth code not accepted",
			"Page layout broken on Safari",
			"API returns 500 error on bulk update",
			"Auto-save not working in editor",
			"Dark mode colors are unreadable",
			"Keyboard shortcuts stopped working",
			"Filter combination causes blank page",
			"Copy-paste strips formatting",
			"Drag and drop not working in Chrome",
			"Print preview missing header logo",
			"Sorting by date column is incorrect",
			"Memory leak when tab left open overnight",
		},
		descs: []string{
			"After the latest update, the app crashes immediately when I try to log in.",
			"The dashboard metrics don't match what I see in the detailed reports.",
			"When I export tickets to CSV, the downloaded file is completely empty.",
			"Notification emails are arriving 2-3 hours after the actual event.",
			"I'm searching for items I know exist but getting zero results returned.",
		},
	},
	{
		name: "feature_request",
		tags: []string{"feature_request", "enhancement"},
		subjects: []string{
			"Add dark mode support",
			"Request: Bulk ticket assignment",
			"Webhook support for ticket events",
			"Custom dashboard widgets",
			"Slack integration for notifications",
			"Add ticket templates feature",
			"Request: SLA reporting dashboard",
			"Multi-language support for help center",
			"API rate limit increase option",
			"Add ticket merge functionality",
			"Custom roles and permissions",
			"Request: Scheduled ticket creation",
			"Advanced search with boolean operators",
			"Mobile push notification preferences",
			"Add audit log for admin actions",
		},
		descs: []string{
			"It would be great to have a dark mode option for the interface.",
			"We need the ability to assign multiple tickets to an agent at once.",
			"We'd like webhook notifications when ticket status changes.",
			"Would love custom widgets on the dashboard to track our specific metrics.",
			"Can you add Slack integration so we get notified of new tickets?",
		},
	},
	{
		name: "account",
		tags: []string{"account", "password", "access"},
		subjects: []string{
			"Cannot reset my password",
			"Account locked after failed attempts",
			"Need to change account email address",
			"SSO login not working for our team",
			"Request to delete my account",
			"Two-factor authentication setup help",
			"Account permissions not updating",
			"Need to transfer account ownership",
			"Login page redirects in a loop",
			"Guest access not working as expected",
			"API key generation failing",
			"Session expires too quickly",
			"Account merge request for duplicates",
			"Role assignment not reflecting correctly",
			"SAML configuration assistance needed",
		},
		descs: []string{
			"I've tried resetting my password multiple times but never receive the email.",
			"My account got locked after entering the wrong password. I need it unlocked.",
			"I changed my email and need to update it on my account.",
			"Our team's SSO integration stopped working after a recent change.",
			"I'd like to request full deletion of my account and all associated data.",
		},
	},
	{
		name: "integration",
		tags: []string{"api", "integration", "webhook"},
		subjects: []string{
			"REST API authentication failing",
			"Webhook delivery not reliable",
			"JIRA integration sync issues",
			"OAuth token refresh not working",
			"API pagination returning duplicates",
			"Salesforce connector timeout errors",
			"GraphQL endpoint returning wrong schema",
			"Rate limiting too aggressive for our use",
			"Custom integration with internal tool",
			"Zapier trigger not firing on updates",
		},
		descs: []string{
			"Our API calls are getting 401 errors even though the token hasn't expired.",
			"We're missing about 20% of webhook deliveries to our endpoint.",
			"The JIRA sync stopped working and tickets aren't being created automatically.",
			"OAuth refresh tokens are being rejected even though they should be valid.",
			"When paginating through results, we see duplicate entries between pages.",
		},
	},
	{
		name: "general",
		tags: []string{"question", "how-to", "general"},
		subjects: []string{
			"How to set up automated responses",
			"Best practices for ticket categorization",
			"Question about data retention policy",
			"Help with report generation",
			"Understanding ticket priority levels",
			"How to configure business hours",
			"Question about agent collision detection",
			"Need help with macro setup",
			"How to create custom ticket fields",
			"Training request for new team members",
		},
		descs: []string{
			"Can you guide me through setting up automated responses for common questions?",
			"We're looking for best practices on how to categorize our incoming tickets.",
			"What is your data retention policy for closed tickets?",
			"I need help generating a report that shows response times by agent.",
			"Can you explain how ticket priority levels affect routing and SLAs?",
		},
	},
	{
		name: "performance",
		tags: []string{"performance", "slow", "latency"},
		subjects: []string{
			"Dashboard loading extremely slowly",
			"API response times degraded",
			"Search performance issues with large dataset",
			"Page timeout when loading ticket list",
			"Report generation taking too long",
			"Mobile app slow on older devices",
			"Bulk operations timing out",
			"Real-time updates lagging behind",
			"File upload speed very slow",
			"Autocomplete search has noticeable delay",
		},
		descs: []string{
			"The dashboard takes over 30 seconds to load, making it unusable.",
			"Our API response times have increased from 200ms to over 2 seconds.",
			"Search queries on our dataset of 500k tickets are extremely slow.",
			"Loading the ticket list page times out when we have filters applied.",
			"Report generation for the last quarter has been running for over an hour.",
		},
	},
}

// Weighted distribution: billing 20%, bug 20%, feature 15%, account 15%, integration 10%, general 10%, performance 10%
var categoryWeights = []int{20, 20, 15, 15, 10, 10, 10}

func pickCategory(r *rand.Rand) int {
	n := r.Intn(100)
	cumulative := 0
	for i, w := range categoryWeights {
		cumulative += w
		if n < cumulative {
			return i
		}
	}
	return len(categoryWeights) - 1
}

var statuses = []string{"new", "open", "pending", "hold", "solved"}

// Status distribution: 15 new, 30 open, 20 pending, 10 hold, 25 solved
var statusWeights = []int{15, 30, 20, 10, 25}

func pickStatus(r *rand.Rand) string {
	n := r.Intn(100)
	cumulative := 0
	for i, w := range statusWeights {
		cumulative += w
		if n < cumulative {
			return statuses[i]
		}
	}
	return statuses[len(statuses)-1]
}

var priorities = []string{"urgent", "high", "normal", "low"}

// Priority distribution: 5 urgent, 20 high, 50 normal, 25 low
var priorityWeights = []int{5, 20, 50, 25}

func pickPriority(r *rand.Rand) string {
	n := r.Intn(100)
	cumulative := 0
	for i, w := range priorityWeights {
		cumulative += w
		if n < cumulative {
			return priorities[i]
		}
	}
	return priorities[len(priorities)-1]
}

var commentBodies = []string{
	"Thank you for reaching out. I'm looking into this for you now.",
	"I've been able to reproduce this issue on our end. Let me escalate to the engineering team.",
	"Could you provide more details about when this started happening?",
	"I've updated my ticket with the information you requested. Please let me know if you need anything else.",
	"This has been happening intermittently for the past week. Any updates?",
	"I've applied a fix to your account. Can you try again and let me know if it works?",
	"Still experiencing the same issue after the suggested fix. Please advise.",
	"Great news — the engineering team has deployed a fix for this. Can you verify on your end?",
	"Confirmed, the issue is now resolved. Thank you for the quick turnaround!",
	"I'll need to check with our billing department. I'll get back to you within 24 hours.",
	"We've identified the root cause and a permanent fix will be included in next week's release.",
	"I've attached a screenshot showing the error I'm seeing.",
	"Thanks for your patience. We've prioritized this and it's being worked on now.",
	"Could you try clearing your cache and logging in again?",
	"I've forwarded this to our integrations team for further investigation.",
}

func (s *Store) generateTickets(r *rand.Rand) {
	now := time.Now().UTC()

	for i := int64(1); i <= 100; i++ {
		catIdx := pickCategory(r)
		cat := categories[catIdx]

		subject := cat.subjects[r.Intn(len(cat.subjects))]
		desc := cat.descs[r.Intn(len(cat.descs))]

		// Spread over last 90 days
		daysAgo := r.Intn(90)
		hoursAgo := r.Intn(24)
		createdAt := now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(hoursAgo)*time.Hour)
		// Updated 0-48 hours after creation (but not in the future)
		updatedOffset := time.Duration(r.Intn(48)) * time.Hour
		updatedAt := createdAt.Add(updatedOffset)
		if updatedAt.After(now) {
			updatedAt = now
		}

		requester := endUserIDs[r.Intn(len(endUserIDs))]
		assignee := agentIDs[r.Intn(len(agentIDs))]

		ticket := types.Ticket{
			ID:          i,
			Subject:     subject,
			Description: desc,
			Status:      pickStatus(r),
			Priority:    pickPriority(r),
			Type:        "question",
			RequesterID: requester,
			AssigneeID:  assignee,
			Tags:        cat.tags,
			CreatedAt:   createdAt.Truncate(time.Second),
			UpdatedAt:   updatedAt.Truncate(time.Second),
		}

		s.Tickets[i] = ticket

		// 1-5 comments per ticket
		numComments := 1 + r.Intn(5)
		comments := make([]types.Comment, 0, numComments)
		for j := 0; j < numComments; j++ {
			isPublic := r.Float64() > 0.3 // 70% public
			pub := isPublic
			var authorID int64
			if isPublic && r.Float64() > 0.5 {
				authorID = requester
			} else {
				authorID = agentIDs[r.Intn(len(agentIDs))]
			}

			commentTime := createdAt.Add(time.Duration(j+1) * time.Hour)
			if commentTime.After(now) {
				commentTime = now
			}

			body := commentBodies[r.Intn(len(commentBodies))]

			comment := types.Comment{
				ID:        i*100 + int64(j+1),
				Body:      body,
				Public:    &pub,
				AuthorID:  authorID,
				CreatedAt: commentTime.Truncate(time.Second),
			}

			// ~25% of comments get image attachments
			if r.Float64() < 0.25 {
				comment.Attachments = sampleAttachments(r, comment.ID)
			}

			comments = append(comments, comment)
		}
		s.Comments[i] = comments

		// Generate audits from ticket + comments
		s.Audits[i] = generateAudits(r, ticket, comments)
		s.nextID = i
	}

	// Also set nextID past the comment IDs
	s.nextID = 100
}

func (s *Store) UserByID(id int64) *types.User {
	for i := range s.Users {
		if s.Users[i].ID == id {
			return &s.Users[i]
		}
	}
	return nil
}

func (s *Store) CollectUsers(tickets []types.Ticket) []types.User {
	seen := make(map[int64]bool)
	var users []types.User
	for _, t := range tickets {
		for _, id := range []int64{t.RequesterID, t.AssigneeID} {
			if id != 0 && !seen[id] {
				seen[id] = true
				if u := s.UserByID(id); u != nil {
					users = append(users, *u)
				}
			}
		}
	}
	return users
}

func (s *Store) CollectCommentUsers(comments []types.Comment) []types.User {
	seen := make(map[int64]bool)
	var users []types.User
	for _, c := range comments {
		if c.AuthorID != 0 && !seen[c.AuthorID] {
			seen[c.AuthorID] = true
			if u := s.UserByID(c.AuthorID); u != nil {
				users = append(users, *u)
			}
		}
	}
	return users
}

func (s *Store) CollectAuditUsers(audits []types.Audit) []types.User {
	seen := make(map[int64]bool)
	var users []types.User
	for _, a := range audits {
		if a.AuthorID != 0 && !seen[a.AuthorID] {
			seen[a.AuthorID] = true
			if u := s.UserByID(a.AuthorID); u != nil {
				users = append(users, *u)
			}
		}
		for _, ev := range a.Events {
			if ev.AuthorID != 0 && !seen[ev.AuthorID] {
				seen[ev.AuthorID] = true
				if u := s.UserByID(ev.AuthorID); u != nil {
					users = append(users, *u)
				}
			}
		}
	}
	return users
}

var sampleImageFiles = []struct {
	name        string
	contentType string
	size        int64
}{
	{"screenshot.png", "image/png", 45200},
	{"error-dialog.jpg", "image/jpeg", 128000},
	{"form-fields.png", "image/png", 67500},
	{"ui-bug.gif", "image/gif", 230400},
	{"receipt.png", "image/png", 23100},
	{"console-output.png", "image/png", 89000},
}

var sampleNonImageFiles = []struct {
	name        string
	contentType string
	size        int64
}{
	{"debug-log.txt", "text/plain", 2048},
	{"report.pdf", "application/pdf", 154000},
}

func sampleAttachments(r *rand.Rand, commentID int64) []types.Attachment {
	var attachments []types.Attachment
	// 1-2 image attachments
	n := 1 + r.Intn(2)
	for j := 0; j < n; j++ {
		img := sampleImageFiles[r.Intn(len(sampleImageFiles))]
		attachments = append(attachments, types.Attachment{
			ID:          commentID*10 + int64(j+1),
			FileName:    img.name,
			ContentURL:  fmt.Sprintf("https://example.com/attachments/%d/%s", commentID, img.name),
			ContentType: img.contentType,
			Size:        img.size,
		})
	}
	// Sometimes add a non-image attachment too
	if r.Float64() < 0.3 {
		other := sampleNonImageFiles[r.Intn(len(sampleNonImageFiles))]
		attachments = append(attachments, types.Attachment{
			ID:          commentID*10 + int64(n+1),
			FileName:    other.name,
			ContentURL:  fmt.Sprintf("https://example.com/attachments/%d/%s", commentID, other.name),
			ContentType: other.contentType,
			Size:        other.size,
		})
	}
	return attachments
}

// generateAudits creates audit entries from a ticket and its comments.
func generateAudits(r *rand.Rand, ticket types.Ticket, comments []types.Comment) []types.Audit {
	var audits []types.Audit
	auditID := ticket.ID * 1000

	// First audit: ticket creation with description as comment + Create events
	auditID++
	createEvents := []types.AuditEvent{
		{
			ID:       auditID * 10,
			Type:     "Comment",
			Body:     ticket.Description,
			Public:   boolPtr(true),
			AuthorID: ticket.RequesterID,
		},
		{
			ID:        auditID*10 + 1,
			Type:      "Create",
			FieldName: "status",
			Value:     "new",
		},
	}
	if ticket.Priority != "" {
		createEvents = append(createEvents, types.AuditEvent{
			ID:        auditID*10 + 2,
			Type:      "Create",
			FieldName: "priority",
			Value:     ticket.Priority,
		})
	}
	audits = append(audits, types.Audit{
		ID:        auditID,
		TicketID:  ticket.ID,
		AuthorID:  ticket.RequesterID,
		CreatedAt: ticket.CreatedAt,
		Events:    createEvents,
	})

	// Status transition path for field-change audits
	statusPath := statusTransitionPath(ticket.Status)

	// Interleave comments with field-change audits
	changeInserted := 0
	for ci, c := range comments {
		// Insert a field-change audit before some comments
		if ci > 0 && changeInserted < len(statusPath)-1 && r.Float64() < 0.6 {
			auditID++
			changeTime := c.CreatedAt.Add(-time.Duration(r.Intn(30)+1) * time.Minute)
			agent := agentIDs[r.Intn(len(agentIDs))]

			var events []types.AuditEvent
			events = append(events, types.AuditEvent{
				ID:            auditID * 10,
				Type:          "Change",
				FieldName:     "status",
				PreviousValue: statusPath[changeInserted],
				Value:         statusPath[changeInserted+1],
			})
			changeInserted++

			// Sometimes add a priority change too
			if r.Float64() < 0.3 {
				oldPri := priorities[r.Intn(len(priorities))]
				newPri := priorities[r.Intn(len(priorities))]
				if oldPri != newPri {
					events = append(events, types.AuditEvent{
						ID:            auditID*10 + 1,
						Type:          "Change",
						FieldName:     "priority",
						PreviousValue: oldPri,
						Value:         newPri,
					})
				}
			}

			// Sometimes add an assignee change
			if r.Float64() < 0.3 {
				newAssignee := agentIDs[r.Intn(len(agentIDs))]
				events = append(events, types.AuditEvent{
					ID:            auditID*10 + 2,
					Type:          "Change",
					FieldName:     "assignee_id",
					PreviousValue: fmt.Sprintf("%d", ticket.AssigneeID),
					Value:         fmt.Sprintf("%d", newAssignee),
				})
			}

			audits = append(audits, types.Audit{
				ID:        auditID,
				TicketID:  ticket.ID,
				AuthorID:  agent,
				CreatedAt: changeTime,
				Events:    events,
			})
		}

		// Comment audit
		auditID++
		commentEvents := []types.AuditEvent{
			{
				ID:          auditID * 10,
				Type:        "Comment",
				Body:        c.Body,
				Public:      c.Public,
				AuthorID:    c.AuthorID,
				Attachments: c.Attachments,
			},
		}
		audits = append(audits, types.Audit{
			ID:        auditID,
			TicketID:  ticket.ID,
			AuthorID:  c.AuthorID,
			CreatedAt: c.CreatedAt,
			Events:    commentEvents,
		})
	}

	return audits
}

// statusTransitionPath returns a plausible status path ending at the target status.
func statusTransitionPath(target string) []string {
	switch target {
	case "new":
		return []string{"new"}
	case "open":
		return []string{"new", "open"}
	case "pending":
		return []string{"new", "open", "pending"}
	case "hold":
		return []string{"new", "open", "hold"}
	case "solved":
		return []string{"new", "open", "pending", "solved"}
	case "closed":
		return []string{"new", "open", "solved", "closed"}
	default:
		return []string{"new", "open"}
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// DemoSubdomain returns the subdomain used for demo mode URLs.
const DemoSubdomain = "demo"

// TicketURL returns a plausible ticket URL for demo mode.
func TicketURL(id int64) string {
	return fmt.Sprintf("https://%s.zendesk.com/api/v2/tickets/%d.json", DemoSubdomain, id)
}
