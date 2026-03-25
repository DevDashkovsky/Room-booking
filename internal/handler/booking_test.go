package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	page, pageSize := parsePagination(r)
	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize = %d, want 20", pageSize)
	}
}

func TestParsePagination_Custom(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/bookings/list?page=3&pageSize=50", nil)
	page, pageSize := parsePagination(r)
	if page != 3 {
		t.Errorf("page = %d, want 3", page)
	}
	if pageSize != 50 {
		t.Errorf("pageSize = %d, want 50", pageSize)
	}
}

func TestParsePagination_MaxPageSize(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/bookings/list?pageSize=999", nil)
	_, pageSize := parsePagination(r)
	if pageSize != 100 {
		t.Errorf("pageSize = %d, want 100 (capped)", pageSize)
	}
}

func TestParsePagination_InvalidValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/bookings/list?page=abc&pageSize=-1", nil)
	page, pageSize := parsePagination(r)
	if page != 1 {
		t.Errorf("page = %d, want 1 (default)", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize = %d, want 20 (default)", pageSize)
	}
}
