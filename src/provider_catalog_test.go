package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseModelCatalogOpenRouterShape(t *testing.T) {
	raw := []byte(`{"data":[{"id":"acme/fast","name":"Fast","context_length":200000,"pricing":{"prompt":"0.000001","completion":"0.000002"}},{"id":"plain"}]}`)
	got, err := parseModelCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	if got[0].ContextLimit != 200000 || got[0].PromptPrice != "0.000001" {
		t.Fatalf("%#v", got[0])
	}
	if got[1].ID != "plain" || got[1].ContextLimit != 0 {
		t.Fatalf("%#v", got[1])
	}
}

func TestFetchProviderModelCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"m1","context_window":8192}]}`))
	}))
	defer srv.Close()
	models, err := fetchProviderModelCatalog(context.Background(), srv.URL+"/v1", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "m1" || models[0].ContextLimit != 8192 {
		t.Fatalf("%#v", models)
	}
}
