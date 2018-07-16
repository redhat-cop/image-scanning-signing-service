package openscap

import (
	"encoding/xml"
)

type OpenSCAPReport struct {
	XMLName xml.Name
	Reports []Reports `xml:"reports"`
}

type Reports struct {
	XMLName xml.Name
	Report  []Report `xml:"report"`
}

type Report struct {
	XMLName xml.Name
	Content Content `xml:"content"`
}

type Content struct {
	XMLName    xml.Name
	TestResult TestResult `xml:"TestResult"`
}

type TestResult struct {
	XMLName     xml.Name
	Title       string       `xml:"title"`
	RuleResults []RuleResult `xml:"rule-result"`
}

type RuleResult struct {
	XMLName  xml.Name
	Severity string `xml:"severity,attr"`
	Result   string `xml:"result"`
}
