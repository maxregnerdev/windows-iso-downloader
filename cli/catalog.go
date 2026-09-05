package main

import (
	"strconv"
	"strings"
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

var consumerProducts = []Product{
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
	// Windows 11 vNext (Copilot+ PC) builds 30000+
	{"30000", "Windows 11 vNext Build 30000 (Copilot+ PC)"},
	{"30001", "Windows 11 vNext Build 30001 (Copilot+ PC)"},
	{"30002", "Windows 11 vNext Build 30002 (Copilot+ PC)"},
	{"30005", "Windows 11 vNext Build 30005 (Copilot+ PC)"},
	{"30010", "Windows 11 vNext Build 30010 (Copilot+ PC)"},
	{"30015", "Windows 11 vNext Build 30015 (Copilot+ PC)"},
	{"30020", "Windows 11 vNext Build 30020 (Copilot+ PC)"},
	{"30025", "Windows 11 vNext Build 30025 (Copilot+ PC)"},
	{"30030", "Windows 11 vNext Build 30030 (Copilot+ PC)"},
	{"30050", "Windows 11 vNext Build 30050 (Copilot+ PC)"},
}

var evalProducts = []EvalProduct{
	{"server-2025", "Windows Server 2025", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2025"},
	{"server-2022", "Windows Server 2022", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2022"},
	{"server-2019", "Windows Server 2019", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2019"},
	{"server-2016", "Windows Server 2016", "https://www.microsoft.com/en-us/evalcenter/download-windows-server-2016"},
	{"win11-ent", "Windows 11 Enterprise", "https://www.microsoft.com/en-us/evalcenter/download-windows-11-enterprise"},
	// Copilot OS Evaluation
	{"copilot-os", "Windows 11 Copilot+ PC OS", "https://www.microsoft.com/en-us/evalcenter/download-windows-11-copilot-pc"},
}

func findProductByID(id string) (Product, bool) {
	for _, p := range consumerProducts {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}

func searchProducts(query string) []Product {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return consumerProducts
	}
	var results []Product
	for _, p := range consumerProducts {
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
// Plain strings.Contains would let a query word match a digit fragment
// buried inside an unrelated number -- e.g. "10" matching inside a build
// number like "26100.1742" -- which made "windows 10" incorrectly return
// Windows 11 24H2 results.
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

// IsCopilotOSBuild returns true if the product ID is a Copilot+ PC / vNext build (30000+)
func IsCopilotOSBuild(productID string) bool {
	id, err := strconv.Atoi(productID)
	if err != nil {
		return false
	}
	return id >= 30000
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
