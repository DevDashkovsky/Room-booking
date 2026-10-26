package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type deadlineAuthService struct{ seen context.Context }

func (s *deadlineAuthService) Register(context.Context, string, string, string) (*domain.User, error) {
	return nil, nil
}
func (s *deadlineAuthService) Login(ctx context.Context, _, _ string) (string, error) {
	s.seen = ctx
	return "token", nil
}

func TestRouter_PropagatesAndReleasesRequestDeadline(t *testing.T) {
	for _, parentDeadline := range []bool{false, true} {
		t.Run(fmt.Sprint(parentDeadline), func(t *testing.T) {
			svc := &deadlineAuthService{}
			router := NewRouter(Deps{AuthH: NewAuthHandler("secret", svc)})
			parent := t.Context()
			if parentDeadline {
				var cancel context.CancelFunc
				parent, cancel = context.WithTimeout(parent, time.Second)
				defer cancel()
			}
			req := httptest.NewRequestWithContext(parent, http.MethodPost, "/login", strings.NewReader(`{"email":"u@example.com","password":"pass"}`))
			before := time.Now()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK || svc.seen == nil {
				t.Fatalf("login not called: %d", w.Code)
			}
			deadline, ok := svc.seen.Deadline()
			if !ok || deadline.Before(before) || deadline.After(time.Now().Add(requestTimeout)) {
				t.Fatalf("invalid deadline: %v, %v", deadline, ok)
			}
			if parentDeadline {
				want, _ := parent.Deadline()
				if !deadline.Equal(want) {
					t.Fatalf("parent deadline extended: %v != %v", deadline, want)
				}
			}
			if svc.seen.Err() != context.Canceled {
				t.Fatal("request context not released after response")
			}
		})
	}
}

func TestHandleServiceError_DeadlineContract(t *testing.T) {
	w := httptest.NewRecorder()
	handleServiceError(w, fmt.Errorf("query: %w", context.DeadlineExceeded))
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusServiceUnavailable || body.Error.Code != "UNAVAILABLE" {
		t.Fatalf("deadline response: %d %+v", w.Code, body)
	}
}
