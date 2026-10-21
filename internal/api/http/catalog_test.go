package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/isbe-bot/bottagon/internal/catalog"
	"github.com/isbe-bot/bottagon/internal/matchmaker"
)

func TestCatalogListSearchPageLookupAndCreate(t *testing.T) {
	server := NewServer(matchmaker.NewQueue())
	server.SetCatalog(catalog.NewMemory())
	router := server.Routes()
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	bots := get("/bots?role=contender&limit=2&page=2")
	if bots.Code != http.StatusOK {
		t.Fatal(bots.Body.String())
	}
	var botPage struct {
		Items  []catalog.Bot `json:"items"`
		Total  int           `json:"total"`
		Page   int           `json:"page"`
		Offset int           `json:"offset"`
	}
	if err := json.Unmarshal(bots.Body.Bytes(), &botPage); err != nil {
		t.Fatal(err)
	}
	if botPage.Total != 8 || botPage.Page != 2 || botPage.Offset != 2 || len(botPage.Items) != 2 {
		t.Fatalf("%+v", botPage)
	}

	matches := get("/matches?q=habit&outcome=contender&stake=20&limit=6")
	var matchPage struct {
		Items []catalog.Match `json:"items"`
		Total int             `json:"total"`
	}
	if err := json.Unmarshal(matches.Body.Bytes(), &matchPage); err != nil || matchPage.Total == 0 {
		t.Fatalf("%d %v %s", matches.Code, err, matches.Body.String())
	}
	if matchPage.Items[0].Rounds[0].QuoteA == "" || matchPage.Items[0].A.Facts.Record == "" {
		t.Fatalf("list payload missing rounds or facts: %+v", matchPage.Items[0])
	}

	one := get("/matches/AR-1048")
	var match catalog.Match
	if err := json.Unmarshal(one.Body.Bytes(), &match); err != nil || match.A.ID != "sovereign" || len(match.Rounds) != 7 {
		t.Fatalf("%s %v", one.Body.String(), err)
	}
	if missing := get("/bots/nobody"); missing.Code != http.StatusNotFound {
		t.Fatal(missing.Code)
	}
	log := get("/matches?bot=blinky&limit=50")
	var blinky struct{ Total int }
	if err := json.Unmarshal(log.Body.Bytes(), &blinky); err != nil || blinky.Total != 8 {
		t.Fatalf("%+v %s", blinky, log.Body.String())
	}
	show := get("/exhibition")
	var exhibit catalog.Exhibition
	if err := json.Unmarshal(show.Body.Bytes(), &exhibit); err != nil || exhibit.Left.Name != "Sovereign-1" || len(exhibit.Rounds) != 7 {
		t.Fatal(show.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/bots", strings.NewReader(`{"name":"New Contender","email":"owner@example.com","persona":"Reads the room and counters."}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "owner@example.com") {
		t.Fatal("registration response includes the owner email")
	}
	roster := get("/bots?q=New+Contender")
	if strings.Contains(roster.Body.String(), "owner@example.com") {
		t.Fatal("roster includes the owner email")
	}
	trap := httptest.NewRequest(http.MethodPost, "/bots", strings.NewReader(`{"name":"Trap Bot","email":"trap@example.com","persona":"Reads the room and counters.","website":"http://spam.example"}`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, trap)
	if w.Code != http.StatusBadRequest {
		t.Fatal(w.Code, w.Body.String())
	}
	dup := httptest.NewRequest(http.MethodPost, "/bots", strings.NewReader(`{"name":"new contender","email":"other@example.com","persona":"Reads the room and counters."}`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, dup)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "already") {
		t.Fatal(w.Code, w.Body.String())
	}
	bad := get("/matches?outcome=nope")
	if bad.Code != http.StatusBadRequest {
		t.Fatal(bad.Code)
	}
}
