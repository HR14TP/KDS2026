package main

import (
	"regexp"
	"strings"

	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

//var referenceReg = regexp.MustCompile("[0-9]{1}.[0-9]{1,2}.[0-9]{1,2} [0-9]{1,2}:[0-9]{2} [a-z]{2}(.*)")
//var referenceReg = regexp.MustCompile("[0-9]{1}.[0-9]{1,2}.[0-9]{1,2} [0-9]{1,2}:[0-9]{2} .* ([a-z]{1}[0-9]*.[0-9]*)")

var referenceReg = regexp.MustCompile("[0-9]{1,2}:[0-9]{2} .* ([a-z]{1}[0-9]*.[0-9]*)")
var staffReg = regexp.MustCompile(`(ipad|mini)(.*|\s*)/(.*)`)

var screenReg = regexp.MustCompile(`[*]{1}\s[*]{1}\s(.+)\s[*]{1}\s[*]{1}`)

//var screenReg = regexp.MustCompile(`[*]{1}\s[*]{1}\s(.+)\s[*]{1}\s[*]{1}`)

var profileReg = regexp.MustCompile(`\s\s[*]{1}\s([^*][a-zA-Z0-9&\s]{1,})\s[*]{1}`)

var courseReg = regexp.MustCompile(`[*]{2}\s([^*][a-zA-Z0-9\s]{1,})\s[*]{2}`)

//var profileReg = regexp.MustCompile(`\*\s([^*]*)\*`)

var collectionReg = regexp.MustCompile("(collection code)")
var pagernrReg = regexp.MustCompile("pagernr\\S\\S\\S[0-9]{1,}")
var getNumber = regexp.MustCompile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
var getTransfer = regexp.MustCompile(`(transferred to)(.*)`)
var getTransferNL = regexp.MustCompile(`(verplaatst naar)(.*)`)

//screen : [*]{1}\s[*]{1}\s(.+)\s[*]{1}\s[*]{1}
//profile : \s\s[*]{1}\s([^*][a-z0-9\s]{1,})\s[*]{1}
//course : [*]{2}\s([^*][a-z0-9\s]{1,})\s[*]{2}

var deliverTime = regexp.MustCompile(`(delivery time:)(.*)`)
var consumerName = regexp.MustCompile(`(consumer:)(.*)`)

var tableReg = regexp.MustCompile(`(table.*|order.*|tafel.*|tisch.*)`)
var tableOnlyReg = regexp.MustCompile("(table.[0-9]{1,}|tafel.[0-9]{1,}|tisch.[0-9]{1,})")
var orderOnlyReg = regexp.MustCompile("(order.*)")
var tableModified = regexp.MustCompile(`\s\s#(.+)`)

var mainitemReg = regexp.MustCompile(`\s\s\s(\d+)\s(\s|[x])(.*)`)

// var subitemReg = regexp.MustCompile(`^\s\s(\d)\s(\s|[x])(.*)`)
var proditemReg = regexp.MustCompile(`\d{1,}\s[x]\s(.*)`)

//var qtyReg = regexp.MustCompile(`^(\d+)`)

var qtyReg = regexp.MustCompile(`([0-9]{0,}[,.]{0,}[0-9]{1,})\s[x]{1}`)

var endofmainReg = regexp.MustCompile(`(-=-)`)
var genericItemReg = regexp.MustCompile(`(\d+)\s[x](.*)`)
var productionInstructions = regexp.MustCompile(`(\d+\s{1}[a-z]{2,}.*)`)

var customerCell = regexp.MustCompile(`^[+][0-9]{5,}`)
var customerCellUnsafe = regexp.MustCompile(`^[0-9]{6,}`)

func IsThisTransfer(content string) (bool, string) {
	transfer := GetMatchAndReturn(content, getTransfer)
	if transfer == "" {
		transfer = GetMatchAndReturn(content, getTransferNL)
	}
	if transfer != "" {
		ShowDebug("TRANSFER : " + transfer)
		table := strings.ReplaceAll(transfer, ")", "")
		table = strings.ReplaceAll(table, "(", "")
		table = strings.ReplaceAll(table, "transferred to", "")
		table = strings.ReplaceAll(table, "verplaatst naar", "")
		table = strings.TrimSpace(table)
		return true, table
	}
	return false, ""
}

// GetItemMatches ... GetItemMatches
func GetItemMatches(content string, reg *regexp.Regexp, obj *kdsmodels.KDSPollObject) bool {
	/* need to check if there is already a main item set first */
	ShowDebug("GetItemMatches")
	if obj.Currentitem.Qty > 0 {
		obj.Items = append(obj.Items, obj.Currentitem)
		obj.Currentitem = kdsmodels.KDSMainItems{}
	}
	matches := reg.FindAllStringSubmatch(content, -1)
	ShowDebugInt(matches)
	if len(matches) > 0 {
		qtytemp := GetMatchAndReturn(content, qtyReg)
		//spew.Dump(qtytemp)
		qtytemp = strings.Replace(qtytemp, ",", ".", 1)
		qty := UtilStringToInt(qtytemp)
		//spew.Dump(qty)
		name := GetMatchAndReturn(content, mainitemReg)
		obj.Currentitem.Name = name
		obj.Currentitem.Qty = qty
		obj.Currentitem.QtyRaw = helper.UtilStringToFloat64(qtytemp)
		ShowDebugInt(obj.Currentitem)
		return true
	}
	return false
}

// GetSubItemMatches ... GetSubItemMatches
func GetSubItemMatches(content string, reg *regexp.Regexp, obj *kdsmodels.KDSPollObject) bool {
	matches := reg.FindAllStringSubmatch(content, -1)
	if len(matches) > 0 {
		qtytemp := GetMatchAndReturn(content, qtyReg)
		qtytemp = strings.Replace(qtytemp, ",", ".", 1)
		qty := UtilStringToInt(qtytemp)
		//qty := UtilStringToInt(GetMatchAndReturn(content, qtyReg))
		name := GetMatchAndReturn(content, reg)
		if len(name) > 1 {
			obj.Currentitem.Subs = append(obj.Currentitem.Subs, kdsmodels.KDSSubMainItems{Name: name, Qty: qty, QtyRaw: helper.UtilStringToFloat64(qtytemp)})
		}
		return true
	}
	return false
}

func GetProductionInstructions(content string, reg *regexp.Regexp, obj *kdsmodels.KDSPollObject) bool {
	matches := reg.FindAllStringSubmatch(content, -1)
	if len(matches) > 0 {
		qtytemp := GetMatchAndReturn(content, qtyReg)
		qtytemp = strings.Replace(qtytemp, ",", ".", 1)
		qty := UtilStringToInt(qtytemp)
		name := GetMatchAndReturn(content, reg)
		if len(name) > 1 {
			// Fold onto the sub-item this instruction actually modifies
			// (e.g. "one egg, your way" -> "one egg, your way - hard
			// poached") instead of appending a same-level flat sibling -
			// the data model has no third nesting level to represent
			// "modifier of a modifier", and a flat sibling becomes
			// ambiguous the moment a main item legitimately has more than
			// one of the sub-item this was meant to qualify (e.g. two "one
			// egg, your way" subs on one order, one hard-poached and one
			// medium-poached).
			if len(obj.Currentitem.Subs) > 0 {
				last := &obj.Currentitem.Subs[len(obj.Currentitem.Subs)-1]
				last.Name = last.Name + " - " + name
			} else {
				obj.Currentitem.Subs = append(obj.Currentitem.Subs, kdsmodels.KDSSubMainItems{Name: "instruction: " + name, Qty: qty, QtyRaw: helper.UtilStringToFloat64(qtytemp)})
			}
		}
		return true
	}
	return false
}

// GetMatchAndReturn ... GetMatchAndReturn
func GetMatchAndReturn(content string, reg *regexp.Regexp) string {
	matches := reg.FindAllStringSubmatch(content, -1)
	return GetLastMatch(matches)
}

// GetLastMatch ... GetLastMatch
func GetLastMatch(matches [][]string) string {
	if len(matches) > 0 {
		if len(matches[len(matches)-1]) > 0 {
			profile := (matches[len(matches)-1][len(matches[len(matches)-1])-1])
			profile = strings.TrimSpace(profile)
			return profile
		}
		return ""
	}
	return ""
}

// GetFirstMatch ... GetFirstMatch
func GetFirstMatch(matches [][]string) string {
	if len(matches) > 0 {
		if len(matches[0]) > 0 {
			profile := matches[0][0]
			profile = strings.TrimSpace(profile)
			return profile
		}
		return ""
	}
	return ""
}

// GetInts ... GetInts
func GetInts(content string) string {
	matches := GetMatchAndReturn(content, getNumber)
	return matches
}

// GetProfile ... GetProfile
func GetProfile(content string) string {
	rexProfile := regexp.MustCompile("\\*\\s\\*(.+)\\*\\s\\*")
	matches := rexProfile.FindAllStringSubmatch(content, -1)
	return GetLastMatch(matches)
}

// GetStaff ... GetStaff
func GetStaff(content string) string {
	rexStaff := regexp.MustCompile("(ipad|mini)(\\d*|\\s*)/(.*)")
	matches := rexStaff.FindAllStringSubmatch(content, -1)
	return GetLastMatch(matches)
}

// GetItemsMatching ... GetItemsMatching
func GetItemsMatching(content string, obj *kdsmodels.KDSPollObject) bool {
	matches := genericItemReg.FindAllStringSubmatch(content, -1)
	if len(matches) > 0 {
		ShowDebug("GetItemsMatching")
		aa := []byte(content)
		ShowDebugInt(matches)
		if len(aa) > 4 {
			if aa[2] == 56 {
				aa[2] = 32
			}
			// Some POS ticket templates for a build-your-own dish (e.g.
			// "Breakfast Your Way") reprint the dish's own name as a
			// separator line before every selected option/modifier, at the
			// same indentation as the options themselves. The byte-offset
			// heuristic below has no way to tell that apart from a
			// genuinely new dish, and used to open a brand new main item
			// (or add a junk sub-item) for every repeat - splitting what's
			// really one order into several stray tiles, and leaving the
			// real options scattered across them or lost entirely. A
			// repeat of the name of the item that's already open carries no
			// new information either way, so skip it outright and keep
			// accumulating the following lines as sub-items of the item
			// that's already open. Only collapses once at least one real
			// option/modifier has already been recorded under this item
			// (len(Subs) &gt; 0) - two covers genuinely ordering the same
			// plain, uncustomized dish back-to-back, with nothing between
			// the two ticket lines, must still become two separate tiles,
			// not one.
			// Critically, this must ALSO require the repeated line to be
			// indented like a sub-item (the same test the byte-offset block
			// below uses to open a new main item) - a customer genuinely
			// ordering the same dish twice back-to-back (e.g. two "Hash Benny
			// Stack" with different poached-egg styles) reprints the name
			// flush-left, exactly like any other new main item, with nothing
			// to tell it apart from the build-your-own separator by name
			// alone. Without this guard, that second, flush-left order was
			// silently dropped and its own modifier line got folded onto the
			// first item instead of becoming its own item - the board then
			// showed only one "1 x" tile for what was actually two, with
			// both modifiers (e.g. "medium poached" and "hard poached")
			// stacked onto the single surviving item.
			if obj.Currentitem.Qty > 0 && len(obj.Currentitem.Subs) > 0 {
				looksLikeSubItemLine := aa[0] == 32 && !((aa[2] == 32 && aa[3] != 32) || (aa[1] == 97 && aa[2] == 0 && aa[3] == 32))
				if looksLikeSubItemLine {
					repeatName := RemoveNonUTF(GetMatchAndReturn(content, genericItemReg))
					if strings.EqualFold(strings.TrimSpace(repeatName), strings.TrimSpace(obj.Currentitem.Name)) {
						return true
					}
				}
			}
			// obj.Currentitem.Qty == 0 means nothing is open yet, so this
			// line cannot be a sub-item of anything - it must be the item
			// that opens the section, regardless of what the byte-offset
			// checks below say. Without this, a ticket whose very first
			// item-shaped line happens to be indented (failing every check
			// below) accumulated as orphaned sub-items of an unnamed,
			// zero-qty item until some later line finally tripped one of
			// the byte-offset conditions - at which point those orphaned
			// lines got silently adopted as subs of whatever happened to
			// open it, rather than being that main item's own identity
			// from the start.
			if obj.Currentitem.Qty == 0 || aa[0] != 32 || (aa[2] == 32 && aa[3] != 32) || (aa[2] == 56 && aa[3] != 32) || (aa[1] == 97 && aa[2] == 0 && aa[3] == 32) {
				ShowDebug(content)
				if obj.Currentitem.Qty > 0 {
					obj.Items = append(obj.Items, obj.Currentitem)
					obj.Currentitem = kdsmodels.KDSMainItems{}
				}
				qtytemp := GetMatchAndReturn(strings.TrimSpace(content), qtyReg)
				qtytemp = strings.Replace(qtytemp, ",", ".", 1)
				qtyraw := helper.UtilStringToFloat64(qtytemp)
				qty := UtilStringToInt(qtytemp)
				//qty := UtilStringToInt(GetMatchAndReturn(strings.TrimSpace(content), qtyReg))
				ShowDebug("RESOLVED QTY AS : ")
				ShowDebugInt(qty)
				name := GetMatchAndReturn(content, genericItemReg)
				obj.Currentitem.Name = RemoveNonUTF(name)
				obj.Currentitem.Qty = qty
				obj.Currentitem.QtyRaw = qtyraw
				//spew.Dump(obj.Currentitem)
				obj.Itemsection = true
				ShowDebugInt(obj.Currentitem)
				return true
			}

			qtytemp := GetMatchAndReturn(strings.TrimSpace(content), qtyReg)

			qtytemp = strings.Replace(qtytemp, ",", ".", 1)
			//fmt.Println("Current Temp")
			//fmt.Println(qtytemp)
			//fmt.Println(helper.UtilStringToFloat64(qtytemp))
			qty := UtilStringToInt(qtytemp)
			//qty := UtilStringToInt(GetMatchAndReturn(strings.TrimSpace(content), qtyReg))
			name := RemoveNonUTF(GetMatchAndReturn(content, genericItemReg))
			obj.Currentitem.Subs = append(obj.Currentitem.Subs, kdsmodels.KDSSubMainItems{Name: name, Qty: qty, QtyRaw: helper.UtilStringToFloat64(qtytemp)})
			return true
		}
	}
	return false
}

func getVoidItemsMatching(content string, obj *kdsmodels.KDSPollObject) bool {
	matches := genericItemReg.FindAllStringSubmatch(content, -1)
	if len(matches) > 0 {
		aa := []byte(content)
		if len(aa) > 4 {
			if aa[0] != 32 || (aa[2] == 32 && aa[3] != 32) {
				qty := UtilStringToInt(GetMatchAndReturn(strings.TrimSpace(content), qtyReg))
				name := GetMatchAndReturn(content, genericItemReg)
				tempitem := kdsmodels.KDSMainItems{}
				tempitem.Name = RemoveNonUTF(name)
				tempitem.Qty = qty
				obj.Voiditems = append(obj.Voiditems, tempitem)
				return true
			}

			qty := UtilStringToInt(GetMatchAndReturn(strings.TrimSpace(content), qtyReg))
			name := GetMatchAndReturn(content, genericItemReg)
			//currentitem.Subs = append(currentitem.Subs, models.SubMainItems{Name: name, Qty: qty})
			tempitem := kdsmodels.KDSMainItems{}
			tempitem.Name = RemoveNonUTF(name)
			tempitem.Qty = qty
			obj.Voiditems = append(obj.Voiditems, tempitem)
			return true
		}
	}
	return false
}

func getIsThisVoid(content string) bool {
	if strings.Contains(content, "voided item") || strings.Contains(content, "geannuleerd item") {
		ShowDebug("FOUND VOIDED ITEM")
	}
	return strings.Contains(content, "voided item") || strings.Contains(content, "geannuleerd item")
}

func getIsThisCollectionCode(content string) bool {
	if strings.Contains(content, "collection code") {
		ShowDebug("Collection Code")
	}
	return strings.Contains(content, "collection code")
}

func endOfItem(content string, reg *regexp.Regexp, obj *kdsmodels.KDSPollObject) bool {
	matches := reg.FindAllStringSubmatch(content, -1)
	if len(matches) > 0 {
		obj.Items = append(obj.Items, obj.Currentitem)
		obj.Currentitem = kdsmodels.KDSMainItems{}
		obj.Itemsection = false
		return true
	}
	return false
}
