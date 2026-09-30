// kds-order-injector sends test orders directly into a running kds-webserver
// instance, the same way the POS ("Sally") integration does when it POSTs to
// /order. It exists so orders can be injected on a KDS terminal for testing
// without needing an actual POS terminal or print feed running.
//
// The JSON shape below mirrors kds-golang-lib's kdsmodels.SallyOrder exactly
// (field-for-field) rather than importing that module, so this tool has no
// dependency on the private kds-golang-lib repo and builds standalone.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type SallyOrder struct {
	Id            string           `json:"id"`
	ProductCenter string           `json:"productCenter"`
	TableName     string           `json:"tableName"`
	WaiterName    string           `json:"waiterName"`
	DeviceID      string           `json:"deviceID"`
	DateTime      string           `json:"dateTime"`
	Notes         string           `json:"notes"`
	Profile       *string          `json:"profile,omitempty"`
	ConsumerName  *string          `json:"consumerName,omitempty"`
	Items         []SallyMainItems `json:"items"`
}

type SallyMainItems struct {
	Sku        string          `json:"sku"`
	Name       string          `json:"name"`
	Note       *string         `json:"note,omitempty"`
	VoidReason *string         `json:"voidReason,omitempty"`
	Qty        int             `json:"qty"`
	Price      int             `json:"price"`
	Seat       *int            `json:"seat,omitempty"`
	Course     *int            `json:"course,omitempty"`
	SubItems   []SallySubItems `json:"subItems"`
}

type SallySubItems struct {
	Qty   int    `json:"qty"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type menuEntry struct {
	Name          string
	Sku           string
	Price         int
	ProductCenter string
	SubItems      []string
}

var menu = []menuEntry{
	{"Beef Burger", "BRG001", 8500, "Grill", []string{"No Onion", "Extra Cheese", "Well Done"}},
	{"Margherita Pizza", "PZA010", 9500, "Pizza", []string{"Extra Cheese", "Thin Base"}},
	{"Caesar Salad", "SAL003", 6500, "Cold Kitchen", []string{"No Croutons", "Dressing on Side"}},
	{"Fish and Chips", "FSH007", 11000, "Fryer", []string{"Tartare Sauce", "No Vinegar"}},
	{"Chicken Wings", "WNG002", 7500, "Grill", []string{"Hot Sauce", "BBQ Sauce"}},
	{"Coke 300ml", "BEV001", 2500, "Bar", nil},
	{"Cappuccino", "BEV010", 3200, "Bar", nil},
	{"Chocolate Cake", "DES004", 5500, "Dessert", []string{"Extra Cream"}},
}

var sampleWaiters = []string{"Test Waiter 1", "Test Waiter 2", "Sipho M", "Anele K"}
var sampleTables = []string{"Table 1", "Table 2", "Table 5", "Bar 1", "Takeaway"}

func main() {
	host := flag.String("host", "http://localhost:2020", "Base URL of the kds-webserver to inject into, no trailing slash")
	count := flag.Int("count", 1, "Number of orders to send")
	interval := flag.Duration("interval", 500*time.Millisecond, "Delay between orders when sending more than one")
	file := flag.String("file", "", "Path to a JSON file with one order object or an array of order objects to send verbatim (skips random generation)")
	ticketFile := flag.String("ticket-file", "", "Path to a raw ticket/print-feed text file, parsed the same way kds-poller parses a real ticket and sent as one order (skips random generation and -file)")
	screen := flag.String("screen", "Kitchen", "Product center/screen to use for a -ticket-file order when the ticket text doesn't carry its own screen marker")
	table := flag.String("table", "", "Table name for generated orders (random if omitted)")
	waiter := flag.String("waiter", "", "Waiter name for generated orders (random if omitted)")
	itemCount := flag.Int("items", 0, "Number of line items per generated order (0 = random 1-4)")
	seed := flag.Int64("seed", 0, "RNG seed for generated orders (0 = time-based, non-reproducible)")
	ping := flag.Bool("ping", false, "Just check connectivity against <host>/ping and exit")
	verbose := flag.Bool("verbose", false, "Print each request/response body")
	gui := flag.Bool("gui", false, "Launch the browser-based GUI instead of the CLI and ignore the other flags")
	listen := flag.String("listen", "127.0.0.1:8787", "Address the GUI server listens on (used with -gui)")
	flag.Parse()

	if *gui {
		runGUI(*listen)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	if *ping {
		if err := doPing(client, *host, *verbose); err != nil {
			log.Fatalf("ping failed: %v", err)
		}
		fmt.Println("ping OK -", *host)
		return
	}

	if *ticketFile != "" {
		data, err := os.ReadFile(*ticketFile)
		if err != nil {
			log.Fatalf("reading %s: %v", *ticketFile, err)
		}
		order := ticketToSallyOrder(parseTicketText(string(data)), *screen)
		if err := sendOrder(client, *host, order, *verbose); err != nil {
			log.Fatalf("order %s: FAILED: %v", order.Id, err)
		}
		log.Printf("order %s: OK - table %q, waiter %q, %d item(s)", order.Id, order.TableName, order.WaiterName, len(order.Items))
		return
	}

	seedVal := *seed
	if seedVal == 0 {
		seedVal = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seedVal))

	var templates []SallyOrder
	if *file != "" {
		var err error
		templates, err = loadOrdersFromFile(*file)
		if err != nil {
			log.Fatalf("loading %s: %v", *file, err)
		}
		if len(templates) == 0 {
			log.Fatalf("%s contained no orders", *file)
		}
	}

	failures := 0
	for i := 0; i < *count; i++ {
		var order SallyOrder
		if len(templates) > 0 {
			order = templates[i%len(templates)]
			// Repeated sends of the same template would all update one
			// order in place (the server matches on Id) rather than create
			// separate tickets, so give repeats after the first pass a
			// fresh Id.
			if i >= len(templates) {
				order.Id = fmt.Sprintf("%s-%d", order.Id, time.Now().UnixNano())
			}
		} else {
			order = randomOrder(rng, *table, *waiter, *itemCount)
		}

		if err := sendOrder(client, *host, order, *verbose); err != nil {
			log.Printf("order %d/%d (%s): FAILED: %v", i+1, *count, order.Id, err)
			failures++
		} else {
			log.Printf("order %d/%d (%s): OK - table %q, waiter %q, %d item(s)", i+1, *count, order.Id, order.TableName, order.WaiterName, len(order.Items))
		}

		if i < *count-1 {
			time.Sleep(*interval)
		}
	}

	if failures > 0 {
		os.Exit(1)
	}
}

func randomOrder(rng *rand.Rand, table, waiter string, itemCount int) SallyOrder {
	if table == "" {
		table = sampleTables[rng.Intn(len(sampleTables))]
	}
	if waiter == "" {
		waiter = sampleWaiters[rng.Intn(len(sampleWaiters))]
	}
	if itemCount <= 0 {
		itemCount = 1 + rng.Intn(4)
	}

	order := SallyOrder{
		Id:            fmt.Sprintf("TESTORDER-%d", time.Now().UnixNano()),
		TableName:     table,
		WaiterName:    waiter,
		DeviceID:      "kds-order-injector",
		DateTime:      time.Now().Format(time.RFC3339),
		Notes:         "injected by kds-order-injector",
		ProductCenter: "Kitchen",
	}

	for i := 0; i < itemCount; i++ {
		m := menu[rng.Intn(len(menu))]
		item := SallyMainItems{
			Sku:   m.Sku,
			Name:  m.Name,
			Qty:   1 + rng.Intn(3),
			Price: m.Price,
		}
		if len(m.SubItems) > 0 && rng.Intn(2) == 0 {
			sub := m.SubItems[rng.Intn(len(m.SubItems))]
			item.SubItems = []SallySubItems{{Qty: 1, Name: sub}}
		}
		order.Items = append(order.Items, item)
	}

	return order
}

func loadOrdersFromFile(path string) ([]SallyOrder, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOrders(data)
}

// parseOrders accepts either a single order object or an array of order
// objects, so both a hand-written one-order file and a batch export parse
// the same way - and so the GUI's paste box can take either shape too.
func parseOrders(data []byte) ([]SallyOrder, error) {
	var multi []SallyOrder
	if err := json.Unmarshal(data, &multi); err == nil {
		return multi, nil
	}

	var single SallyOrder
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("not a valid order object or array of orders: %w", err)
	}
	return []SallyOrder{single}, nil
}

func sendOrder(client *http.Client, host string, order SallyOrder, verbose bool) error {
	body, err := json.Marshal(order)
	if err != nil {
		return err
	}
	if verbose {
		fmt.Println("-> POST", host+"/order")
		fmt.Println(string(body))
	}

	resp, err := client.Post(host+"/order", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if verbose {
		fmt.Println("<-", resp.Status, string(respBody))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s: %s", resp.Status, string(respBody))
	}
	return nil
}

func doPing(client *http.Client, host string, verbose bool) error {
	resp, err := client.Get(host + "/ping")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if verbose {
		fmt.Println("<-", resp.Status, string(respBody))
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s: %s", resp.Status, string(respBody))
	}
	return nil
}
