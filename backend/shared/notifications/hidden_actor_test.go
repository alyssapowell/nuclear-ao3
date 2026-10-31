package notifications

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
)

// capturingRepo keeps what the service saves.
type capturingRepo struct {
	mockNotificationRepo
	saved []*models.NotificationItem
}

func (r *capturingRepo) CreateNotification(ctx context.Context, n *models.NotificationItem) error {
	r.saved = append(r.saved, n)
	return nil
}

// A hidden creator's account id decides who is told, but never reaches what
// is stored for the recipient.
func TestHiddenActorIsNotStored(t *testing.T) {
	for _, hidden := range []bool{true, false} {
		repo := &capturingRepo{}
		svc := NewNotificationService(&mockMessageService{}, &mockSubscriptionRepo{}, repo,
			&mockDigestRepo{}, &mockPreferenceRepo{}, NotificationServiceConfig{})
		actor, recipient := uuid.New(), uuid.New()
		err := svc.ProcessEvent(context.Background(), &EventData{
			Type:        models.EventCommentReceived,
			SourceID:    uuid.New(),
			SourceType:  "work",
			Title:       "New comment on work",
			ActorID:     &actor,
			ActorName:   "Anonymous Creator",
			ActorHidden: hidden,
			Recipients:  []uuid.UUID{recipient, actor},
		})
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, n := range repo.saved {
			if n.UserID == actor {
				t.Errorf("the actor was told about their own comment")
			}
			if n.UserID != recipient {
				continue
			}
			found = true
			if hidden && n.ActorID != nil {
				t.Errorf("hidden actor: stored actor id %v", n.ActorID)
			}
			if !hidden && (n.ActorID == nil || *n.ActorID != actor) {
				t.Errorf("visible actor: stored actor id %v, want %v", n.ActorID, actor)
			}
		}
		if !found {
			t.Fatalf("hidden=%v: the recipient was not notified (saved %d)", hidden, len(repo.saved))
		}
	}
}
