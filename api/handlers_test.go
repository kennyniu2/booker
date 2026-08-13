package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func newTestRouter(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mockDB.Close() })
	return SetupRouter(sqlx.NewDb(mockDB, "sqlmock")), mock
}

func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAddBook(t *testing.T) {
	r, mock := newTestRouter(t)
	mock.ExpectQuery("INSERT INTO books").
		WithArgs("Dune", "Frank Herbert", "", "", "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

	w := do(r, "POST", "/books", `{"title":"Dune","author":"Frank Herbert"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	var resp struct {
		Book Book `json:"book"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Book.ID != 3 {
		t.Errorf("book id = %d, want 3", resp.Book.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestAddBookValidation(t *testing.T) {
	r, _ := newTestRouter(t)
	w := do(r, "POST", "/books", `{"title":"No author"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestUpdateRating(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		affected int64
		wantCode int
		expectDB bool
	}{
		{"ok", `{"rating":4,"review":"great"}`, 1, http.StatusOK, true},
		{"not found", `{"rating":4}`, 0, http.StatusNotFound, true},
		{"rating too high", `{"rating":6}`, 0, http.StatusBadRequest, false},
		{"rating missing", `{"review":"x"}`, 0, http.StatusBadRequest, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, mock := newTestRouter(t)
			if tt.expectDB {
				mock.ExpectExec("UPDATE user_books").
					WillReturnResult(sqlmock.NewResult(0, tt.affected))
			}
			w := do(r, "PUT", "/user-books/1/rating", tt.body)
			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d (body %s)", w.Code, tt.wantCode, w.Body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRemoveBook(t *testing.T) {
	for _, tt := range []struct {
		affected int64
		want     int
	}{{1, http.StatusOK}, {0, http.StatusNotFound}} {
		r, mock := newTestRouter(t)
		mock.ExpectExec("DELETE FROM user_books").
			WithArgs("5").
			WillReturnResult(sqlmock.NewResult(0, tt.affected))
		if w := do(r, "DELETE", "/user-books/5", ""); w.Code != tt.want {
			t.Errorf("affected=%d: status = %d, want %d", tt.affected, w.Code, tt.want)
		}
	}
}

func TestGetUserBooks(t *testing.T) {
	t.Run("missing user_id", func(t *testing.T) {
		r, _ := newTestRouter(t)
		if w := do(r, "GET", "/user-books", ""); w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
	t.Run("invalid user_id", func(t *testing.T) {
		r, _ := newTestRouter(t)
		if w := do(r, "GET", "/user-books?user_id=abc", ""); w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
	t.Run("returns books", func(t *testing.T) {
		r, mock := newTestRouter(t)
		cols := []string{"id", "user_id", "book_id", "status", "rating", "review", "progress",
			"title", "author", "description", "isbn", "cover_url"}
		mock.ExpectQuery("SELECT ub.id").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow(1, 1, 2, "completed", 5, "loved it", 100, "Dune", "Frank Herbert", "d", "i", "c"))
		w := do(r, "GET", "/user-books?user_id=1", "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", w.Code, w.Body)
		}
		var resp struct {
			Count int `json:"count"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Count != 1 {
			t.Errorf("count = %d, want 1", resp.Count)
		}
	})
}
