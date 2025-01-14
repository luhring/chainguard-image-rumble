package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ImageScanSummary struct {
	ID string `bigquery:"id"` // This is faux primary key, the shas256sum of (image + "--" + scanner + "--" + time)

	Image            string `bigquery:"image"`
	Digest           string `bigquery:"digest"`
	Scanner          string `bigquery:"scanner"`
	ScannerVersion   string `bigquery:"scanner_version"`
	ScannerDbVersion string `bigquery:"scanner_db_version"`
	Time             string `bigquery:"time"`
	Created          string `bigquery:"created"`
	LowCveCount      int    `bigquery:"low_cve_count"`
	MedCveCount      int    `bigquery:"med_cve_count"`
	HighCveCount     int    `bigquery:"high_cve_count"`
	CritCveCount     int    `bigquery:"crit_cve_count"`

	// NegligibleCveCount is a grype specific field
	NegligibleCveCount int `bigquery:"negligible_cve_count"`

	UnknownCveCount int  `bigquery:"unknown_cve_count"`
	TotCveCount     int  `bigquery:"tot_cve_count"`
	Success         bool `bigquery:"success"`

	RawGrypeJSON string `bigquery:"raw_grype_json"`
}

func (row *ImageScanSummary) SetID() {
	row.ID = sha256Sum(row.id())
}

func (row *ImageScanSummary) id() string {
	return strings.Join([]string{row.Image, row.Scanner, row.Time}, "--")
}

func (row *ImageScanSummary) ExtractVulns() ([]*Vuln, error) {
	// No Grype data present which we rely on for this info
	if row.RawGrypeJSON == "" {
		return []*Vuln{}, nil
	}
	if row.ID == "" {
		row.SetID()
	}
	var output GrypeScanOutput
	if err := json.Unmarshal([]byte(row.RawGrypeJSON), &output); err != nil {
		return nil, err
	}
	uniqueVulns := map[string]*Vuln{}
	for _, match := range output.Matches {
		v := Vuln{
			ScanID:        row.ID,
			Name:          match.Artifact.Name,
			Installed:     match.Artifact.Version,
			FixedIn:       strings.Join(match.Vulnerability.Fix.Versions, ","),
			Type:          match.Artifact.Type,
			Vulnerability: match.Vulnerability.ID,
			Severity:      match.Vulnerability.Severity,
			Time:          row.Time,
		}
		v.SetID()
		uniqueVulns[v.ID] = &v
	}
	vulns := []*Vuln{}
	for _, vuln := range uniqueVulns {
		vulns = append(vulns, vuln)
	}
	sort.Slice(vulns, func(i, j int) bool {
		return vulns[i].id() < vulns[j].id()
	})
	return vulns, nil
}

type Vuln struct {
	ID            string `bigquery:"id"`      // This is faux primary key, the shas256sum of (name + "--" + installed + "--" + vulnerability + "--" + type + "--" + time)
	ScanID        string `bigquery:"scan_id"` // This is faux foreign key to the table above
	Name          string `bigquery:"name"`
	Installed     string `bigquery:"installed"`
	FixedIn       string `bigquery:"fixed_in"`
	Type          string `bigquery:"type"`
	Vulnerability string `bigquery:"vulnerability"`
	Severity      string `bigquery:"severity"`
	Time          string `bigquery:"time"`
}

func (row *Vuln) SetID() {
	row.ID = sha256Sum(row.id())
}

func (row *Vuln) id() string {
	return strings.Join([]string{row.Name, row.Installed, row.Vulnerability, row.Type, row.Time}, "--")
}

func sha256Sum(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
