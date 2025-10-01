package remotemcphost

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerListing(t *testing.T) {
	ctx := context.Background()

	host, _ := NewMcpHost(&McpHostOptions{Lm: echoLm{}})
	host.AddSessionsFromConfig(ctx, strings.NewReader("![../test_servers/greetings][greetings] go run greetings.go"), nil)

	mux := NewRemoteMcpMux(&host)
	r := httptest.NewRequest("GET", "/servers", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
	var listing McpServerList
	json.NewDecoder(res.Body).Decode(&listing)

	if len(listing.Servers) != 1 {
		t.Errorf("Expected 1 MCP server(s) but found %d", len(listing.Servers))
	} else if listing.Servers[0].Name != "greetings" {
		t.Errorf("Expected the MCP server(s) to have names %v but had names %v", []string{"greetings"}, listing.Servers)
	}

}

func TestServerGenerate(t *testing.T) {
	ctx := context.Background()

	host, _ := NewMcpHost(&McpHostOptions{Lm: echoLm{}})
	host.AddSessionsFromConfig(ctx, strings.NewReader("![../test_servers/greetings][greetings] go run greetings.go"), nil)

	mux := NewRemoteMcpMux(&host)

	req, _ := json.Marshal(GenerationRequest{
		Messages: []Message{{
			Role:  "user",
			Parts: []UnionPart{{NewTextPart("hello, world")}},
		}},
	})

	r := httptest.NewRequest("POST", "/generations", strings.NewReader(string(req)))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("expected status OK; got %v with body '%s'", res.Status, body)
	}
	var genRes GenerationResponse
	err := json.NewDecoder(res.Body).Decode(&genRes)

	if err != nil {
		t.Fatalf("Could not decode response")
	}

	if genRes.Message.Role != "model" {
		t.Errorf("Expected the response's role to be \"model\", fount \"%s\"", genRes.Message.Role)
	}

	if len(genRes.Message.Parts) == 0 {
		t.Fatalf("Expected the response to have at least one part but had none")
	}
	if tp, ok := genRes.Message.Parts[0].Part.(TextPart); !ok || tp.Text != "hello, world" {
		t.Fatalf("Expected the response have a text part containing \"hello, world\" as its first part, but it did not. Instead found %v", genRes.Message)
	}
}

type echoLm struct {
}

func (lm echoLm) Generate(messages []Message, opts *GenerateOptions) (*GenerateResult, error) {
	text := "nothing to echo"
	if len(messages) > 0 && len(messages[len(messages)-1].Parts) > 0 {
		if tp, ok := messages[len(messages)-1].Parts[len(messages[len(messages)-1].Parts)-1].Part.(TextPart); ok {
			text = tp.Text
		}
	}
	return &GenerateResult{
		Parts: []UnionPart{{Part: NewTextPart(text)}},
		Stop:  true,
	}, nil
}
