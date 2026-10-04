package kdsmodels

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type ForToday struct {
	Counter int `gorm:"column:counter;" json:"counter"`
}
type ItemSubOrder struct {
	Counter     int `gorm:"column:counter;" json:"counter"`
	Numoforders int `gorm:"column:numoforders;" json:"numoforders"`
}
type PrepTime struct {
	Avgmins int `gorm:"column:avgmins;" json:"avgmins"`
	Avgsecs int `gorm:"column:avgsecs;" json:"avgsecs"`
}
type KDSRegisterUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type KDSVersionCheckInput struct {
	Name    string `json:"name"`
	Current string `json:"current"`
}
type KDSVersionCheckOutput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Asar    string `json:"asar"`
	Sha1    string `json:"sha1"`
	Info    string `json:"info"`
}

type KDSApplicationMySQL struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	Name      string    `gorm:"column:name" json:"name"`
	Version   string    `gorm:"column:version" json:"version"`
	AppDomain string    `gorm:"column:appdomain" json:"appdomain"`
	Info      string    `gorm:"column:info" json:"info"`
	URL       string    `gorm:"column:url" json:"url"`
	Sha1      string    `gorm:"column:sha1" json:"sha1"`
	CreatedAt time.Time `gorm:"<-:false;column:created_at" json:"created"`
	UpdatedAt time.Time `gorm:"<-:false;column:updated_at" json:"updated"`
}

func (p *KDSApplicationMySQL) TableName() string {
	return "applications"
}

type KDSRegisterTerminal struct {
	ConnectionCode string `json:"connection_code"`
	HWID           string `json:"hwid"`
}

type KDSConfigNew struct {
	Bumpbars struct {
		Enabled       bool `json:"enabled"`
		Keycloseorder struct {
			Key     string `json:"key"`
			Shift   bool   `json:"shift"`
			Control bool   `json:"control"`
			Alt     bool   `json:"alt"`
		} `json:"keycloseorder"`
		Keyrecallorder struct {
			Key     string `json:"key"`
			Shift   bool   `json:"shift"`
			Control bool   `json:"control"`
			Alt     bool   `json:"alt"`
		} `json:"keyrecallorder"`
		Keycancelaction struct {
			Key     string `json:"key"`
			Shift   bool   `json:"shift"`
			Control bool   `json:"control"`
			Alt     bool   `json:"alt"`
		} `json:"keycancelaction"`
	} `json:"bumpbars"`

	Client struct {
		Name     string `json:"name"`
		Screen   string `json:"screen"`
		Terminal string `json:"terminal"`
		Serial   string `json:"serial"`
	} `json:"client"`

	Duplication struct {
		Enableduplication     bool   `json:"enableduplication"`
		Enable2wayduplication bool   `json:"enable2wayduplication"`
		Screen1               string `json:"screen1"`
		Screen2               string `json:"screen2"`
		Screen3               string `json:"screen3"`
		Enabledumbterminal    bool   `json:"enabledumbterminal"`
		Dumbscreen            string `json:"dumbscreen"`
	} `json:"duplication"`

	Flags struct {
		Compressed            bool `json:"compressed"`
		Showheadings          bool `json:"showheadings"`
		Leftalign             bool `json:"leftalign"`
		Checkboxes            bool `json:"checkboxes"`
		Sortsubitems          bool `json:"sortsubitems"`
		Primaryfocus          bool `json:"primaryfocus"`
		Enableallcenters      bool `json:"enableallcenters"`
		Allowmultiprogram     bool `json:"allowmultiprogram"`
		Compressedsub         bool `json:"compressedsub"`
		Hidesub1noncompressed bool `json:"hidesub1noncompressed"`
		Hidesub1compressed    bool `json:"hidesub1compressed"`
		Checkboxesautoclose   bool `json:"checkboxesautoclose"`
		Neworderprio          bool `json:"neworderprio"`
	} `json:"flags"`

	Integrations struct {
		Enablesendprinter       bool   `json:"enablesendprinter"`
		Sendprinterip           string `json:"sendprinterip"`
		SendprinterDelay        int    `json:"sendprinter_delay"`
		Enabletwiliointegration bool   `json:"enabletwiliointegration"`
		TwilioDelay             int    `json:"twilio_delay"`
		TwilioAccountsid        string `json:"twilio_accountsid"`
		TwilioAuthtoken         string `json:"twilio_authtoken"`
		TwilioSendingnumber     string `json:"twilio_sendingnumber"`
		TwilioCreateSend        bool   `json:"twilio_create_send"`
		TwilioCreateNessage     string `json:"twilio_create_message"`
		TwilioCompleteSend      bool   `json:"twilio_complete_send"`
		TwilioCompleteMessage   string `json:"twilio_complete_message"`
	} `json:"integrations"`

	Timers struct {
		Late1color           string `json:"late1color"`
		Late2color           string `json:"late2color"`
		Late1timer           string `json:"late1timer"`
		Late2timer           string `json:"late2timer"`
		Enablereversecounter bool   `json:"enablereversecounter"`
		Reversecounter       int    `json:"reversecounter"`
	} `json:"timers"`
}

type KDSConfig struct {
	Server struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Location string `json:"location"`
		Serial   string `json:"serial"`
	} `json:"server"`
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		Schema   string `json:"schema"`
	} `json:"database"`
	Webserver struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"webserver"`
	Timers struct {
		Drinks struct {
			Time  []int    `json:"time"`
			Color []string `json:"color"`
		} `json:"drinks"`
		Grill struct {
			Time  []int    `json:"time"`
			Color []string `json:"color"`
		} `json:"grill"`
	} `json:"timers"`
	Client struct {
		Name     string `json:"name"`
		Css      string `json:"css"`
		Screen   string `json:"screen"`
		Terminal string `json:"terminal"`
		Serial   string `json:"serial"`
		// Passterminal names the pass terminal (its own Client.Screen value)
		// responsible for bumping THIS screen's orders - empty means this
		// screen bumps its own orders itself. Set via this terminal's own
		// config UI, then broadcast to every other terminal on boot (see
		// announceOwnScreenPassAssignment in kds-webserver-main) so the
		// named pass terminal's local kds_screens copy learns the
		// assignment too.
		Passterminal string `json:"passterminal"`
	} `json:"client"`
	// This struct silently fell out of sync with what the frontend (Client
	// Config, kds.js) actually reads/writes/saves to this same JSON file -
	// encoding/json drops any field on load that isn't declared here, and
	// then obviously can't re-marshal a field it never loaded either, so
	// every flag added to the frontend after this struct was last updated
	// (Showoldestfirst chief among them - see kds-webserver-main's
	// GetconfigHandler/LoadConfiguration, the only path the frontend's
	// config actually comes from) was saved to disk correctly by Client
	// Config, then silently never reached the board at all: KDS.flagcheck
	// always saw it as absent and fell back to false, regardless of what
	// was actually saved or shown as toggled on in the config UI. Every
	// flag below is one actually referenced in kds.js/frontend.js as of
	// this fix - keep this in sync when adding a new one.
	Flags struct {
		Compressed         bool `json:"compressed"`
		Showheadings       bool `json:"showheadings"`
		Leftalign          bool `json:"leftalign"`
		Checkboxes         bool `json:"checkboxes"`
		Sortsubitems       bool `json:"sortsubitems"`
		Primaryfocus       bool `json:"primaryfocus"`
		Enableallcenters   bool `json:"enableallcenters"`
		Allowmultiprogram  bool `json:"allowmultiprogram"`
		EnableGDPR         bool `json:"enablegdpr"`
		Passmode           bool `json:"passmode"`
		Odsmode            bool `json:"odsmode"`
		Fontsize           int  `json:"fontsize"`
		Productionmode     bool `json:"productionmode"`
		Touchui            bool `json:"touchui"`
		Sortbyprodstate    bool `json:"sortbyprodstate"`
		Summarybystatus    bool `json:"summarybystatus"`
		Enablemulticolumn  bool `json:"enablemulticolumn"`
		Standalonemode     bool `json:"standalonemode"`
		Showoldestfirst    bool `json:"showoldestfirst"`
		Enabletilecollapse bool `json:"enabletilecollapse"`
		Fixedtilesize      bool `json:"fixedtilesize"`
		Fixedtilerows      int  `json:"fixedtilerows"`
	} `json:"flags"`
	Colors struct {
		Mainitem         string `json:"mainitem"`
		Subitem          string `json:"subitem"`
		Late1            string `json:"late1"`
		Late2            string `json:"late2"`
		Late1time        string `json:"late1time"`
		Late2time        string `json:"late2time"`
		Reversecount     bool   `json:"reversecount"`
		Reversecounttime int    `json:"reversecounttime"`
	} `json:"colors"`
	Text struct {
		Consumer string `json:"consumer"`
	} `json:"text"`
	Duplicate struct {
		Enableduplication bool     `json:"enableduplication"`
		Screen1           string   `json:"screen1"`
		Screen2           string   `json:"screen2"`
		Screen3           string   `json:"screen3"`
		Remote            string   `json:"remote"`
		Peers             []string `json:"peers"`
	} `json:"duplicate"`
	Oncomplete struct {
		Enablesendprinter bool   `json:"enablesendprinter"`
		Sendprinterip     string `json:"sendprinterip"`
	} `json:"oncomplete"`
	Bumpbars struct {
		Enabled         bool `json:"enabled"`
		KEY_CLOSE_ORDER struct {
			KEY     string `json:"KEY"`
			SHIFT   bool   `json:"SHIFT"`
			CONTROL bool   `json:"CONTROL"`
			ALT     bool   `json:"ALT"`
		} `json:"KEY_CLOSE_ORDER"`
		KEY_RECALL_ORDER struct {
			KEY     string `json:"KEY"`
			SHIFT   bool   `json:"SHIFT"`
			CONTROL bool   `json:"CONTROL"`
			ALT     bool   `json:"ALT"`
		} `json:"KEY_RECALL_ORDER"`
		KEY_CANCEL_ACTION struct {
			KEY     string `json:"KEY"`
			SHIFT   bool   `json:"SHIFT"`
			CONTROL bool   `json:"CONTROL"`
			ALT     bool   `json:"ALT"`
		} `json:"KEY_CANCEL_ACTION"`
		Mapping struct {
			Enabled         bool `json:"enabled"`
			KEY_CLOSE_ORDER struct {
				KEY     string `json:"KEY"`
				SHIFT   bool   `json:"SHIFT"`
				CONTROL bool   `json:"CONTROL"`
				ALT     bool   `json:"ALT"`
			} `json:"KEY_CLOSE_ORDER"`
			KEY_RECALL_ORDER struct {
				KEY     string `json:"KEY"`
				SHIFT   bool   `json:"SHIFT"`
				CONTROL bool   `json:"CONTROL"`
				ALT     bool   `json:"ALT"`
			} `json:"KEY_RECALL_ORDER"`
			KEY_CANCEL_ACTION struct {
				KEY     string `json:"KEY"`
				SHIFT   bool   `json:"SHIFT"`
				CONTROL bool   `json:"CONTROL"`
				ALT     bool   `json:"ALT"`
			} `json:"KEY_CANCEL_ACTION"`
		} `json:"mapping"`
	} `json:"bumpbars"`
	Kdsfeatures struct {
		Prioritykeywordenabled bool   `json:"prioritykeywordenabled"`
		Prioritykeywordtext    string `json:"prioritykeywordtext"`
		Enablecourses          bool   `json:"enablecourses"`
	} `json:"kdsfeatures"`
}

type KDSMainItems struct {
	Qty    int
	Name   string
	QtyRaw float64
	Subs   []KDSSubMainItems
}
type KDSSubMainItems struct {
	Qty    int
	QtyRaw float64
	Name   string
}

type KDSPollObject struct {
	Items                    []KDSMainItems
	Voiditems                []KDSMainItems
	Currentitem              KDSMainItems
	Config                   KDSConfig
	Neworder                 KDSOrders
	Screenkds                KDSScreens
	Itemsection              bool
	Cancontinue              bool
	Foundprofile             bool
	Foundstaff               bool
	Foundreference           bool
	Foundscreen              bool
	Foundtable               bool
	Voiditem                 bool
	Foundpagernr             bool
	Nextlineisorder          bool
	Nextlineiscollectioncode bool
	Itemadded                bool
	Foundconsumer            bool
	Founddelivery            bool
	Foundcell                bool
}
type KDSClientConfig struct {
	Config  KDSConfig    `json:"config"`
	Screens []KDSScreens `json:"screens"`
}
type KDSWebInstruction struct {
	InstructionType string `json:"type"`
	Screen          string `json:"screen"`
	OrderID         string `json:"order_id"`
	PrepMins        int    `json:"prepmins"`
	PrepSecs        int    `json:"prepsecs"`
}
type KDSWSPacket struct {
	Data        interface{} `json:"data"`
	MessageType string      `json:"messagetype"`
}
type KDSEnv struct {
	Config *KDSConfig
	DB     *gorm.DB
}

type KDSChannelMessage struct {
	Topic   string             `json:"topic"`
	Message KDSTerminalMessage `json:"message"`
}

type KDSTerminalMessage struct {
	TerminalID int    `json:"terminal_id"`
	HWID       string `json:"hwid"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	// Screen and Passmode identify which terminal sent this message - used
	// by sub-order-close so a close-order request can be scoped to the
	// sending terminal's own screen (or, for a pass terminal, to every
	// screen assigned to it) instead of always closing the whole order.
	// Zero-value (Screen: "", Passmode: false) for every other message type
	// on this struct, which none of them read.
	Screen   string `json:"screen"`
	Passmode bool   `json:"passmode"`
}

// KDSOrderScreenClose is the peer-sync payload for a screen-scoped order
// close (see closeOrderScreenDatabase/closeOrderForPassTerminal in
// kds-webserver-main) - carries just enough for a receiving terminal to
// find its own copy of the order (by Reference, since each terminal has its
// own auto-increment ids) and re-apply the same scoped completion locally.
type KDSOrderScreenClose struct {
	Reference string `json:"reference"`
	Screen    string `json:"screen"`
	Passmode  bool   `json:"passmode"`
}

// KDSOrderScreenRecall is the peer-sync payload for a recalled order,
// mirroring KDSOrderScreenClose - a recall only ever undoes a close on
// whichever terminal(s) actually saw the order in the first place, but
// without this, only the terminal that pressed Recall Orders itself ever
// reopened its own copy; every other terminal the order originally showed
// on (any screen that had completed items on it) kept its own copy marked
// COMPLETE forever, with no way to reopen it.
type KDSOrderScreenRecall struct {
	Reference string `json:"reference"`
	Screen    string `json:"screen"`
	Passmode  bool   `json:"passmode"`
	// Table is the order's own Table at the moment this recall was sent -
	// see D08/getDatabaseOrderFromReferenceAnyStatus's own doc comment for
	// why: a reference reused later the same day can match more than one
	// order, and the table a recall actually originated from is the one
	// piece of context available to disambiguate which one it really
	// means. Advisory only - a peer that doesn't recognize this field, or
	// receives it blank, falls back to its prior id-desc-only behavior.
	Table string `json:"table"`
}

// KDSNewItemsOnOrder is the peer-sync payload for items added to an order
// that already exists on every terminal's own copy (see kds-poller's
// broadcastNewItemsOnOrder) - unlike a brand new order (broadcastNewOrder),
// this never creates a new order row on the receiving end, only appends
// Items to whichever local order already matches Reference. KDSScreenID on
// each item is a local foreign key, not portable between terminals' own
// databases, so the receiving end re-resolves each item's screen by name
// (Screen.Name), same as applyNewOrderRemote already does for a new order's
// own items.
type KDSNewItemsOnOrder struct {
	Reference string     `json:"reference"`
	Items     []KDSItems `json:"items"`
}

// KDSScreenPassAssignment is the peer-sync payload announcing which pass
// terminal (if any) bumps a given screen's orders - see
// announceOwnScreenPassAssignment in kds-webserver-main.
type KDSScreenPassAssignment struct {
	Screen       string `json:"screen"`
	Passterminal string `json:"passterminal"`
}

// KDSPassTerminalAnnounce is the peer-sync payload a pass-mode terminal
// broadcasts so every section terminal can list it as an option in their
// own "Pass Terminal" config dropdown - see announceAsPassTerminal in
// kds-webserver-main.
type KDSPassTerminalAnnounce struct {
	Name string `json:"name"`
}

func (p *KDSTerminalMessage) Publish(channel string, nc *nats.Conn) {
	b, _ := json.Marshal(&p)
	fmt.Println("Publishing KDSMESSAGE on : " + channel)
	nc.Publish(channel, b)
}

type KDSNATSUpdateMessage struct {
	HWID    string `json:"hwid"`
	Service string `json:"service"`
	Extra   string `json:"extra"`
}

type KDSAccountUpdateAJAX struct {
	UserID    int   `json:"user_id"`
	AccountID []int `json:"user_accounts"`
}

type KDSReportsInput struct {
	ReportType   string `json:"report_type"`
	FromDate     string `json:"from_date"`
	ToDate       string `json:"to_date"`
	Medium       string `json:"medium"`
	UserID       int    `json:"user_id"`
	AccountID    int    `json:"account_id"`
	ReportFormat string `json:"report_format"`
	ReportOutput string `json:"report_output"`
	ReportEmail  string `json:"report_email"`
}

type KDSAdminReportsAccountData struct {
	UserID    int    `json:"user_id"`
	AccountID int    `json:"account_id"`
	FromDate  string `json:"from_date"`
	ToDate    string `json:"to_date"`
}

type KDSTerminalSummary struct {
	Terminal      KDSTerminal `json:"terminal"`
	Avg_prep_mins int         `json:"avg_prep_mins"`
	Avg_prep_secs int         `json:"avg_prep_secs"`
	Total_items   int         `json:"total_items"`
	Total_Orders  int         `json:"total_orders"`
	Late1_Orders  int         `json:"late1_orders"`
	Late2_Orders  int         `json:"late2_orders"`
}
type KDSAccountSummary struct {
	Account   KDSAccount           `json:"account"`
	Terminals []KDSTerminal        `json:"terminals"`
	Summary   []KDSTerminalSummary `json:"summary"`
}

type KDSAccountSummaryMainTerminalProfile struct {
	Profile       string `json:"profile"`
	Avg_prep_mins int    `json:"avg_prep_mins"`
	Avg_prep_secs int    `json:"avg_prep_secs"`
	Total_items   int    `json:"total_items"`
	Total_Orders  int    `json:"total_orders"`
	Stage1        int    `json:"stage1"` // 0 - 2 Minutes
	Stage2        int    `json:"stage2"` // 0 - 4 Minutes
	Stage3        int    `json:"stage3"` // 0 - 6 Minutes
	Stage4        int    `json:"stage4"` // 0 - 8 Minutes
	Stage5        int    `json:"stage5"` // 0 - 10 Minutes
	Stage6        int    `json:"stage6"` // 0 - 20 Minutes
	Stage7        int    `json:"stage7"` // > 20 Minutes
}
type KDSAccountSummaryMainTerminal struct {
	Terminal      KDSTerminal                            `json:"terminal"`
	Account       KDSAccount                             `json:"account"`
	Profiles      []KDSAccountSummaryMainTerminalProfile `json:"profiles"`
	Avg_prep_mins int                                    `json:"avg_prep_mins"`
	Avg_prep_secs int                                    `json:"avg_prep_secs"`
	Total_items   int                                    `json:"total_items"`
	Total_Orders  int                                    `json:"total_orders"`
	Stage1        int                                    `json:"stage1"` // 0 - 2 Minutes
	Stage2        int                                    `json:"stage2"` // 0 - 4 Minutes
	Stage3        int                                    `json:"stage3"` // 0 - 6 Minutes
	Stage4        int                                    `json:"stage4"` // 0 - 8 Minutes
	Stage5        int                                    `json:"stage5"` // 0 - 10 Minutes
	Stage6        int                                    `json:"stage6"` // 0 - 20 Minutes
	Stage7        int                                    `json:"stage7"` // > 20 Minutes
}
type KDSAccountSummaryMain struct {
	Terminals []KDSAccountSummaryMainTerminal `json:"terminals"`
}
type KDSInputUpdateConfigFromKDS struct {
	Serial string `json:"serial"`
	Config string `json:"config"`
	HWID   string `json:"hwid"`
}
