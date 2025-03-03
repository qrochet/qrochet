package roh

import "testing"

func TestNewServer(t *testing.T) {
	srv := NewServer("localhost:2044")
	t.Cleanup(func() { _ = srv.Close() })
	go srv.ListenAndServe()
	_ = <-srv.State
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("http://localhost:2044")
	if err != nil {
		t.Fatalf("NewClient: %s", err)
	}
	_ = client
}

type RequestTest struct {
	Name string `json:name`
	ID   int    `json:id`
}

type ResponseTest struct {
	Names []string `json:names`
	IDs   []int    `json:ids`
}

func TestInteraction(t *testing.T) {
	ready := make(chan (struct{}))

	srv := NewServer("localhost:2044")
	t.Cleanup(func() { _ = srv.Close() })

	Register(srv, "/request", func(request RequestTest) (ResponseTest, error) {
		return ResponseTest{Names: []string{"one", "two"}}, nil
	})

	go func() {
		go srv.ListenAndServe()
		ready <- struct{}{}
	}()
	_ = <-ready

	client, err := NewClient("http://localhost:2044")
	if err != nil {
		t.Fatalf("NewClient: %s", err)
	}
	request := RequestTest{}
	re, err := Call[ResponseTest](client, "/request", request)
	if err != nil {
		t.Fatalf("Call: %s", err)
	}
	if len(re.Names) != 2 || re.Names[0] != "one " || re.Names[1] != "two" {
		t.Fatalf("Call response: %v", re)
	}

}
