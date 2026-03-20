package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Success(c, map[string]string{"id": "1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data wrapper")
	}
	if data["id"] != "1" {
		t.Errorf("expected id=1, got %v", data["id"])
	}
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Error(c, http.StatusNotFound, "Not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "Not found" {
		t.Errorf("expected 'Not found', got %v", body["error"])
	}
}

func TestPaginated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Paginated(c, []int{1, 2, 3}, 10, 1, 3)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Pagination.Total != 10 {
		t.Errorf("expected total=10, got %d", body.Pagination.Total)
	}
	if body.Pagination.TotalPages != 4 {
		t.Errorf("expected totalPages=4, got %d", body.Pagination.TotalPages)
	}
}

func TestGetPagination_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=2&pageSize=10", nil)

	p := GetPagination(c)

	if p.Page != 2 {
		t.Errorf("expected page=2, got %d", p.Page)
	}
	if p.PageSize != 10 {
		t.Errorf("expected pageSize=10, got %d", p.PageSize)
	}
	if p.Offset != 10 {
		t.Errorf("expected offset=10, got %d", p.Offset)
	}
}
