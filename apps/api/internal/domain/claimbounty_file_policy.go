package domain

import (
	"fmt"
	"path"
	"strings"
)

func (f OrderFile) CaseBundlePath() string {
	directory := map[string]string{"primary_paper": "paper", "supplement": "supplements", "preregistration": "preregistration", "data": "data", "code": "code", "environment": "environment", "data_dictionary": "dictionaries", "other_evidence": "sources"}[f.Role]
	name := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '-'
		}
		return r
	}, f.OriginalDisplayName)
	return path.Join(directory, fmt.Sprintf("%s-%s", f.ID, name))
}

func (f OrderFile) ArchivePath() string { return path.Join("case-bundle", f.CaseBundlePath()) }

func (f OrderFile) AcceptsDeclaredMediaType(declared string) bool {
	declared = strings.ToLower(strings.TrimSpace(strings.SplitN(declared, ";", 2)[0]))
	extension := strings.ToLower(path.Ext(f.OriginalDisplayName))
	if !allowedFileName(f.Role, f.OriginalDisplayName) {
		return false
	}
	accepted := map[string]map[string]bool{
		".pdf": set("application/pdf"), ".txt": set("text/plain"), ".md": set("text/markdown", "text/plain"),
		".docx": set("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip"),
		".csv":  set("text/csv", "text/plain", "application/csv"), ".tsv": set("text/tab-separated-values", "text/plain"),
		".xlsx": set("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip"),
		".json": set("application/json", "text/plain"), ".parquet": set("application/vnd.apache.parquet", "application/octet-stream"),
		".dta": set("application/x-stata", "application/octet-stream"), ".sav": set("application/x-spss-sav", "application/octet-stream"),
		".rds": set("application/x-r-data", "application/octet-stream", "application/gzip"), ".rdata": set("application/x-r-data", "application/octet-stream"),
		".r": set("text/x-r-source", "text/plain"), ".py": set("text/x-python", "text/plain"), ".ipynb": set("application/json"),
		".do": set("text/plain"), ".sql": set("application/sql", "text/plain"), ".sh": set("text/x-shellscript", "application/x-sh", "text/plain"),
		".zip": set("application/zip", "application/x-zip-compressed"), ".lock": set("text/plain", "application/json"),
		".yaml": set("application/yaml", "text/yaml", "text/plain"), ".yml": set("application/yaml", "text/yaml", "text/plain"),
		".toml": set("application/toml", "text/plain"), "": set("text/plain"),
	}
	return accepted[extension][declared]
}

func allowedFileName(role, displayName string) bool {
	extension := strings.ToLower(path.Ext(displayName))
	if role != "environment" {
		return allowedRoleExtensions[role][extension]
	}
	name := strings.ToLower(displayName)
	if name == "dockerfile" || name == "renv.lock" || name == "requirements.txt" || strings.HasPrefix(name, "requirements-") && extension == ".txt" {
		return true
	}
	return extension == ".lock" || extension == ".yaml" || extension == ".yml" || extension == ".toml"
}
