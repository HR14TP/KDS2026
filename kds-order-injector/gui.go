package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

//go:embed gui.html
var guiFS embed.FS

type sendResult struct {
	ID     string `json:"id"`
	Table  string `json:"table"`
	Waiter string `json:"waiter"`
	Items  int    `json:"items"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

type pasteRequest struct {
	Host string `json:"host"`
	Body string `json:"body"`
}

type ticketRequest struct {
	Host   string `json:"host"`
	Body   string `json:"body"`
	Screen string `json:"screen"`
}

type randomRequest struct {
	Host       string `json:"host"`
	Count      int    `json:"count"`
	IntervalMs int    `json:"intervalMs"`
	Table      string `json:"table"`
	Waiter     string `json:"waiter"`
	Items      int    `json:"items"`
	Seed       int64  `json:"seed"`
}

// runGUI serves a single-page browser UI for pasting/sending orders, so
// testing doesn't require remembering CLI flags or hand-quoting JSON.
// Everything (HTML/CSS/JS) is embedded in the binary - no external files or
// network access needed once it's running, same as the CLI.
func runGUI(addr string) {
	client := &http.Client{Timeout: 10 * time.Second}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := guiFS.ReadFile("gui.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		host := r.URL.Query().Get("host")
		if host == "" {
			http.Error(w, "host required", http.StatusBadRequest)
			return
		}
		err := doPing(client, host, false)
		writeJSON(w, map[string]any{"ok": err == nil, "error": errString(err)})
	})

	mux.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		var req pasteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		orders, err := parseOrders([]byte(req.Body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results := make([]sendResult, 0, len(orders))
		for _, order := range orders {
			err := sendOrder(client, req.Host, order, false)
			results = append(results, sendResult{
				ID: order.Id, Table: order.TableName, Waiter: order.WaiterName,
				Items: len(order.Items), OK: err == nil, Error: errString(err),
			})
		}
		writeJSON(w, results)
	})

	mux.HandleFunc("/api/parse-ticket", func(w http.ResponseWriter, r *http.Request) {
		var req ticketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		order := ticketToSallyOrder(parseTicketText(req.Body), req.Screen)
		writeJSON(w, order)
	})

	mux.HandleFunc("/api/send-ticket", func(w http.ResponseWriter, r *http.Request) {
		var req ticketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		order := ticketToSallyOrder(parseTicketText(req.Body), req.Screen)
		err := sendOrder(client, req.Host, order, false)
		writeJSON(w, sendResult{
			ID: order.Id, Table: order.TableName, Waiter: order.WaiterName,
			Items: len(order.Items), OK: err == nil, Error: errString(err),
		})
	})

	mux.HandleFunc("/api/random", func(w http.ResponseWriter, r *http.Request) {
		var req randomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Count <= 0 {
			req.Count = 1
		}
		seed := req.Seed
		if seed == 0 {
			seed = time.Now().UnixNano()
		}
		rng := rand.New(rand.NewSource(seed))

		results := make([]sendResult, 0, req.Count)
		for i := 0; i < req.Count; i++ {
			order := randomOrder(rng, req.Table, req.Waiter, req.Items)
			err := sendOrder(client, req.Host, order, false)
			results = append(results, sendResult{
				ID: order.Id, Table: order.TableName, Waiter: order.WaiterName,
				Items: len(order.Items), OK: err == nil, Error: errString(err),
			})
			if i < req.Count-1 && req.IntervalMs > 0 {
				time.Sleep(time.Duration(req.IntervalMs) * time.Millisecond)
			}
		}
		writeJSON(w, results)
	})

	url := fmt.Sprintf("http://%s", addr)
	log.Printf("kds-order-injector GUI running at %s (Ctrl+C to stop)", url)
	go openBrowser(url)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func openBrowser(url string) {
	time.Sleep(300 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
