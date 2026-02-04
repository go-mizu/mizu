package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/types"
)

// SeedLabels creates system and custom labels.
func (s *Store) SeedLabels(ctx context.Context) error {
	systemLabels := []types.Label{
		{ID: "inbox", Name: "Inbox", Type: types.LabelTypeSystem, Visible: true, Position: 0},
		{ID: "starred", Name: "Starred", Type: types.LabelTypeSystem, Visible: true, Position: 1},
		{ID: "important", Name: "Important", Type: types.LabelTypeSystem, Visible: true, Position: 2},
		{ID: "snoozed", Name: "Snoozed", Type: types.LabelTypeSystem, Color: "#F4B400", Visible: true, Position: 3},
		{ID: "sent", Name: "Sent", Type: types.LabelTypeSystem, Visible: true, Position: 4},
		{ID: "drafts", Name: "Drafts", Type: types.LabelTypeSystem, Visible: true, Position: 5},
		{ID: "all", Name: "All Mail", Type: types.LabelTypeSystem, Visible: true, Position: 6},
		{ID: "spam", Name: "Spam", Type: types.LabelTypeSystem, Visible: true, Position: 7},
		{ID: "trash", Name: "Trash", Type: types.LabelTypeSystem, Visible: true, Position: 8},
		{ID: "scheduled", Name: "Scheduled", Type: types.LabelTypeSystem, Color: "#34A853", Visible: true, Position: 9},
	}

	for _, label := range systemLabels {
		if err := s.CreateLabel(ctx, &label); err != nil {
			continue
		}
	}

	customLabels := []types.Label{
		{ID: "work", Name: "Work", Color: "#4285F4", Type: types.LabelTypeUser, Visible: true, Position: 10},
		{ID: "personal", Name: "Personal", Color: "#34A853", Type: types.LabelTypeUser, Visible: true, Position: 11},
		{ID: "finance", Name: "Finance", Color: "#FBBC05", Type: types.LabelTypeUser, Visible: true, Position: 12},
		{ID: "travel", Name: "Travel", Color: "#EA4335", Type: types.LabelTypeUser, Visible: true, Position: 13},
	}

	for _, label := range customLabels {
		if err := s.CreateLabel(ctx, &label); err != nil {
			continue
		}
	}

	return nil
}

// SeedContacts creates sample contacts.
func (s *Store) SeedContacts(ctx context.Context) error {
	now := time.Now()
	contacts := []types.Contact{
		{ID: "contact-01", Email: "alice.chen@techcorp.io", Name: "Alice Chen", IsFrequent: true, ContactCount: 24, CreatedAt: now},
		{ID: "contact-02", Email: "bob.martinez@startup.com", Name: "Bob Martinez", IsFrequent: true, ContactCount: 18, CreatedAt: now},
		{ID: "contact-03", Email: "carol.johnson@bigbank.com", Name: "Carol Johnson", IsFrequent: true, ContactCount: 15, CreatedAt: now},
		{ID: "contact-04", Email: "david.kim@university.edu", Name: "David Kim", IsFrequent: false, ContactCount: 8, CreatedAt: now},
		{ID: "contact-05", Email: "emma.wilson@design.co", Name: "Emma Wilson", IsFrequent: true, ContactCount: 22, CreatedAt: now},
		{ID: "contact-06", Email: "frank.brown@devops.io", Name: "Frank Brown", IsFrequent: false, ContactCount: 6, CreatedAt: now},
		{ID: "contact-07", Email: "grace.lee@marketing.com", Name: "Grace Lee", IsFrequent: true, ContactCount: 12, CreatedAt: now},
		{ID: "contact-08", Email: "henry.taylor@legal.com", Name: "Henry Taylor", IsFrequent: false, ContactCount: 4, CreatedAt: now},
		{ID: "contact-09", Email: "irene.garcia@hr.company.com", Name: "Irene Garcia", IsFrequent: false, ContactCount: 7, CreatedAt: now},
		{ID: "contact-10", Email: "james.white@sales.io", Name: "James White", IsFrequent: true, ContactCount: 16, CreatedAt: now},
		{ID: "contact-11", Email: "karen.patel@product.co", Name: "Karen Patel", IsFrequent: true, ContactCount: 20, CreatedAt: now},
		{ID: "contact-12", Email: "leo.nguyen@engineering.dev", Name: "Leo Nguyen", IsFrequent: false, ContactCount: 9, CreatedAt: now},
		{ID: "contact-13", Email: "notifications@github.com", Name: "GitHub", IsFrequent: true, ContactCount: 35, CreatedAt: now},
		{ID: "contact-14", Email: "noreply@accounts.google.com", Name: "Google", IsFrequent: false, ContactCount: 5, CreatedAt: now},
		{ID: "contact-15", Email: "travel@booking.com", Name: "Booking.com", IsFrequent: false, ContactCount: 3, CreatedAt: now},
	}

	for i := range contacts {
		t := now.Add(-time.Duration(i) * 24 * time.Hour)
		contacts[i].LastContacted = &t
		if err := s.CreateContact(ctx, &contacts[i]); err != nil {
			continue
		}
	}

	return nil
}

// SeedEmails creates sample email threads with realistic content.
func (s *Store) SeedEmails(ctx context.Context) error {
	now := time.Now()

	// Helper to create a time offset from now
	ago := func(days, hours, minutes int) time.Time {
		return now.Add(-time.Duration(days)*24*time.Hour - time.Duration(hours)*time.Hour - time.Duration(minutes)*time.Minute)
	}

	// Thread 1: Q4 Planning Discussion (3 emails in thread)
	thread1 := "thread-001"
	emails := []types.Email{
		{
			ID: "email-001", ThreadID: thread1,
			MessageID:   "<msg-001@techcorp.io>",
			FromAddress: "alice.chen@techcorp.io", FromName: "Alice Chen",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			CCAddresses: []types.Recipient{{Name: "Karen Patel", Address: "karen.patel@product.co"}},
			Subject:     "Q4 Planning - Engineering Priorities",
			BodyText:    "Hi team,\n\nI wanted to kick off our Q4 planning discussion. We need to align on engineering priorities for the next quarter. Key areas I'd like to discuss:\n\n1. Platform scalability improvements\n2. Developer experience tooling\n3. Security audit follow-ups\n4. Technical debt reduction\n\nCan we schedule a meeting for next Tuesday to go over these? I've attached the preliminary roadmap for your review.\n\nBest,\nAlice",
			BodyHTML:    "<p>Hi team,</p><p>I wanted to kick off our Q4 planning discussion. We need to align on engineering priorities for the next quarter. Key areas I'd like to discuss:</p><ol><li>Platform scalability improvements</li><li>Developer experience tooling</li><li>Security audit follow-ups</li><li>Technical debt reduction</li></ol><p>Can we schedule a meeting for next Tuesday to go over these? I've attached the preliminary roadmap for your review.</p><p>Best,<br>Alice</p>",
			Snippet:     "Hi team, I wanted to kick off our Q4 planning discussion. We need to align on engineering priorities...",
			IsRead:      true, IsImportant: true,
			Labels:     []string{"inbox", "all", "work"},
			ReceivedAt: ago(2, 10, 0), CreatedAt: ago(2, 10, 0), UpdatedAt: ago(2, 10, 0),
		},
		{
			ID: "email-002", ThreadID: thread1,
			MessageID:   "<msg-002@example.com>",
			InReplyTo:   "<msg-001@techcorp.io>",
			References:  []string{"<msg-001@techcorp.io>"},
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "Alice Chen", Address: "alice.chen@techcorp.io"}},
			CCAddresses: []types.Recipient{{Name: "Karen Patel", Address: "karen.patel@product.co"}},
			Subject:     "Re: Q4 Planning - Engineering Priorities",
			BodyText:    "Hi Alice,\n\nTuesday works for me. I'd also like to add API versioning to the list. We've been getting more requests from enterprise clients about backward compatibility.\n\nI'll review the roadmap before the meeting.\n\nThanks,\nMe",
			BodyHTML:    "<p>Hi Alice,</p><p>Tuesday works for me. I'd also like to add API versioning to the list. We've been getting more requests from enterprise clients about backward compatibility.</p><p>I'll review the roadmap before the meeting.</p><p>Thanks,<br>Me</p>",
			Snippet:     "Hi Alice, Tuesday works for me. I'd also like to add API versioning to the list...",
			IsRead:      true, IsSent: true,
			Labels:     []string{"sent", "all"},
			SentAt:     timePtr(ago(2, 8, 30)),
			ReceivedAt: ago(2, 8, 30), CreatedAt: ago(2, 8, 30), UpdatedAt: ago(2, 8, 30),
		},
		{
			ID: "email-003", ThreadID: thread1,
			MessageID:   "<msg-003@product.co>",
			InReplyTo:   "<msg-002@example.com>",
			References:  []string{"<msg-001@techcorp.io>", "<msg-002@example.com>"},
			FromAddress: "karen.patel@product.co", FromName: "Karen Patel",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}, {Name: "Alice Chen", Address: "alice.chen@techcorp.io"}},
			Subject:     "Re: Q4 Planning - Engineering Priorities",
			BodyText:    "Great points! I've also compiled user feedback from the last quarter. The top requests are:\n\n- Better search functionality\n- Mobile responsive improvements\n- Faster page load times\n\nI'll bring the data to the meeting. See you Tuesday!\n\nKaren",
			BodyHTML:    "<p>Great points! I've also compiled user feedback from the last quarter. The top requests are:</p><ul><li>Better search functionality</li><li>Mobile responsive improvements</li><li>Faster page load times</li></ul><p>I'll bring the data to the meeting. See you Tuesday!</p><p>Karen</p>",
			Snippet:     "Great points! I've also compiled user feedback from the last quarter. The top requests are...",
			IsImportant: true,
			Labels:      []string{"inbox", "all", "work"},
			ReceivedAt:  ago(2, 6, 15), CreatedAt: ago(2, 6, 15), UpdatedAt: ago(2, 6, 15),
		},
	}

	// Thread 2: Database Migration (2 emails)
	thread2 := "thread-002"
	emails = append(emails,
		types.Email{
			ID: "email-004", ThreadID: thread2,
			MessageID:   "<msg-004@devops.io>",
			FromAddress: "frank.brown@devops.io", FromName: "Frank Brown",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Database Migration Plan - Production",
			BodyText:    "Hey,\n\nI've prepared the database migration plan for the production environment. The migration window is scheduled for this Saturday at 2 AM UTC.\n\nKey steps:\n1. Enable read-only mode\n2. Run schema migrations\n3. Verify data integrity\n4. Switch traffic to new cluster\n5. Monitor for 2 hours\n\nEstimated downtime: 30 minutes max.\n\nPlease review and approve the runbook. I need sign-off by Thursday.\n\nFrank",
			BodyHTML:    "<p>Hey,</p><p>I've prepared the database migration plan for the production environment. The migration window is scheduled for this Saturday at 2 AM UTC.</p><p>Key steps:</p><ol><li>Enable read-only mode</li><li>Run schema migrations</li><li>Verify data integrity</li><li>Switch traffic to new cluster</li><li>Monitor for 2 hours</li></ol><p>Estimated downtime: 30 minutes max.</p><p>Please review and approve the runbook. I need sign-off by Thursday.</p><p>Frank</p>",
			Snippet:     "Hey, I've prepared the database migration plan for the production environment...",
			IsImportant: true, IsStarred: true,
			Labels:     []string{"inbox", "all", "work"},
			ReceivedAt: ago(1, 14, 0), CreatedAt: ago(1, 14, 0), UpdatedAt: ago(1, 14, 0),
		},
		types.Email{
			ID: "email-005", ThreadID: thread2,
			MessageID:   "<msg-005@engineering.dev>",
			InReplyTo:   "<msg-004@devops.io>",
			References:  []string{"<msg-004@devops.io>"},
			FromAddress: "leo.nguyen@engineering.dev", FromName: "Leo Nguyen",
			ToAddresses: []types.Recipient{{Name: "Frank Brown", Address: "frank.brown@devops.io"}, {Name: "Me", Address: "me@example.com"}},
			Subject:     "Re: Database Migration Plan - Production",
			BodyText:    "Frank,\n\nI've reviewed the runbook. A few suggestions:\n\n- Add a rollback procedure for step 4\n- Include monitoring dashboard links\n- Set up PagerDuty alerts for the migration window\n\nOtherwise, the plan looks solid. I'll be available during the migration window as backup.\n\nLeo",
			BodyHTML:    "<p>Frank,</p><p>I've reviewed the runbook. A few suggestions:</p><ul><li>Add a rollback procedure for step 4</li><li>Include monitoring dashboard links</li><li>Set up PagerDuty alerts for the migration window</li></ul><p>Otherwise, the plan looks solid. I'll be available during the migration window as backup.</p><p>Leo</p>",
			Snippet:     "Frank, I've reviewed the runbook. A few suggestions: Add a rollback procedure for step 4...",
			IsStarred:   true,
			Labels:      []string{"inbox", "all", "work"},
			ReceivedAt:  ago(1, 10, 45), CreatedAt: ago(1, 10, 45), UpdatedAt: ago(1, 10, 45),
		},
	)

	// Thread 3: Design Review (2 emails)
	thread3 := "thread-003"
	emails = append(emails,
		types.Email{
			ID: "email-006", ThreadID: thread3,
			MessageID:   "<msg-006@design.co>",
			FromAddress: "emma.wilson@design.co", FromName: "Emma Wilson",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Design Review: New Dashboard Mockups",
			BodyText:    "Hi!\n\nI've finished the new dashboard mockups. Here are the key changes:\n\n- Simplified navigation sidebar\n- New data visualization widgets\n- Dark mode support\n- Improved mobile layout\n\nThe Figma link is in the attached doc. I'd love to get your feedback before we start implementation.\n\nLet me know your thoughts!\n\nEmma",
			BodyHTML:    "<p>Hi!</p><p>I've finished the new dashboard mockups. Here are the key changes:</p><ul><li>Simplified navigation sidebar</li><li>New data visualization widgets</li><li>Dark mode support</li><li>Improved mobile layout</li></ul><p>The Figma link is in the attached doc. I'd love to get your feedback before we start implementation.</p><p>Let me know your thoughts!</p><p>Emma</p>",
			Snippet:     "Hi! I've finished the new dashboard mockups. Here are the key changes: Simplified navigation...",
			HasAttachments: true,
			Labels:         []string{"inbox", "all", "work"},
			ReceivedAt:     ago(3, 9, 0), CreatedAt: ago(3, 9, 0), UpdatedAt: ago(3, 9, 0),
		},
		types.Email{
			ID: "email-007", ThreadID: thread3,
			MessageID:   "<msg-007@example.com>",
			InReplyTo:   "<msg-006@design.co>",
			References:  []string{"<msg-006@design.co>"},
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "Emma Wilson", Address: "emma.wilson@design.co"}},
			Subject:     "Re: Design Review: New Dashboard Mockups",
			BodyText:    "Emma, these look fantastic!\n\nA few quick notes:\n\n1. Love the dark mode - can we make it the default?\n2. The sidebar collapse animation is smooth\n3. Consider adding keyboard shortcuts for power users\n4. The mobile layout could use larger touch targets\n\nOverall, great work. Let's sync on Thursday to finalize.\n\nCheers!",
			BodyHTML:    "<p>Emma, these look fantastic!</p><p>A few quick notes:</p><ol><li>Love the dark mode - can we make it the default?</li><li>The sidebar collapse animation is smooth</li><li>Consider adding keyboard shortcuts for power users</li><li>The mobile layout could use larger touch targets</li></ol><p>Overall, great work. Let's sync on Thursday to finalize.</p><p>Cheers!</p>",
			Snippet:     "Emma, these look fantastic! A few quick notes: Love the dark mode - can we make it the default?...",
			IsRead:      true, IsSent: true,
			Labels:     []string{"sent", "all"},
			SentAt:     timePtr(ago(3, 7, 0)),
			ReceivedAt: ago(3, 7, 0), CreatedAt: ago(3, 7, 0), UpdatedAt: ago(3, 7, 0),
		},
	)

	// Thread 4: Financial Report (single email)
	thread4 := "thread-004"
	emails = append(emails,
		types.Email{
			ID: "email-008", ThreadID: thread4,
			MessageID:   "<msg-008@bigbank.com>",
			FromAddress: "carol.johnson@bigbank.com", FromName: "Carol Johnson",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Monthly Financial Summary - January 2026",
			BodyText:    "Dear Client,\n\nPlease find attached your monthly financial summary for January 2026.\n\nHighlights:\n- Portfolio value: $142,350.00 (+3.2%)\n- Dividends received: $425.00\n- Pending transactions: 2\n\nYour next statement will be available on March 1st.\n\nIf you have any questions about your account, please don't hesitate to reach out.\n\nBest regards,\nCarol Johnson\nSenior Financial Advisor\nBigBank Wealth Management",
			BodyHTML:    "<p>Dear Client,</p><p>Please find attached your monthly financial summary for January 2026.</p><p><strong>Highlights:</strong></p><ul><li>Portfolio value: $142,350.00 (+3.2%)</li><li>Dividends received: $425.00</li><li>Pending transactions: 2</li></ul><p>Your next statement will be available on March 1st.</p><p>If you have any questions about your account, please don't hesitate to reach out.</p><p>Best regards,<br>Carol Johnson<br>Senior Financial Advisor<br>BigBank Wealth Management</p>",
			Snippet:     "Dear Client, Please find attached your monthly financial summary for January 2026...",
			IsRead:      true, HasAttachments: true,
			Labels:     []string{"inbox", "all", "finance"},
			ReceivedAt: ago(5, 8, 0), CreatedAt: ago(5, 8, 0), UpdatedAt: ago(5, 8, 0),
		},
	)

	// Thread 5: GitHub Notification (single email)
	thread5 := "thread-005"
	emails = append(emails,
		types.Email{
			ID: "email-009", ThreadID: thread5,
			MessageID:   "<msg-009@github.com>",
			FromAddress: "notifications@github.com", FromName: "GitHub",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "[go-mizu/mizu] Pull Request #247: Add WebSocket support",
			BodyText:    "alice-chen opened a pull request in go-mizu/mizu\n\n#247 Add WebSocket support\n\nThis PR adds WebSocket support to the Mizu framework:\n\n- New WebSocket upgrade handler\n- Connection pool management\n- Automatic ping/pong heartbeat\n- Message broadcasting\n\nCloses #198\n\n---\nReply to this email directly or view it on GitHub:\nhttps://github.com/go-mizu/mizu/pull/247",
			BodyHTML:    "<p><strong>alice-chen</strong> opened a pull request in <a href='https://github.com/go-mizu/mizu'>go-mizu/mizu</a></p><h3>#247 Add WebSocket support</h3><p>This PR adds WebSocket support to the Mizu framework:</p><ul><li>New WebSocket upgrade handler</li><li>Connection pool management</li><li>Automatic ping/pong heartbeat</li><li>Message broadcasting</li></ul><p>Closes #198</p>",
			Snippet:     "alice-chen opened a pull request in go-mizu/mizu #247 Add WebSocket support...",
			Labels:      []string{"inbox", "all"},
			ReceivedAt:  ago(0, 6, 30), CreatedAt: ago(0, 6, 30), UpdatedAt: ago(0, 6, 30),
		},
	)

	// Thread 6: Marketing Campaign (2 emails)
	thread6 := "thread-006"
	emails = append(emails,
		types.Email{
			ID: "email-010", ThreadID: thread6,
			MessageID:   "<msg-010@marketing.com>",
			FromAddress: "grace.lee@marketing.com", FromName: "Grace Lee",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}, {Name: "James White", Address: "james.white@sales.io"}},
			Subject:     "Product Launch Campaign - Content Review",
			BodyText:    "Hi team,\n\nThe product launch campaign materials are ready for review:\n\n1. Blog post draft (2,000 words)\n2. Social media content calendar (4 weeks)\n3. Email newsletter sequence (5 emails)\n4. Press release\n\nLaunch date: February 15th\n\nPlease review by end of this week so we can finalize everything. The blog post especially needs technical accuracy review.\n\nThanks!\nGrace",
			BodyHTML:    "<p>Hi team,</p><p>The product launch campaign materials are ready for review:</p><ol><li>Blog post draft (2,000 words)</li><li>Social media content calendar (4 weeks)</li><li>Email newsletter sequence (5 emails)</li><li>Press release</li></ol><p><strong>Launch date: February 15th</strong></p><p>Please review by end of this week so we can finalize everything. The blog post especially needs technical accuracy review.</p><p>Thanks!<br>Grace</p>",
			Snippet:     "Hi team, The product launch campaign materials are ready for review: Blog post draft...",
			HasAttachments: true,
			Labels:         []string{"inbox", "all", "work"},
			ReceivedAt:     ago(4, 11, 0), CreatedAt: ago(4, 11, 0), UpdatedAt: ago(4, 11, 0),
		},
		types.Email{
			ID: "email-011", ThreadID: thread6,
			MessageID:   "<msg-011@sales.io>",
			InReplyTo:   "<msg-010@marketing.com>",
			References:  []string{"<msg-010@marketing.com>"},
			FromAddress: "james.white@sales.io", FromName: "James White",
			ToAddresses: []types.Recipient{{Name: "Grace Lee", Address: "grace.lee@marketing.com"}, {Name: "Me", Address: "me@example.com"}},
			Subject:     "Re: Product Launch Campaign - Content Review",
			BodyText:    "Grace,\n\nI've reviewed the materials. The press release is solid. A couple of notes:\n\n- Can we add customer testimonials to the blog post?\n- The email sequence needs pricing info in email #3\n- Social calendar looks great, but let's add LinkedIn posts\n\nI'll have the sales deck aligned with these by Wednesday.\n\nJames",
			BodyHTML:    "<p>Grace,</p><p>I've reviewed the materials. The press release is solid. A couple of notes:</p><ul><li>Can we add customer testimonials to the blog post?</li><li>The email sequence needs pricing info in email #3</li><li>Social calendar looks great, but let's add LinkedIn posts</li></ul><p>I'll have the sales deck aligned with these by Wednesday.</p><p>James</p>",
			Snippet:     "Grace, I've reviewed the materials. The press release is solid. A couple of notes...",
			IsRead:      true,
			Labels:      []string{"inbox", "all", "work"},
			ReceivedAt:  ago(4, 5, 20), CreatedAt: ago(4, 5, 20), UpdatedAt: ago(4, 5, 20),
		},
	)

	// Thread 7: Travel booking (single email)
	thread7 := "thread-007"
	emails = append(emails,
		types.Email{
			ID: "email-012", ThreadID: thread7,
			MessageID:   "<msg-012@booking.com>",
			FromAddress: "travel@booking.com", FromName: "Booking.com",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Booking Confirmation - Hotel Yamato, Tokyo",
			BodyText:    "Your booking is confirmed!\n\nBooking Reference: BK-294857\n\nHotel Yamato\nShinjuku, Tokyo, Japan\n\nCheck-in: March 10, 2026\nCheck-out: March 15, 2026\nRoom: Deluxe Double, City View\nGuests: 2\n\nTotal: $1,245.00\n\nFree cancellation until March 3, 2026.\n\nHave a wonderful trip!\nBooking.com Team",
			BodyHTML:    "<h2>Your booking is confirmed!</h2><p><strong>Booking Reference:</strong> BK-294857</p><h3>Hotel Yamato</h3><p>Shinjuku, Tokyo, Japan</p><table><tr><td>Check-in</td><td>March 10, 2026</td></tr><tr><td>Check-out</td><td>March 15, 2026</td></tr><tr><td>Room</td><td>Deluxe Double, City View</td></tr><tr><td>Guests</td><td>2</td></tr><tr><td><strong>Total</strong></td><td><strong>$1,245.00</strong></td></tr></table><p>Free cancellation until March 3, 2026.</p><p>Have a wonderful trip!<br>Booking.com Team</p>",
			Snippet:     "Your booking is confirmed! Booking Reference: BK-294857, Hotel Yamato, Shinjuku, Tokyo...",
			IsRead:      true, IsStarred: true,
			Labels:     []string{"inbox", "all", "travel"},
			ReceivedAt: ago(7, 15, 0), CreatedAt: ago(7, 15, 0), UpdatedAt: ago(7, 15, 0),
		},
	)

	// Thread 8: Interview Follow-up (2 emails)
	thread8 := "thread-008"
	emails = append(emails,
		types.Email{
			ID: "email-013", ThreadID: thread8,
			MessageID:   "<msg-013@hr.company.com>",
			FromAddress: "irene.garcia@hr.company.com", FromName: "Irene Garcia",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Interview Feedback - Senior Engineer Candidate",
			BodyText:    "Hi,\n\nThank you for conducting the technical interview with Sarah Mitchell yesterday. Could you please provide your feedback by end of day tomorrow?\n\nHere's the scorecard template:\n- Technical skills (1-5):\n- Problem solving (1-5):\n- Communication (1-5):\n- Culture fit (1-5):\n- Overall recommendation: Hire / No Hire / Need more data\n\nWe're trying to make a decision by Friday.\n\nThanks,\nIrene\nTalent Acquisition",
			BodyHTML:    "<p>Hi,</p><p>Thank you for conducting the technical interview with Sarah Mitchell yesterday. Could you please provide your feedback by end of day tomorrow?</p><p>Here's the scorecard template:</p><ul><li>Technical skills (1-5):</li><li>Problem solving (1-5):</li><li>Communication (1-5):</li><li>Culture fit (1-5):</li><li>Overall recommendation: Hire / No Hire / Need more data</li></ul><p>We're trying to make a decision by Friday.</p><p>Thanks,<br>Irene<br>Talent Acquisition</p>",
			Snippet:     "Hi, Thank you for conducting the technical interview with Sarah Mitchell yesterday...",
			IsImportant: true,
			Labels:      []string{"inbox", "all"},
			ReceivedAt:  ago(1, 3, 0), CreatedAt: ago(1, 3, 0), UpdatedAt: ago(1, 3, 0),
		},
		types.Email{
			ID: "email-014", ThreadID: thread8,
			MessageID:   "<msg-014@example.com>",
			InReplyTo:   "<msg-013@hr.company.com>",
			References:  []string{"<msg-013@hr.company.com>"},
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "Irene Garcia", Address: "irene.garcia@hr.company.com"}},
			Subject:     "Re: Interview Feedback - Senior Engineer Candidate",
			BodyText:    "Hi Irene,\n\nHere's my feedback on Sarah Mitchell:\n\n- Technical skills: 5/5 - Strong Go and distributed systems knowledge\n- Problem solving: 4/5 - Excellent algorithmic thinking, took a moment on edge cases\n- Communication: 5/5 - Clear, structured explanations\n- Culture fit: 4/5 - Collaborative mindset, good questions about team dynamics\n- Overall recommendation: Hire\n\nShe's one of the strongest candidates I've interviewed this year. Strong recommend.\n\nBest,\nMe",
			BodyHTML:    "<p>Hi Irene,</p><p>Here's my feedback on Sarah Mitchell:</p><ul><li>Technical skills: 5/5 - Strong Go and distributed systems knowledge</li><li>Problem solving: 4/5 - Excellent algorithmic thinking, took a moment on edge cases</li><li>Communication: 5/5 - Clear, structured explanations</li><li>Culture fit: 4/5 - Collaborative mindset, good questions about team dynamics</li><li>Overall recommendation: <strong>Hire</strong></li></ul><p>She's one of the strongest candidates I've interviewed this year. Strong recommend.</p><p>Best,<br>Me</p>",
			Snippet:     "Hi Irene, Here's my feedback on Sarah Mitchell: Technical skills: 5/5...",
			IsRead:      true, IsSent: true,
			Labels:     []string{"sent", "all"},
			SentAt:     timePtr(ago(0, 22, 0)),
			ReceivedAt: ago(0, 22, 0), CreatedAt: ago(0, 22, 0), UpdatedAt: ago(0, 22, 0),
		},
	)

	// Thread 9: Standup Update (single email)
	thread9 := "thread-009"
	emails = append(emails,
		types.Email{
			ID: "email-015", ThreadID: thread9,
			MessageID:   "<msg-015@startup.com>",
			FromAddress: "bob.martinez@startup.com", FromName: "Bob Martinez",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Quick sync on API rate limiting",
			BodyText:    "Hey,\n\nJust wanted to loop you in - I'm implementing the API rate limiting feature we discussed. I'm going with a token bucket algorithm with these defaults:\n\n- 100 requests per minute for free tier\n- 1000 requests per minute for pro tier\n- 10000 requests per minute for enterprise\n\nI'm using Redis for the distributed counter. Should have a PR up by tomorrow.\n\nDo you have any preferences on the rate limit response headers? I was thinking:\n- X-RateLimit-Limit\n- X-RateLimit-Remaining\n- X-RateLimit-Reset\n\nLet me know if you want to chat about this.\n\nBob",
			BodyHTML:    "<p>Hey,</p><p>Just wanted to loop you in - I'm implementing the API rate limiting feature we discussed. I'm going with a token bucket algorithm with these defaults:</p><ul><li>100 requests per minute for free tier</li><li>1000 requests per minute for pro tier</li><li>10000 requests per minute for enterprise</li></ul><p>I'm using Redis for the distributed counter. Should have a PR up by tomorrow.</p><p>Do you have any preferences on the rate limit response headers? I was thinking:</p><ul><li>X-RateLimit-Limit</li><li>X-RateLimit-Remaining</li><li>X-RateLimit-Reset</li></ul><p>Let me know if you want to chat about this.</p><p>Bob</p>",
			Snippet:     "Hey, Just wanted to loop you in - I'm implementing the API rate limiting feature we discussed...",
			Labels:      []string{"inbox", "all", "work"},
			ReceivedAt:  ago(0, 4, 15), CreatedAt: ago(0, 4, 15), UpdatedAt: ago(0, 4, 15),
		},
	)

	// Thread 10: Google Account Security (single email)
	thread10 := "thread-010"
	emails = append(emails,
		types.Email{
			ID: "email-016", ThreadID: thread10,
			MessageID:   "<msg-016@accounts.google.com>",
			FromAddress: "noreply@accounts.google.com", FromName: "Google",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Security alert: New sign-in from MacBook Pro",
			BodyText:    "New sign-in to your Google Account\n\nWe noticed a new sign-in to your Google Account on a MacBook Pro.\n\nDevice: MacBook Pro\nLocation: San Francisco, CA, USA\nTime: January 28, 2026, 3:42 PM PST\nIP Address: 198.51.100.42\n\nIf this was you, you can ignore this email.\nIf this wasn't you, your account might be compromised. Please review your account security settings.\n\nBest,\nThe Google Accounts team",
			BodyHTML:    "<h3>New sign-in to your Google Account</h3><p>We noticed a new sign-in to your Google Account on a MacBook Pro.</p><table><tr><td>Device</td><td>MacBook Pro</td></tr><tr><td>Location</td><td>San Francisco, CA, USA</td></tr><tr><td>Time</td><td>January 28, 2026, 3:42 PM PST</td></tr><tr><td>IP Address</td><td>198.51.100.42</td></tr></table><p>If this was you, you can ignore this email.<br>If this wasn't you, your account might be compromised.</p>",
			Snippet:     "New sign-in to your Google Account. We noticed a new sign-in on a MacBook Pro...",
			IsRead:      true,
			Labels:      []string{"inbox", "all"},
			ReceivedAt:  ago(6, 12, 0), CreatedAt: ago(6, 12, 0), UpdatedAt: ago(6, 12, 0),
		},
	)

	// Thread 11: University Newsletter (single email)
	thread11 := "thread-011"
	emails = append(emails,
		types.Email{
			ID: "email-017", ThreadID: thread11,
			MessageID:   "<msg-017@university.edu>",
			FromAddress: "david.kim@university.edu", FromName: "David Kim",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Guest Lecture Invitation - Distributed Systems",
			BodyText:    "Hi,\n\nI'm organizing a guest lecture series on distributed systems at the university this spring. Given your experience with microservices architecture, I'd love to invite you as a speaker.\n\nProposed topic: \"Building Resilient Distributed Systems in Go\"\nDate: April 12, 2026\nTime: 2:00 PM - 3:30 PM\nFormat: 45 min talk + 45 min Q&A\nAudience: ~60 graduate students\n\nWe can offer a small honorarium and cover any travel expenses.\n\nWould you be interested? Happy to discuss further.\n\nBest,\nDr. David Kim\nAssociate Professor, Computer Science",
			BodyHTML:    "<p>Hi,</p><p>I'm organizing a guest lecture series on distributed systems at the university this spring. Given your experience with microservices architecture, I'd love to invite you as a speaker.</p><p><strong>Proposed topic:</strong> \"Building Resilient Distributed Systems in Go\"<br><strong>Date:</strong> April 12, 2026<br><strong>Time:</strong> 2:00 PM - 3:30 PM<br><strong>Format:</strong> 45 min talk + 45 min Q&A<br><strong>Audience:</strong> ~60 graduate students</p><p>We can offer a small honorarium and cover any travel expenses.</p><p>Would you be interested? Happy to discuss further.</p><p>Best,<br>Dr. David Kim<br>Associate Professor, Computer Science</p>",
			Snippet:     "Hi, I'm organizing a guest lecture series on distributed systems at the university...",
			IsStarred:   true,
			Labels:      []string{"inbox", "all", "personal"},
			ReceivedAt:  ago(8, 9, 30), CreatedAt: ago(8, 9, 30), UpdatedAt: ago(8, 9, 30),
		},
	)

	// Thread 12: Legal Document Review (single email)
	thread12 := "thread-012"
	emails = append(emails,
		types.Email{
			ID: "email-018", ThreadID: thread12,
			MessageID:   "<msg-018@legal.com>",
			FromAddress: "henry.taylor@legal.com", FromName: "Henry Taylor",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Contract Review - Vendor Agreement Draft",
			BodyText:    "Hi,\n\nPlease find attached the vendor agreement draft for your review. Key terms to note:\n\n1. Service Level Agreement: 99.9% uptime guarantee\n2. Data Processing Agreement: GDPR compliant\n3. Liability cap: $500,000\n4. Termination clause: 30 days notice\n5. Auto-renewal: Annual, with 60-day opt-out window\n\nI need your comments by next Wednesday so we can negotiate before the quarter ends.\n\nRegards,\nHenry Taylor\nGeneral Counsel",
			BodyHTML:    "<p>Hi,</p><p>Please find attached the vendor agreement draft for your review. Key terms to note:</p><ol><li>Service Level Agreement: 99.9% uptime guarantee</li><li>Data Processing Agreement: GDPR compliant</li><li>Liability cap: $500,000</li><li>Termination clause: 30 days notice</li><li>Auto-renewal: Annual, with 60-day opt-out window</li></ol><p>I need your comments by next Wednesday so we can negotiate before the quarter ends.</p><p>Regards,<br>Henry Taylor<br>General Counsel</p>",
			Snippet:     "Hi, Please find attached the vendor agreement draft for your review. Key terms to note...",
			HasAttachments: true, IsImportant: true,
			Labels:         []string{"inbox", "all"},
			ReceivedAt:     ago(0, 8, 0), CreatedAt: ago(0, 8, 0), UpdatedAt: ago(0, 8, 0),
		},
	)

	// Thread 13: Draft email
	thread13 := "thread-013"
	emails = append(emails,
		types.Email{
			ID: "email-019", ThreadID: thread13,
			MessageID:   "<msg-019@example.com>",
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "Alice Chen", Address: "alice.chen@techcorp.io"}},
			Subject:     "Proposal: Microservices Migration Strategy",
			BodyText:    "Hi Alice,\n\nI've been thinking about our monolith-to-microservices migration strategy. Here's my initial proposal:\n\nPhase 1 (Q1): Extract authentication service\nPhase 2 (Q2): Extract notification service\nPhase 3 (Q3): Extract billing service\n\nEach phase would include:\n- Service extraction\n- API contract definition\n- Integration testing\n- Gradual traffic migration\n\n[Draft - still working on cost estimates]",
			BodyHTML:    "<p>Hi Alice,</p><p>I've been thinking about our monolith-to-microservices migration strategy. Here's my initial proposal:</p><p><strong>Phase 1 (Q1):</strong> Extract authentication service<br><strong>Phase 2 (Q2):</strong> Extract notification service<br><strong>Phase 3 (Q3):</strong> Extract billing service</p><p>Each phase would include:</p><ul><li>Service extraction</li><li>API contract definition</li><li>Integration testing</li><li>Gradual traffic migration</li></ul><p><em>[Draft - still working on cost estimates]</em></p>",
			Snippet:     "Hi Alice, I've been thinking about our monolith-to-microservices migration strategy...",
			IsRead:      true, IsDraft: true,
			Labels:     []string{"drafts", "all"},
			ReceivedAt: ago(0, 2, 0), CreatedAt: ago(0, 2, 0), UpdatedAt: ago(0, 2, 0),
		},
	)

	// Thread 14: GitHub PR merged notification
	thread14 := "thread-014"
	emails = append(emails,
		types.Email{
			ID: "email-020", ThreadID: thread14,
			MessageID:   "<msg-020@github.com>",
			FromAddress: "notifications@github.com", FromName: "GitHub",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "[go-mizu/mizu] PR #243 merged: Fix graceful shutdown race condition",
			BodyText:    "Merged #243 into main.\n\nFix graceful shutdown race condition\n\nThis PR fixes a race condition in the graceful shutdown handler where in-flight requests could be dropped if the shutdown signal arrives during request processing.\n\nChanges:\n- Add WaitGroup for tracking in-flight requests\n- Defer shutdown until all requests complete\n- Add 30-second timeout for shutdown\n\nTests: All passing\nReviewed by: alice-chen",
			BodyHTML:    "<p>Merged <strong>#243</strong> into main.</p><h3>Fix graceful shutdown race condition</h3><p>This PR fixes a race condition in the graceful shutdown handler where in-flight requests could be dropped if the shutdown signal arrives during request processing.</p><p>Changes:</p><ul><li>Add WaitGroup for tracking in-flight requests</li><li>Defer shutdown until all requests complete</li><li>Add 30-second timeout for shutdown</li></ul><p>Tests: All passing<br>Reviewed by: alice-chen</p>",
			Snippet:     "Merged #243 into main. Fix graceful shutdown race condition...",
			IsRead:      true,
			Labels:      []string{"inbox", "all"},
			ReceivedAt:  ago(1, 2, 0), CreatedAt: ago(1, 2, 0), UpdatedAt: ago(1, 2, 0),
		},
	)

	// Thread 15: Conference Speaking (3 emails)
	thread15 := "thread-015"
	emails = append(emails,
		types.Email{
			ID: "email-021", ThreadID: thread15,
			MessageID:   "<msg-021@startup.com>",
			FromAddress: "bob.martinez@startup.com", FromName: "Bob Martinez",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "GopherCon 2026 - Talk Submission Deadline",
			BodyText:    "Hey!\n\nJust a reminder that the GopherCon 2026 CFP closes on February 28th. Are you still planning to submit? I'm submitting a talk on \"Go Generics in Production: Lessons Learned\".\n\nMaybe we could do a joint workshop on building web frameworks? What do you think?\n\nBob",
			BodyHTML:    "<p>Hey!</p><p>Just a reminder that the GopherCon 2026 CFP closes on February 28th. Are you still planning to submit? I'm submitting a talk on \"Go Generics in Production: Lessons Learned\".</p><p>Maybe we could do a joint workshop on building web frameworks? What do you think?</p><p>Bob</p>",
			Snippet:     "Hey! Just a reminder that the GopherCon 2026 CFP closes on February 28th...",
			Labels:      []string{"inbox", "all", "personal"},
			ReceivedAt:  ago(10, 11, 0), CreatedAt: ago(10, 11, 0), UpdatedAt: ago(10, 11, 0),
		},
		types.Email{
			ID: "email-022", ThreadID: thread15,
			MessageID:   "<msg-022@example.com>",
			InReplyTo:   "<msg-021@startup.com>",
			References:  []string{"<msg-021@startup.com>"},
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "Bob Martinez", Address: "bob.martinez@startup.com"}},
			Subject:     "Re: GopherCon 2026 - Talk Submission Deadline",
			BodyText:    "Bob,\n\nA joint workshop sounds awesome! I was thinking of submitting \"Building HTTP Frameworks from Scratch\" but a workshop would be even better.\n\nHow about: \"Hands-on Workshop: Build Your Own Go Web Framework in 90 Minutes\"\n\nWe could cover:\n- ServeMux patterns\n- Middleware chains\n- Context management\n- Error handling patterns\n\nLet's sync next week to outline the content?\n\nCheers!",
			BodyHTML:    "<p>Bob,</p><p>A joint workshop sounds awesome! I was thinking of submitting \"Building HTTP Frameworks from Scratch\" but a workshop would be even better.</p><p>How about: <strong>\"Hands-on Workshop: Build Your Own Go Web Framework in 90 Minutes\"</strong></p><p>We could cover:</p><ul><li>ServeMux patterns</li><li>Middleware chains</li><li>Context management</li><li>Error handling patterns</li></ul><p>Let's sync next week to outline the content?</p><p>Cheers!</p>",
			Snippet:     "Bob, A joint workshop sounds awesome! I was thinking of submitting...",
			IsRead:      true, IsSent: true,
			Labels:     []string{"sent", "all"},
			SentAt:     timePtr(ago(10, 9, 0)),
			ReceivedAt: ago(10, 9, 0), CreatedAt: ago(10, 9, 0), UpdatedAt: ago(10, 9, 0),
		},
		types.Email{
			ID: "email-023", ThreadID: thread15,
			MessageID:   "<msg-023@startup.com>",
			InReplyTo:   "<msg-022@example.com>",
			References:  []string{"<msg-021@startup.com>", "<msg-022@example.com>"},
			FromAddress: "bob.martinez@startup.com", FromName: "Bob Martinez",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Re: GopherCon 2026 - Talk Submission Deadline",
			BodyText:    "Love it! Let's do it.\n\nI'll start a shared doc for the workshop outline. I'll send the link tomorrow.\n\nLet's make sure we include live coding demos - those always get the best audience engagement.\n\nBob",
			BodyHTML:    "<p>Love it! Let's do it.</p><p>I'll start a shared doc for the workshop outline. I'll send the link tomorrow.</p><p>Let's make sure we include live coding demos - those always get the best audience engagement.</p><p>Bob</p>",
			Snippet:     "Love it! Let's do it. I'll start a shared doc for the workshop outline...",
			IsRead:      true,
			Labels:      []string{"inbox", "all", "personal"},
			ReceivedAt:  ago(10, 3, 30), CreatedAt: ago(10, 3, 30), UpdatedAt: ago(10, 3, 30),
		},
	)

	// Thread 16: Billing notification (single email, spam-like)
	thread16 := "thread-016"
	emails = append(emails,
		types.Email{
			ID: "email-024", ThreadID: thread16,
			MessageID:   "<msg-024@billing.cloudhost.io>",
			FromAddress: "billing@cloudhost.io", FromName: "CloudHost Billing",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "Invoice #INV-2026-0128 - CloudHost Services",
			BodyText:    "Invoice #INV-2026-0128\n\nBilling Period: January 1 - January 31, 2026\n\nServices:\n- Compute (4x c5.xlarge): $384.00\n- Storage (500 GB SSD): $57.50\n- Bandwidth (2 TB): $18.00\n- Database (RDS PostgreSQL): $145.00\n- CDN: $12.50\n\nSubtotal: $617.00\nTax: $0.00\nTotal Due: $617.00\n\nPayment will be automatically charged to your card ending in 4242.\n\nView detailed invoice: https://cloudhost.io/billing/INV-2026-0128",
			BodyHTML:    "<h2>Invoice #INV-2026-0128</h2><p>Billing Period: January 1 - January 31, 2026</p><table><tr><td>Compute (4x c5.xlarge)</td><td>$384.00</td></tr><tr><td>Storage (500 GB SSD)</td><td>$57.50</td></tr><tr><td>Bandwidth (2 TB)</td><td>$18.00</td></tr><tr><td>Database (RDS PostgreSQL)</td><td>$145.00</td></tr><tr><td>CDN</td><td>$12.50</td></tr><tr><td><strong>Total Due</strong></td><td><strong>$617.00</strong></td></tr></table>",
			Snippet:     "Invoice #INV-2026-0128. Billing Period: January 1 - January 31, 2026. Total Due: $617.00...",
			IsRead:      true,
			Labels:      []string{"inbox", "all", "finance"},
			ReceivedAt:  ago(3, 6, 0), CreatedAt: ago(3, 6, 0), UpdatedAt: ago(3, 6, 0),
		},
	)

	// Thread 17: Team lunch poll (single email)
	thread17 := "thread-017"
	emails = append(emails,
		types.Email{
			ID: "email-025", ThreadID: thread17,
			MessageID:   "<msg-025@product.co>",
			FromAddress: "karen.patel@product.co", FromName: "Karen Patel",
			ToAddresses: []types.Recipient{
				{Name: "Me", Address: "me@example.com"},
				{Name: "Alice Chen", Address: "alice.chen@techcorp.io"},
				{Name: "Bob Martinez", Address: "bob.martinez@startup.com"},
			},
			Subject:  "Team lunch this Friday?",
			BodyText: "Hey everyone!\n\nAnyone up for team lunch this Friday? I found a great new ramen place near the office.\n\nOptions:\n1. Ichiran Ramen - 11:30 AM\n2. Sushi Katsu - 12:00 PM\n3. Thai Basil - 12:30 PM\n\nReply with your vote!\n\nKaren",
			BodyHTML: "<p>Hey everyone!</p><p>Anyone up for team lunch this Friday? I found a great new ramen place near the office.</p><p>Options:</p><ol><li>Ichiran Ramen - 11:30 AM</li><li>Sushi Katsu - 12:00 PM</li><li>Thai Basil - 12:30 PM</li></ol><p>Reply with your vote!</p><p>Karen</p>",
			Snippet:  "Hey everyone! Anyone up for team lunch this Friday? I found a great new ramen place...",
			Labels:   []string{"inbox", "all", "personal"},
			ReceivedAt: ago(0, 3, 0), CreatedAt: ago(0, 3, 0), UpdatedAt: ago(0, 3, 0),
		},
	)

	// Thread 18: Ongoing project update (4 emails)
	thread18 := "thread-018"
	emails = append(emails,
		types.Email{
			ID: "email-026", ThreadID: thread18,
			MessageID:   "<msg-026@techcorp.io>",
			FromAddress: "alice.chen@techcorp.io", FromName: "Alice Chen",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}, {Name: "Leo Nguyen", Address: "leo.nguyen@engineering.dev"}},
			Subject:     "Incident Report: API Latency Spike (Jan 25)",
			BodyText:    "Team,\n\nWe experienced an API latency spike on January 25th between 14:00-14:45 UTC. Here's the preliminary analysis:\n\nImpact:\n- P95 latency increased from 120ms to 2.3s\n- 3.2% of requests received 503 errors\n- Approximately 1,200 users affected\n\nRoot Cause:\n- Database connection pool exhaustion\n- Triggered by a slow query in the search endpoint\n\nI'm working on the full post-mortem. Will share by EOD tomorrow.\n\nAlice",
			BodyHTML:    "<p>Team,</p><p>We experienced an API latency spike on January 25th between 14:00-14:45 UTC. Here's the preliminary analysis:</p><p><strong>Impact:</strong></p><ul><li>P95 latency increased from 120ms to 2.3s</li><li>3.2% of requests received 503 errors</li><li>Approximately 1,200 users affected</li></ul><p><strong>Root Cause:</strong></p><ul><li>Database connection pool exhaustion</li><li>Triggered by a slow query in the search endpoint</li></ul><p>I'm working on the full post-mortem. Will share by EOD tomorrow.</p><p>Alice</p>",
			Snippet:     "Team, We experienced an API latency spike on January 25th between 14:00-14:45 UTC...",
			IsRead:      true, IsImportant: true,
			Labels:     []string{"inbox", "all", "work"},
			ReceivedAt: ago(9, 16, 0), CreatedAt: ago(9, 16, 0), UpdatedAt: ago(9, 16, 0),
		},
		types.Email{
			ID: "email-027", ThreadID: thread18,
			MessageID:   "<msg-027@engineering.dev>",
			InReplyTo:   "<msg-026@techcorp.io>",
			References:  []string{"<msg-026@techcorp.io>"},
			FromAddress: "leo.nguyen@engineering.dev", FromName: "Leo Nguyen",
			ToAddresses: []types.Recipient{{Name: "Alice Chen", Address: "alice.chen@techcorp.io"}, {Name: "Me", Address: "me@example.com"}},
			Subject:     "Re: Incident Report: API Latency Spike (Jan 25)",
			BodyText:    "Alice,\n\nI've identified the slow query. It was a full table scan on the search_logs table due to a missing index. I've pushed a fix:\n\nCREATE INDEX idx_search_logs_created_at ON search_logs(created_at);\n\nThis should prevent recurrence. I've also added connection pool monitoring alerts.\n\nLeo",
			BodyHTML:    "<p>Alice,</p><p>I've identified the slow query. It was a full table scan on the <code>search_logs</code> table due to a missing index. I've pushed a fix:</p><pre><code>CREATE INDEX idx_search_logs_created_at ON search_logs(created_at);</code></pre><p>This should prevent recurrence. I've also added connection pool monitoring alerts.</p><p>Leo</p>",
			Snippet:     "Alice, I've identified the slow query. It was a full table scan on the search_logs table...",
			IsRead:      true,
			Labels:      []string{"inbox", "all", "work"},
			ReceivedAt:  ago(9, 12, 0), CreatedAt: ago(9, 12, 0), UpdatedAt: ago(9, 12, 0),
		},
		types.Email{
			ID: "email-028", ThreadID: thread18,
			MessageID:   "<msg-028@techcorp.io>",
			InReplyTo:   "<msg-027@engineering.dev>",
			References:  []string{"<msg-026@techcorp.io>", "<msg-027@engineering.dev>"},
			FromAddress: "alice.chen@techcorp.io", FromName: "Alice Chen",
			ToAddresses: []types.Recipient{{Name: "Leo Nguyen", Address: "leo.nguyen@engineering.dev"}, {Name: "Me", Address: "me@example.com"}},
			Subject:     "Re: Incident Report: API Latency Spike (Jan 25)",
			BodyText:    "Great catch, Leo! I've verified the fix in staging - query time dropped from 4.2s to 12ms.\n\nAction items for the post-mortem:\n1. Add index (done)\n2. Set up connection pool monitoring (done)\n3. Add query timeout of 5s for search endpoints\n4. Review all queries without EXPLAIN ANALYZE\n\nI'll schedule the post-mortem review for Thursday.\n\nAlice",
			BodyHTML:    "<p>Great catch, Leo! I've verified the fix in staging - query time dropped from 4.2s to 12ms.</p><p>Action items for the post-mortem:</p><ol><li>Add index (done)</li><li>Set up connection pool monitoring (done)</li><li>Add query timeout of 5s for search endpoints</li><li>Review all queries without EXPLAIN ANALYZE</li></ol><p>I'll schedule the post-mortem review for Thursday.</p><p>Alice</p>",
			Snippet:     "Great catch, Leo! I've verified the fix in staging - query time dropped from 4.2s to 12ms...",
			IsRead:      true,
			Labels:      []string{"inbox", "all", "work"},
			ReceivedAt:  ago(9, 8, 0), CreatedAt: ago(9, 8, 0), UpdatedAt: ago(9, 8, 0),
		},
		types.Email{
			ID: "email-029", ThreadID: thread18,
			MessageID:   "<msg-029@example.com>",
			InReplyTo:   "<msg-028@techcorp.io>",
			References:  []string{"<msg-026@techcorp.io>", "<msg-027@engineering.dev>", "<msg-028@techcorp.io>"},
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "Alice Chen", Address: "alice.chen@techcorp.io"}, {Name: "Leo Nguyen", Address: "leo.nguyen@engineering.dev"}},
			Subject:     "Re: Incident Report: API Latency Spike (Jan 25)",
			BodyText:    "Thanks for the quick turnaround on this. I'll also add automated slow query detection to our CI pipeline so we catch these before they hit production.\n\nSee you Thursday for the post-mortem review.",
			BodyHTML:    "<p>Thanks for the quick turnaround on this. I'll also add automated slow query detection to our CI pipeline so we catch these before they hit production.</p><p>See you Thursday for the post-mortem review.</p>",
			Snippet:     "Thanks for the quick turnaround on this. I'll also add automated slow query detection...",
			IsRead:      true, IsSent: true,
			Labels:     []string{"sent", "all"},
			SentAt:     timePtr(ago(9, 6, 0)),
			ReceivedAt: ago(9, 6, 0), CreatedAt: ago(9, 6, 0), UpdatedAt: ago(9, 6, 0),
		},
	)

	// Thread 19: Another draft
	thread19 := "thread-019"
	emails = append(emails,
		types.Email{
			ID: "email-030", ThreadID: thread19,
			MessageID:   "<msg-030@example.com>",
			FromAddress: "me@example.com", FromName: "Me",
			ToAddresses: []types.Recipient{{Name: "David Kim", Address: "david.kim@university.edu"}},
			Subject:     "Re: Guest Lecture Invitation - Distributed Systems",
			BodyText:    "Dear Dr. Kim,\n\nThank you for the invitation! I'd be honored to speak at your lecture series.\n\nThe proposed topic sounds perfect. I could cover:\n\n1. Service discovery patterns\n2. Circuit breakers and retry logic\n3. Distributed tracing with OpenTelemetry\n4. Event-driven architecture with Go channels\n\n[Draft - need to check calendar availability]",
			BodyHTML:    "<p>Dear Dr. Kim,</p><p>Thank you for the invitation! I'd be honored to speak at your lecture series.</p><p>The proposed topic sounds perfect. I could cover:</p><ol><li>Service discovery patterns</li><li>Circuit breakers and retry logic</li><li>Distributed tracing with OpenTelemetry</li><li>Event-driven architecture with Go channels</li></ol><p><em>[Draft - need to check calendar availability]</em></p>",
			Snippet:     "Dear Dr. Kim, Thank you for the invitation! I'd be honored to speak at your lecture series...",
			IsRead:      true, IsDraft: true,
			Labels:     []string{"drafts", "all"},
			ReceivedAt: ago(0, 1, 0), CreatedAt: ago(0, 1, 0), UpdatedAt: ago(0, 1, 0),
		},
	)

	// Thread 20: Spam email (in spam folder)
	thread20 := "thread-020"
	emails = append(emails,
		types.Email{
			ID: "email-031", ThreadID: thread20,
			MessageID:   "<msg-031@promo.deals.xyz>",
			FromAddress: "offers@amazing-deals.xyz", FromName: "Amazing Deals",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "You've Won a FREE iPhone 16 Pro Max! Claim Now!",
			BodyText:    "Congratulations! You've been selected to receive a FREE iPhone 16 Pro Max! Click here to claim your prize. Limited time offer! Act now before it expires! This is a once in a lifetime opportunity that you don't want to miss!",
			BodyHTML:    "<h1 style='color: red;'>Congratulations!</h1><p>You've been selected to receive a <strong>FREE iPhone 16 Pro Max!</strong></p><p><a href='#'>Click here to claim your prize.</a></p><p>Limited time offer! Act now before it expires!</p>",
			Snippet:     "Congratulations! You've been selected to receive a FREE iPhone 16 Pro Max! Click here...",
			IsRead:      true,
			Labels:      []string{"spam", "all"},
			ReceivedAt:  ago(2, 3, 0), CreatedAt: ago(2, 3, 0), UpdatedAt: ago(2, 3, 0),
		},
	)

	// Thread 21: Trashed email
	thread21 := "thread-021"
	emails = append(emails,
		types.Email{
			ID: "email-032", ThreadID: thread21,
			MessageID:   "<msg-032@newsletter.techblog.io>",
			FromAddress: "digest@techblog.io", FromName: "TechBlog Weekly",
			ToAddresses: []types.Recipient{{Name: "Me", Address: "me@example.com"}},
			Subject:     "TechBlog Weekly Digest - Issue #147",
			BodyText:    "This week in tech:\n\n- Go 1.25 Release Candidate Available\n- The State of WebAssembly in 2026\n- Why SQLite is the Most Used Database\n- Building AI Agents with Claude\n- React Server Components Deep Dive",
			BodyHTML:    "<h2>This week in tech:</h2><ul><li>Go 1.25 Release Candidate Available</li><li>The State of WebAssembly in 2026</li><li>Why SQLite is the Most Used Database</li><li>Building AI Agents with Claude</li><li>React Server Components Deep Dive</li></ul>",
			Snippet:     "This week in tech: Go 1.25 Release Candidate Available, The State of WebAssembly...",
			IsRead:      true,
			Labels:      []string{"trash", "all"},
			ReceivedAt:  ago(5, 10, 0), CreatedAt: ago(5, 10, 0), UpdatedAt: ago(5, 10, 0),
		},
	)

	// Insert all emails
	for i := range emails {
		if err := s.CreateEmail(ctx, &emails[i]); err != nil {
			fmt.Printf("Warning: failed to create email %s: %v\n", emails[i].ID, err)
			continue
		}
	}

	return nil
}

// timePtr returns a pointer to a time value.
func timePtr(t time.Time) *time.Time {
	return &t
}
