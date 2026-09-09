package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePagination(t *testing.T) {
	for _, tt := range []struct {
		query      string
		page, size int
		ok         bool
	}{
		{"", 1, 20, true}, {"?page=3&pageSize=50", 3, 50, true}, {"?pageSize=100", 1, 100, true},
		{"?pageSize=101", 0, 0, false}, {"?page=0", 0, 0, false}, {"?pageSize=0", 0, 0, false},
		{"?page=abc", 0, 0, false}, {"?pageSize=-1", 0, 0, false}, {"?page=", 0, 0, false},
		{"?page=9223372036854775807&pageSize=100", 0, 0, false}, {"?page=99999999999999999999999", 0, 0, false}, {"?page=1&page=2", 0, 0, false},
	} {
		t.Run(tt.query, func(t *testing.T) {
			p, s, ok := parsePagination(httptest.NewRequest(http.MethodGet, "/bookings/list"+tt.query, nil))
			if p != tt.page || s != tt.size || ok != tt.ok {
				t.Fatalf("got %d %d %v", p, s, ok)
			}
		})
	}
}
