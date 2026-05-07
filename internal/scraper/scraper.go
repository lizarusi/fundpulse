package scraper

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type FundSnapshot struct {
	Name        string
	Currency    string
	NAV         float64
	NAVDate     time.Time
	Change1DAbs float64
	Change1DPct float64
	RiskLabel   string
	RiskLevel   int
}

func Parse(r io.Reader) (FundSnapshot, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return FundSnapshot{}, fmt.Errorf("parse html: %w", err)
	}

	var s FundSnapshot

	nameNode := doc.Find("h1.productName").First()
	if nameNode.Length() == 0 {
		return s, fmt.Errorf("h1.productName not found")
	}
	s.Name = normalizeText(nameNode.Text())

	priceCell := doc.Find(".productValueSumUp .primaryContent").First()
	if priceCell.Length() == 0 {
		return s, fmt.Errorf("primaryContent not found")
	}
	navText := strings.TrimSpace(priceCell.Find(".productBigText").Text())
	nav, err := parsePolishFloat(navText)
	if err != nil {
		return s, fmt.Errorf("nav %q: %w", navText, err)
	}
	s.NAV = nav

	fullPrice := normalizeText(priceCell.Text())
	s.Currency = strings.TrimSpace(strings.TrimPrefix(fullPrice, navText))

	dateText := strings.TrimSpace(doc.Find(".productValueSumUp .lightProductText").First().Text())
	d, err := time.Parse("02.01.2006", dateText)
	if err != nil {
		return s, fmt.Errorf("nav date %q: %w", dateText, err)
	}
	s.NAVDate = d

	changeText := doc.Find(".productValueChange").First().Text()
	abs, pct, err := parseChange(changeText)
	if err != nil {
		return s, fmt.Errorf("change: %w", err)
	}
	s.Change1DAbs = abs
	s.Change1DPct = pct

	label, level, ok := findRisk(doc)
	if !ok {
		return s, fmt.Errorf("risk indicator not found")
	}
	s.RiskLabel = label
	s.RiskLevel = level

	return s, nil
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	return strings.TrimSpace(s)
}

func parsePolishFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimPrefix(s, "+")
	return strconv.ParseFloat(s, 64)
}

var (
	absRE = regexp.MustCompile(`^[\s]*([+-]?\d+(?:[  ]?\d{3})*,\d+)`)
	pctRE = regexp.MustCompile(`([+-]?\d+,\d+)\s*%`)
)

func parseChange(s string) (abs, pct float64, err error) {
	m := absRE.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, 0, fmt.Errorf("abs change not found in %q", s)
	}
	abs, err = parsePolishFloat(m[1])
	if err != nil {
		return 0, 0, fmt.Errorf("abs: %w", err)
	}
	m = pctRE.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, 0, fmt.Errorf("pct change not found in %q", s)
	}
	pct, err = parsePolishFloat(m[1])
	if err != nil {
		return 0, 0, fmt.Errorf("pct: %w", err)
	}
	return abs, pct, nil
}

func findRisk(doc *goquery.Document) (label string, level int, ok bool) {
	doc.Find(".basicLabel").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if normalizeText(sel.Text()) != "Poziom ryzyka (SRI)" {
			return true
		}
		val := sel.Next()
		level = val.Find(".riskActive").Length()
		clone := val.Clone()
		clone.Find(".riskContainer").Remove()
		label = normalizeText(clone.Text())
		ok = true
		return false
	})
	return label, level, ok
}
