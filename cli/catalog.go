package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Product struct {
	ID   string
	Name string
}

type EvalProduct struct {
	Slug    string
	Name    string
	EvalURL string
}

// Static known products (stable releases)
var staticProducts = []Product{
	{"52", "Windows 8.1 (9600.17415)"},
	{"2378", "Windows 10 22H2 Home China (19045.2006)"},
	{"2618", "Windows 10 22H2 v1 (19045.2965)"},
	{"3113", "Windows 11 24H2 (26100.1742)"},
	{"3114", "Windows 11 24H2 Home China (26100.1742)"},
	{"3115", "Windows 11 24H2 Pro China (26100.1742)"},
	{"3131", "Windows 11 Arm64 24H2 (26100.1742)"},
	{"3132", "Windows 11 Arm64 24H2 Home China (26100.1742)"},
	{"3133", "Windows 11 Arm64 24H2 Pro China (26100.1742)"},
	{"3262", "Windows 11 25H2 (26200.6584)"},
	{"3263", "Windows 11 25H2 Home China (26200.6584)"},
	{"3264", "Windows 11 25H2 Pro China (26200.6584)"},
	{"3265", "Windows 11 Arm64 25H2 (26200.6584)"},
	{"3266", "Windows 11 Arm64 25H2 Home China (26200.6584)"},
	{"3267", "Windows 11 Arm64 25H2 Pro China (26200.6584)"},
	{"3321", "Windows 11 25H2 (V2)"},
	{"3322", "Windows 11 25H2 Home China (V2)"},
	{"3323", "Windows 11 25H2 Pro China (V2)"},
	{"3324", "Windows 11 Arm64 25H2 (V2)"},
	{"3325", "Windows 11 Arm64 25H2 Home China (V2)"},
	{"3326", "Windows 11 Arm64 25H2 Pro China (V2)"},
}

// Copilot OS / Project Aion dynamic discovery range
const (
	CopilotOSMinID = 28000
	CopilotOSMaxID = 32000
)

var evalProducts = []EvalProduct{
	{"server-2025", "Windows Server 2025", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2025"},
	{"server-2022", "Windows Server 2022", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2022"},
	{"server-2019", "Windows Server 2019", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2019"},
	{"server-2016", "Windows Server 2016", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2016"},
	{"win11-ent", "Windows 11 Enterprise", "https://www.microsoft.com/en-us/evalcenter/download-windows-11-enterprise"},
	{"copilot-os", "Windows 11 Copilot+ PC OS", "https://www.microsoft.com/en-us/evalcenter/download-windows-11-copilot-pc"},
}



// DiscoverCopilotOSProducts dynamically fetches Copilot+ PC / Project Aion builds from Microsoft API
func DiscoverCopilotOSProducts() []Product {
	var products []Product

	// Scan the Copilot OS range
	for id := CopilotOSMinID; id <= CopilotOSMaxID; id++ {
		productName := fetchProductNameFromAPI(fmt.Sprintf("%d", id))
		if productName != "" {
			products = append(products, Product{
				ID:   fmt.Sprintf("%d", id),
				Name: productName + " (Copilot+ PC / Project Aion)",
			})
		}
		// Rate limit
		time.Sleep(200 * time.Millisecond)
	}

	return products
}

// fetchProductNameFromAPI fetches product name directly from Microsoft API
func fetchProductNameFromAPI(productID string) string {
	url := fmt.Sprintf("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?profile=%s&productEditionId=%s&SKU=undefined&friendlyFileName=undefined&Locale=%s&sessionID=%s",
		msdlProfile, productID, msdlLocale, generateSessionID())

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("User-Agent", msdlUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.microsoft.com/en-us/software-download/windowsinsiderpreview")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Handle double-encoded JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Try double-decode
		var strData string
		if err := json.Unmarshal(body, &strData); err == nil {
			if err := json.Unmarshal([]byte(strData), &data); err != nil {
				return ""
			}
		} else {
			return ""
		}
	}

	// Check for errors
	if errors, ok := data["Errors"].([]interface{}); ok && len(errors) > 0 {
		return ""
	}

	// Get product name from first SKU
	skus, ok := data["Skus"].([]interface{})
	if !ok || len(skus) == 0 {
		return ""
	}

	firstSKU, ok := skus[0].(map[string]interface{})
	if !ok {
		return ""
	}

	if name, ok := firstSKU["ProductDisplayName"].(string); ok && name != "" {
		return name
	}
	if name, ok := firstSKU["LocalizedProductDisplayName"].(string); ok && name != "" {
		return name
	}

	return ""
}

// generateSessionID creates a new session ID for Microsoft API
func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// GetAllProducts returns all products including dynamically discovered Copilot OS builds
func GetAllProducts() []Product {
	// Start with static products
	all := make([]Product, len(staticProducts))
	copy(all, staticProducts)

	// Add dynamically discovered Copilot OS products
	copilotProducts := DiscoverCopilotOSProducts()
	all = append(all, copilotProducts...)

	return all
}

// GetConsumerProducts returns static products only (for compatibility)
func GetConsumerProducts() []Product {
	return staticProducts
}

func findProductByID(id string) (Product, bool) {
	// First check static products
	for _, p := range staticProducts {
		if p.ID == id {
			return p, true
		}
	}

	// Then check dynamically discovered Copilot OS products
	for _, p := range DiscoverCopilotOSProducts() {
		if p.ID == id {
			return p, true
		}
	}

	return Product{}, false
}

func searchProducts(query string) []Product {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return GetAllProducts()
	}

	all := GetAllProducts()
	var results []Product

	for _, p := range all {
		name := strings.ToLower(p.Name)
		match := true
		for _, w := range words {
			if !containsWordStart(name, w) {
				match = false
				break
			}
		}
		if match {
			results = append(results, p)
		}
	}
	return results
}

// containsWordStart reports whether s contains substr starting at a word
// boundary: the start of s, or right after a non-alphanumeric character.
func containsWordStart(s, substr string) bool {
	from := 0
	for {
		i := strings.Index(s[from:], substr)
		if i < 0 {
			return false
		}
		pos := from + i
		if pos == 0 || !isAlphanumeric(s[pos-1]) {
			return true
		}
		from = pos + 1
	}
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func findEvalProduct(slug string) (EvalProduct, bool) {
	for _, p := range evalProducts {
		if p.Slug == slug {
			return p, true
		}
	}
	return EvalProduct{}, false
}

// IsCopilotOSBuild returns true if the product ID is in Copilot+ PC / Project Aion range
func IsCopilotOSBuild(productID string) bool {
	id, err := strconv.Atoi(productID)
	if err != nil {
		return false
	}
	return id >= CopilotOSMinID && id <= CopilotOSMaxID
}

// IsCopilotOSProduct checks if a product is a Copilot+ PC build
func IsCopilotOSProduct(p Product) bool {
	id, err := strconv.Atoi(p.ID)
	if err != nil {
		return false
	}
	return id >= CopilotOSMinID
}

// consumerProducts is kept for backward compatibility - now returns all products
var consumerProducts = GetConsumerProducts()

func init() {
	// Initialize consumerProducts (already done above)
	// This ensures backward compatibility
}
