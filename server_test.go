package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Nelwhix/todo"
)

func setupAPI(t *testing.T) (string, func()) {
	t.Helper()
	tempTodoFile, err := os.CreateTemp("", "todotest")

	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(newMux(tempTodoFile.Name()))

	for i := 1; i < 3; i++ {
		var body bytes.Buffer
		taskName := fmt.Sprintf("Task Number %d.", i)

		item := struct {
			Task string `json:"task"`
		} {
			Task: taskName,
		}

		if err := json.NewEncoder(&body).Encode(item); err != nil {
			t.Fatal(err)
		}

		r, err := http.Post(ts.URL + "/todo", "application/json", &body)

		if err != nil {
			t.Fatal(err)
		}

		if r.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to add initial items: Status: %d", r.StatusCode)
		}
	}


	return ts.URL, func() {
		ts.Close()
		os.Remove(tempTodoFile.Name())
	}
}

func TestGet(t *testing.T) {
	testCases := []struct {
		name string
		path string
		expCode int
		expitems int
		expContent string
	} {
		{
			name: "Server starts",
			path: "/",
			expContent: "There's an API here",
			expCode: http.StatusOK,
		},
		{
			name: "undefined routes return 404",
			path: "/todo/500",
			expCode: http.StatusNotFound,
			expContent: "Not Found\n",
		},
		{
			name: "it gets all tasks",
			path: "/todo",
			expCode: http.StatusOK,
			expitems: 2,
			expContent: "Task Number 1.",
		}, 
		{
			name: "it can get one task",
			path: "/todo/1",
			expCode: http.StatusOK,
			expitems: 1,
			expContent: "Task Number 1.",
		},
	}

	url, cleanup := setupAPI(t)
	defer cleanup()

	var (
		resp struct {
			Results todo.List `json:"results"`
			Date int64 `json:"date"`
			TotalResults int `json:"total_results"`
		} 
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				body []byte
				err error
			)

			r, err := http.Get(url + tc.path)
			if err != nil {
				t.Error(err)
			}

			defer r.Body.Close()

			if r.StatusCode != tc.expCode {
				t.Fatalf("Expected %q, got %q.", http.StatusText(tc.expCode), http.StatusText(r.StatusCode))
			}

			switch {
			case r.Header.Get("Content-Type") == "application/json":
				if err = json.NewDecoder(r.Body).Decode(&resp); err != nil {
					t.Error(err)
				}
				if resp.TotalResults != tc.expitems {
					t.Errorf("Expected %d items, got %d.", tc.expitems, resp.TotalResults)
				}
				if resp.Results[0].Task != tc.expContent {
					t.Errorf("Expected %q, got %q.", tc.expContent, resp.Results[0].Task)
				}
			case strings.Contains(r.Header.Get("Content-Type"), "text/plain"):
				if body, err = io.ReadAll(r.Body); err != nil {
					t.Error(err)
				}

				if !strings.Contains(string(body), tc.expContent) {
					t.Errorf("Expected %q, got %q. ", tc.expContent, string(body))
				}
			default:
				t.Fatalf("Unsupported Content-Type: %q", r.Header.Get("Content-Type")) 
			}

		})
	}
}

func TestAdd(t *testing.T) {
	url, cleanup := setupAPI(t)
	defer cleanup()

	taskName := "Task number 3."

	t.Run("it can add a new todo", func(t *testing.T) {
		var body bytes.Buffer
		
		item := struct {
			Task string `json:"task"`
		} {
			Task: taskName,
		}

		if err := json.NewEncoder(&body).Encode(item); err != nil {
			t.Fatal(err)
		}

		r, err := http.Post(url + "/todo", "application/json", &body)
		if err != nil {
			t.Fatal(err)
		}

		if r.StatusCode != http.StatusCreated {
			t.Errorf("Expected %q, got %q.", http.StatusText(http.StatusCreated), http.StatusText(r.StatusCode))
		}
	})

	t.Run("it can retrieve one todo", func(t *testing.T) {
		r, err := http.Get(url + "/todo/3")

		if err != nil {
			t.Error(err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("Expected %q, got %q.", http.StatusText(http.StatusOK), http.StatusText(r.StatusCode))
		}

		var resp todoResponse
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		r.Body.Close()

		if resp.Results[0].Task != taskName {
			t.Errorf("Expected %q, got %q.", taskName, resp.Results[0].Task)
		}
	})
}

func TestDelete(t *testing.T) {
	url, cleanup := setupAPI(t)
	defer cleanup()

	t.Run("it can delete tasks", func (t *testing.T)  {
		u := fmt.Sprintf("%s/todo/1", url)
		req, err := http.NewRequest(http.MethodDelete, u, nil)

		if err != nil {
			t.Fatal(err)
		}
		r, err := http.DefaultClient.Do(req)

		if err != nil {
			t.Error(err)
		}

		if r.StatusCode != http.StatusNoContent {
			t.Fatalf("Expected %q, got %q.", http.StatusText(http.StatusNoContent), http.StatusText(r.StatusCode))
		}
	})

	t.Run("Check Task is Deleted", func(t *testing.T) {
		r, err := http.Get(url + "/todo")

		if err != nil {
			t.Error(err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("Expected %q, got %q.",
			http.StatusText(http.StatusOK), http.StatusText(r.StatusCode))
		}
		var resp todoResponse
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
			r.Body.Close()
		if len(resp.Results) != 1 {
			t.Errorf("Expected 1 item, got %d.", len(resp.Results))
		}
		expTask := "Task Number 2."
		if resp.Results[0].Task != expTask {
			t.Errorf("Expected %q, got %q.", expTask, resp.Results[0].Task)
		}
	})
}

func TestComplete(t *testing.T) {
	url, cleanup := setupAPI(t)
	defer cleanup()
	
	t.Run("Complete", func(t *testing.T) {
		u := fmt.Sprintf("%s/todo/1?complete", url)
		req, err := http.NewRequest(http.MethodPatch, u, nil)
	
		if err != nil {
			t.Fatal(err)
		}
		r, err := http.DefaultClient.Do(req)
	
		if err != nil {
			t.Error(err)
		}
	
		if r.StatusCode != http.StatusNoContent {
			t.Fatalf("Expected %q, got %q.",
			http.StatusText(http.StatusNoContent), http.StatusText(r.StatusCode))
		}
	})

	t.Run("CheckComplete", func(t *testing.T) {
		r, err := http.Get(url + "/todo")
		if err != nil {
			t.Error(err)
		}
	
		if r.StatusCode != http.StatusOK {
			t.Fatalf("Expected %q, got %q.",
			http.StatusText(http.StatusOK), http.StatusText(r.StatusCode))
		}
	
		var resp todoResponse
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		
		r.Body.Close()
	
		if len(resp.Results) != 2 {
			t.Errorf("Expected 2 items, got %d.", len(resp.Results))
		}
	
		if !resp.Results[0].Done {
			t.Error("Expected item 1 to be completed")
		}
		
		if resp.Results[1].Done {
			t.Error("Expected item 2 not to be completed")
		}
	})
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}


