package ikentoomodels

// MainMenu ... V3 Menu Main
type MainMenu struct {
	MenuName        string           `json:"menuName"`
	MenuEntryGroups []MenuEntryGroup `json:"menuEntryGroups"`
	IkentooMenuID   int64            `json:"ikentooMenuId"`
}

// MenuEntryGroup ... Menu Entry Groups
type MenuEntryGroup struct {
	Name                 string          `json:"name"`
	ProductName          string          `json:"productName"`
	ProductPrice         float64         `json:"productPrice"`
	Color                string          `json:"color"`
	SKU                  string          `json:"sku"`
	DefaultTaxAmount     string          `json:"defaultTaxAmount"`
	DefaultTaxPercentage string          `json:"defaultTaxPercentage"`
	MenuEntries          []MenuEntry     `json:"menuEntry"`
	MenuDealGroups       []MenuDealGroup `json:"menuDealGroups"`
	Type                 string          `json:"type"`
	MenuGroupItems       []MenuItem      `json:"items"`
}

//MenuDealGroup ... menuDealGroups
type MenuDealGroup struct {
	Description             string          `json:"description"`
	MustSelectAnItem        bool            `json:"mustSelectAnItem"`
	MultiSelectionPermitted bool            `json:"multiSelectionPermitted"`
	MenuDealGroups          []MenuDealGroup `json:"menuDealGroups"`
	MenuDealGroupItems      []MenuItem      `json:"items"`
	MenuEntries             []MenuEntry     `json:"menuEntry"`
}

// MenuEntry ... Menu Entry
type MenuEntry struct {
	Name                  string          `json:"name"`
	ProductName           string          `json:"productName"`
	ProductPrice          string          `json:"productPrice"`
	Color                 string          `json:"color"`
	SKU                   string          `json:"sku"`
	AsSubItem             bool            `json:"asSubItem"`
	CustomItemNameEnabled bool            `json:"customItemNameEnabled"`
	PricingStrategy       string          `json:"pricingStrategy"`
	DefaultTaxAmount      string          `json:"defaultTaxAmount"`
	DefaultTaxPercentage  float64         `json:"defaultTaxPercentage"`
	TaxIncludedInPrice    bool            `json:"taxIncludedInPrice"`
	Type                  string          `json:"type"`
	MenuEntries           []MenuEntry     `json:"menuEntry"`
	MenuDealGroups        []MenuDealGroup `json:"menuDealGroups"`
	MenuEntryItems        []MenuItem      `json:"items"`
}

// MenuItem ... Menu Entry
type MenuItem struct {
	ProductName           string       `json:"productName"`
	ProductPrice          string       `json:"productPrice"`
	SKU                   string       `json:"sku"`
	Color                 string       `json:"color"`
	AsSubItem             bool         `json:"asSubItem"`
	CustomItemNameEnabled bool         `json:"customItemNameEnabled"`
	PricingStrategy       string       `json:"pricingStrategy"`
	TaxIncludedInPrice    bool         `json:"taxIncludedInPrice"`
	Type                  string       `json:"type"`
	ItemRichData          ItemRichData `json:"itemRichData"`
	DefaultTaxAmount      string       `json:"defaultTaxAmount"`
	DefaultTaxPercentage  float64      `json:"defaultTaxPercentage"`
}

// MenuEntryItem ... Menu Entry
type MenuEntryItem struct {
	ProductName           string       `json:"productName"`
	ProductPrice          string       `json:"productPrice"`
	SKU                   string       `json:"sku"`
	AsSubItem             bool         `json:"asSubItem"`
	CustomItemNameEnabled bool         `json:"customItemNameEnabled"`
	PricingStrategy       string       `json:"pricingStrategy"`
	TaxIncludedInPrice    bool         `json:"taxIncludedInPrice"`
	Type                  string       `json:"type"`
	ItemRichData          ItemRichData `json:"itemRichData"`
}

// ItemRichData ... Item Rich Data
type ItemRichData struct {
	SquareImageURL string `json:"squareImageUrl"`
}

// ProductionInstructionList ... Production Instruction List
type ProductionInstructionList struct {
}

// RichItemResponse ... Rich Item Response
type RichItemResponse struct {
	Embedded RichItemDtoList `json:"_embedded"`
}

// RichItemDtoList ... Rich ItemDtoList
type RichItemDtoList struct {
	RichItemDtoList []RichItemDItems `json:"richItemDtoList"`
}

// RichItemDItems ... Rich Item DItems
type RichItemDItems struct {
	BusinessID   int                    `json:"businessId"`
	SKU          string                 `json:"sku"`
	Descriptions []RichItemDescriptions `json:"descriptions"`
}

// RichItemDescriptions ... RichItemDescriptions
type RichItemDescriptions struct {
	LocaleCode  string `json:"localeCode"`
	Description string `json:"description"`
	DisplayName string `json:"displayName"`
}

// MainMenu ... V3 Menu Main
type IKENTOOMainMenu struct {
	MenuName        string                  `json:"menuName"`
	MenuEntryGroups []IKENTOOMenuEntryGroup `json:"menuEntryGroups"`
	IkentooMenuID   int64                   `json:"ikentooMenuId"`
}

// MenuEntryGroup ... Menu Entry Groups
type IKENTOOMenuEntryGroup struct {
	Name                 string                 `json:"name"`
	ProductName          string                 `json:"productName"`
	ProductPrice         float64                `json:"productPrice"`
	Color                string                 `json:"color"`
	SKU                  string                 `json:"sku"`
	DefaultTaxAmount     string                 `json:"defaultTaxAmount"`
	DefaultTaxPercentage string                 `json:"defaultTaxPercentage"`
	MenuEntries          []IKENTOOMenuEntry     `json:"menuEntry"`
	MenuDealGroups       []IKENTOOMenuDealGroup `json:"menuDealGroups"`
	Type                 string                 `json:"type"`
	MenuGroupItems       []IKENTOOMenuItem      `json:"items"`
}

//MenuDealGroup ... menuDealGroups
type IKENTOOMenuDealGroup struct {
	Description             string                 `json:"description"`
	MustSelectAnItem        bool                   `json:"mustSelectAnItem"`
	MultiSelectionPermitted bool                   `json:"multiSelectionPermitted"`
	MenuDealGroups          []IKENTOOMenuDealGroup `json:"menuDealGroups"`
	MenuDealGroupItems      []IKENTOOMenuItem      `json:"items"`
	MenuEntries             []IKENTOOMenuEntry     `json:"menuEntry"`
}

// MenuEntry ... Menu Entry
type IKENTOOMenuEntry struct {
	Name                  string                 `json:"name"`
	ProductName           string                 `json:"productName"`
	ProductPrice          string                 `json:"productPrice"`
	Color                 string                 `json:"color"`
	SKU                   string                 `json:"sku"`
	AsSubItem             bool                   `json:"asSubItem"`
	CustomItemNameEnabled bool                   `json:"customItemNameEnabled"`
	PricingStrategy       string                 `json:"pricingStrategy"`
	DefaultTaxAmount      string                 `json:"defaultTaxAmount"`
	DefaultTaxPercentage  string                 `json:"defaultTaxPercentage"`
	TaxIncludedInPrice    bool                   `json:"taxIncludedInPrice"`
	Type                  string                 `json:"type"`
	MenuEntries           []IKENTOOMenuEntry     `json:"menuEntry"`
	MenuDealGroups        []IKENTOOMenuDealGroup `json:"menuDealGroups"`
	MenuEntryItems        []IKENTOOMenuItem      `json:"items"`
}

// MenuItem ... Menu Entry
type IKENTOOMenuItem struct {
	ProductName           string              `json:"productName"`
	ProductPrice          string              `json:"productPrice"`
	SKU                   string              `json:"sku"`
	Color                 string              `json:"color"`
	AsSubItem             bool                `json:"asSubItem"`
	CustomItemNameEnabled bool                `json:"customItemNameEnabled"`
	PricingStrategy       string              `json:"pricingStrategy"`
	TaxIncludedInPrice    bool                `json:"taxIncludedInPrice"`
	Type                  string              `json:"type"`
	ItemRichData          IKENTOOItemRichData `json:"itemRichData"`
	DefaultTaxAmount      string              `json:"defaultTaxAmount"`
	DefaultTaxPercentage  float64             `json:"defaultTaxPercentage"`
}

// MenuEntryItem ... Menu Entry
type IKENTOOMenuEntryItem struct {
	ProductName           string              `json:"productName"`
	ProductPrice          string              `json:"productPrice"`
	SKU                   string              `json:"sku"`
	AsSubItem             bool                `json:"asSubItem"`
	CustomItemNameEnabled bool                `json:"customItemNameEnabled"`
	PricingStrategy       string              `json:"pricingStrategy"`
	TaxIncludedInPrice    bool                `json:"taxIncludedInPrice"`
	Type                  string              `json:"type"`
	ItemRichData          IKENTOOItemRichData `json:"itemRichData"`
}

// ItemRichData ... Item Rich Data
type IKENTOOItemRichData struct {
	SquareImageURL string `json:"squareImageUrl"`
}

type IKENTOOItemResults struct {
	Results []IKENTOOIkentooItems `json:"results"`
	Amount  int                   `json:"amount"`
	Offset  int                   `json:"offset"`
	Total   int                   `json:"total"`
}

// IKENTOOIkentooItems ... IkentooItems
type IKENTOOIkentooItems struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	DocketName  string          `json:"docketName"`
	SKU         string          `json:"sku"`
	Active      bool            `json:"active"`
	Barcode     string          `json:"barcode"`
	CostPrice   float64         `json:"costPrice"`
	SharingType string          `json:"sharingType"`
	PriceMode   string          `json:"priceMode"`
	Prices      []IKENTOOPrices `json:"prices"`
}

type IKENTOOPrices struct {
	Amount float64 `json:"amount"`
	Name   string  `json:"name"`
}
