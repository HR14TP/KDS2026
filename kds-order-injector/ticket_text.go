// This file lets a raw POS ticket (the same plain-text a real print feed
// produces, and that kds-poller's engine.go/process.go parse into a KDS
// order) be pasted in and converted into an injectable order, instead of
// having to hand-write the Sally JSON shape.
//
// The regexes and the item/sub-item matching heuristics below are ported
// verbatim from kds-poller-main/pkg/process.go and the scanning loop in
// engine.go's ProcessMessage, so a pasted ticket parses the same way it
// would on a real terminal. What's deliberately left out is everything
// engine.go does that only makes sense against an existing DB order:
// matching/merging into an already-open ticket, table transfers, and
// void-item lookups against previously-created items. A pasted ticket is
// always treated as a brand new order; void lines are reported but not
// acted on (there is nothing yet to void).
package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var ticketReferenceReg = regexp.MustCompile("[0-9]{1,2}:[0-9]{2} .* ([a-z]{1}[0-9]*.[0-9]*)")
var ticketStaffReg = regexp.MustCompile(`(ipad|mini)(.*|\s*)/(.*)`)
var ticketScreenReg = regexp.MustCompile(`[*]{1}\s[*]{1}\s(.+)\s[*]{1}\s[*]{1}`)
var ticketProfileReg = regexp.MustCompile(`\s\s[*]{1}\s([^*][a-zA-Z0-9&\s]{1,})\s[*]{1}`)
var ticketCourseReg = regexp.MustCompile(`[*]{2}\s([^*][a-zA-Z0-9\s]{1,})\s[*]{2}`)
var ticketDeliverTimeReg = regexp.MustCompile(`(delivery time:)(.*)`)
var ticketConsumerReg = regexp.MustCompile(`(consumer:)(.*)`)
var ticketTableReg = regexp.MustCompile(`(table.*|order.*|tafel.*|tisch.*)`)
var ticketTableModifiedReg = regexp.MustCompile(`\s\s#(.+)`)
var ticketOrderOnlyReg = regexp.MustCompile(`(order.*)`)
var ticketQtyReg = regexp.MustCompile(`([0-9]{0,}[,.]{0,}[0-9]{1,})\s[x]{1}`)
var ticketEndOfMainReg = regexp.MustCompile(`(-=-)`)
var ticketGenericItemReg = regexp.MustCompile(`(\d+)\s[x](.*)`)
var ticketProdItemReg = regexp.MustCompile(`\d{1,}\s[x]\s(.*)`)
var ticketProductionInstructionsReg = regexp.MustCompile(`(\d+\s{1}[a-z]{2,}.*)`)

type parsedSub struct {
	Name string
	Qty  int
}

type parsedItem struct {
	Name string
	Qty  int
	Subs []parsedSub
}

type parsedTicket struct {
	Table     string
	Waiter    string
	Reference string
	Profile   string
	Screen    string
	Consumer  string
	Delivery  string
	Items     []parsedItem
	// VoidLines lists "N x item name" lines the ticket marked as voided -
	// surfaced for visibility only, since there's no existing order/item
	// for a freshly injected ticket to void against.
	VoidLines []string
}

func ticketMatchLast(content string, reg *regexp.Regexp) string {
	matches := reg.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	if len(last) == 0 {
		return ""
	}
	return strings.TrimSpace(last[len(last)-1])
}

func ticketRemoveNonUTF(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue
			}
		}
		v = append(v, r)
	}
	return string(v)
}

func ticketStringToInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	return i
}

func ticketIsVoidLine(content string) bool {
	return strings.Contains(content, "voided item") || strings.Contains(content, "geannuleerd item")
}

// ticketGetItemMatches ports GetItemsMatching from kds-poller's process.go
// (same byte-offset heuristic for telling a new main item apart from a
// sub-item/modifier of the item already open), adapted to build a
// parsedItem/parsedSub in memory instead of a DB-bound kdsmodels type.
func ticketGetItemMatches(content string, items *[]parsedItem, current *parsedItem) bool {
	matches := ticketGenericItemReg.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return false
	}
	raw := []byte(content)
	if len(raw) <= 4 {
		return false
	}
	if raw[2] == 56 {
		raw[2] = 32
	}

	if current.Qty > 0 && len(current.Subs) > 0 {
		repeatName := ticketRemoveNonUTF(ticketMatchLast(content, ticketGenericItemReg))
		if strings.EqualFold(strings.TrimSpace(repeatName), strings.TrimSpace(current.Name)) {
			return true
		}
	}

	opensNewItem := current.Qty == 0 || raw[0] != 32 ||
		(raw[2] == 32 && raw[3] != 32) ||
		(raw[2] == 56 && raw[3] != 32) ||
		(raw[1] == 97 && raw[2] == 0 && raw[3] == 32)

	qtytemp := strings.Replace(ticketMatchLast(strings.TrimSpace(content), ticketQtyReg), ",", ".", 1)
	qty := ticketStringToInt(qtytemp)
	name := ticketRemoveNonUTF(ticketMatchLast(content, ticketGenericItemReg))

	if opensNewItem {
		if current.Qty > 0 {
			*items = append(*items, *current)
		}
		*current = parsedItem{Name: name, Qty: qty}
		return true
	}

	current.Subs = append(current.Subs, parsedSub{Name: name, Qty: qty})
	return true
}

func ticketGetSubItemMatches(content string, current *parsedItem) bool {
	matches := ticketProdItemReg.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return false
	}
	qtytemp := strings.Replace(ticketMatchLast(content, ticketQtyReg), ",", ".", 1)
	qty := ticketStringToInt(qtytemp)
	name := ticketMatchLast(content, ticketProdItemReg)
	if len(name) > 1 {
		current.Subs = append(current.Subs, parsedSub{Name: name, Qty: qty})
	}
	return true
}

func ticketGetProductionInstructions(content string, current *parsedItem) bool {
	matches := ticketProductionInstructionsReg.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return false
	}
	name := ticketMatchLast(content, ticketProductionInstructionsReg)
	if len(name) > 1 {
		if len(current.Subs) > 0 {
			last := &current.Subs[len(current.Subs)-1]
			last.Name = last.Name + " - " + name
		} else {
			current.Subs = append(current.Subs, parsedSub{Name: "instruction: " + name, Qty: 1})
		}
	}
	return true
}

// parseTicketText runs the same line-by-line scan kds-poller's engine.go
// runs against a real print feed, and returns what it found. Field
// detection (table/waiter/reference/profile/screen/consumer/delivery) each
// latch onto their first match per ticket, same as the original.
func parseTicketText(content string) parsedTicket {
	var t parsedTicket
	var current parsedItem
	var itemSection bool
	var nextLineIsOrder bool
	var foundTable, foundOrder bool
	var foundProfile, foundScreen, foundStaff, foundReference, foundConsumer, foundDelivery bool
	var voidNext bool

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		text := strings.ToLower(scanner.Text())

		if course := ticketMatchLast(text, ticketCourseReg); course != "" {
			if current.Qty > 0 {
				t.Items = append(t.Items, current)
				current = parsedItem{}
			}
			t.Items = append(t.Items, parsedItem{Name: "COURSE : " + course, Qty: 1})
		}

		if nextLineIsOrder {
			table := strings.ReplaceAll(text, ")", "")
			table = strings.ReplaceAll(table, "(", "")
			table = strings.TrimSpace(table)
			if matched := ticketMatchLast(table, ticketTableReg); matched != "" {
				table = matched
			}
			t.Table = table
			nextLineIsOrder = false
			foundTable = true
		}

		if !foundDelivery {
			if delivery := ticketMatchLast(text, ticketDeliverTimeReg); delivery != "" {
				foundDelivery = true
				t.Delivery = delivery
			}
		}

		if !foundProfile {
			if profile := ticketMatchLast(text, ticketProfileReg); profile != "" {
				foundProfile = true
				t.Profile = profile
				nextLineIsOrder = true
			}
		}
		if !foundScreen {
			if screen := ticketMatchLast(text, ticketScreenReg); screen != "" {
				foundScreen = true
				t.Screen = screen
			}
		}
		if !foundStaff {
			if staff := ticketMatchLast(text, ticketStaffReg); staff != "" {
				foundStaff = true
				t.Waiter = staff
			}
		}
		if !foundReference {
			if reference := ticketMatchLast(text, ticketReferenceReg); reference != "" {
				foundReference = true
				t.Reference = reference
			}
		}

		if !foundTable {
			if table := ticketMatchLast(text, ticketTableReg); table != "" {
				foundTable = true
				if t.Table != table {
					t.Table += table
				}
			} else if table2 := ticketMatchLast(text, ticketTableModifiedReg); table2 != "" {
				foundTable = true
				if t.Table != table2 {
					t.Table += table2
				}
			}
		}
		if !foundOrder && !foundTable {
			if table := ticketMatchLast(text, ticketOrderOnlyReg); table != "" {
				foundOrder = true
				t.Table = table
			}
		}

		// A "voided item" marker line is followed by the actual "N x name"
		// line on the ticket - the marker line itself carries no item info,
		// so the lookup for what was voided has to happen one line later
		// (mirrors obj.Voiditem in engine.go's ProcessMessage).
		if voidNext {
			voidNext = false
			if voided := ticketMatchLast(text, ticketGenericItemReg); voided != "" {
				t.VoidLines = append(t.VoidLines, strings.TrimSpace(text))
			}
		} else {
			voidNext = ticketIsVoidLine(text)
			if !voidNext {
				if ticketGetItemMatches(text, &t.Items, &current) {
					itemSection = true
				} else if itemSection {
					if ticketGetSubItemMatches(text, &current) {
						// handled
					} else if ticketGetProductionInstructions(text, &current) {
						// handled
					} else if ticketEndOfMainReg.MatchString(text) {
						t.Items = append(t.Items, current)
						current = parsedItem{}
						itemSection = false
					}
				}
			}
		}

		if !foundConsumer {
			if consumer := ticketMatchLast(text, ticketConsumerReg); consumer != "" {
				foundConsumer = true
				t.Consumer = consumer
			}
		}
	}

	if current.Qty > 0 {
		t.Items = append(t.Items, current)
	}

	return t
}

// ticketToSallyOrder maps a parsed ticket onto the same SallyOrder shape
// the rest of this tool sends to kds-webserver's /order endpoint. All of a
// Sally order's items share one screen (ProductCenter) - unlike a real
// ticket, which can carry a screen marker per section - so defaultScreen is
// used whenever the ticket didn't carry its own "* * SCREEN * *" marker.
func ticketToSallyOrder(t parsedTicket, defaultScreen string) SallyOrder {
	screen := t.Screen
	if screen == "" {
		screen = defaultScreen
	}

	id := t.Reference
	if id == "" {
		id = fmt.Sprintf("TICKET-%d", time.Now().UnixNano())
	}

	order := SallyOrder{
		Id:            id,
		ProductCenter: screen,
		TableName:     t.Table,
		WaiterName:    t.Waiter,
		DeviceID:      "kds-order-injector",
		DateTime:      time.Now().Format(time.RFC3339),
		Notes:         "injected from raw ticket text",
	}
	if t.Profile != "" {
		p := t.Profile
		order.Profile = &p
	}
	if t.Consumer != "" {
		c := t.Consumer
		order.ConsumerName = &c
	}

	for _, item := range t.Items {
		if item.Name == "" {
			continue
		}
		mi := SallyMainItems{Name: item.Name, Qty: item.Qty}
		for _, sub := range item.Subs {
			mi.SubItems = append(mi.SubItems, SallySubItems{Name: sub.Name, Qty: sub.Qty})
		}
		order.Items = append(order.Items, mi)
	}

	return order
}
