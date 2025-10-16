package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v2"
)

type VibeAttachement struct {
	Filename string `yaml:"filename"`
	URL      string `yaml:"url"`
}

type VibeRequest struct {
	Prompt       string            `yaml:"prompt"`
	Checks       string            `yaml:"checks"`
	Attachements []VibeAttachement `yaml:"attachements"`
}

type VibeResponse struct {
	Type     string `yaml:"type"`
	Filename string `yaml:"filename"`
	Content  string `yaml:"content"`
}

var OpenAI openai.Client

func InitOpenAI() error {
	if os.Getenv("OPENAI_KEY") == "" {
		return fmt.Errorf("openai_key_missing")
	}

	OpenAI = openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENAI_KEY")),
	)

	_, err := OpenAI.Models.List(context.Background())
	if err != nil {
		return fmt.Errorf("openai_key_invalid: %w", err)
	}

	return nil
}

func ValidateVibeBundle(files []VibeResponse) []error {
	var errs []error

	readme := findByName(files, "README.md")
	index := findByName(files, "index.html")

	if readme == nil {
		errs = append(errs, errors.New(`missing required file "README.md"`))
	} else if !strings.EqualFold(readme.Type, "markdown") {
		errs = append(errs, fmt.Errorf(`"README.md" must have type "markdown" (got %q)`, readme.Type))
	} else if strings.TrimSpace(readme.Content) == "" {
		errs = append(errs, errors.New(`"README.md" content is empty`))
	}

	if index == nil {
		errs = append(errs, errors.New(`missing required file "index.html"`))
	} else if !strings.EqualFold(index.Type, "html") {
		errs = append(errs, fmt.Errorf(`"index.html" must have type "html" (got %q)`, index.Type))
	} else if strings.TrimSpace(index.Content) == "" {
		errs = append(errs, errors.New(`"index.html" content is empty`))
	}

	if len(files) != 2 {
		errs = append(errs, fmt.Errorf("expected exactly 2 files (README.md and index.html), got %d", len(files)))
	}

	if readme != nil && strings.EqualFold(readme.Type, "markdown") && strings.TrimSpace(readme.Content) != "" {
		if err := validateMarkdown(readme.Content); err != nil {
			errs = append(errs, fmt.Errorf(`markdown parse error in "README.md": %w`, err))
		}
	}

	if index != nil && strings.EqualFold(index.Type, "html") && strings.TrimSpace(index.Content) != "" {
		if err := validateHTML(index.Content); err != nil {
			errs = append(errs, fmt.Errorf(`html parse error in "index.html": %w`, err))
		}
	}

	return errs
}

func findByName(files []VibeResponse, name string) *VibeResponse {
	for i := range files {
		if files[i].Filename == name {
			return &files[i]
		}
	}
	return nil
}

func validateMarkdown(md string) error {
	mdParser := goldmark.New()
	reader := text.NewReader([]byte(md))
	_ = mdParser.Parser().Parse(reader)

	var renderErr error
	defer func() {
		if r := recover(); r != nil {
			renderErr = fmt.Errorf("panic while rendering markdown: %v", r)
		}
	}()

	return renderErr
}

func validateHTML(s string) error {
	_, err := html.Parse(strings.NewReader(s))
	return err
}

func extractRawYAML(s string) string {
	// Trim BOM & whitespace
	s = strings.TrimSpace(strings.TrimPrefix(s, "\uFEFF"))

	// Common cases:
	// ```yaml\n...content...\n```
	// ```yml\n...```
	// ```\n...```
	if strings.HasPrefix(s, "```") {
		// Remove opening ```
		s = strings.TrimPrefix(s, "```")
		// Optional language hint (yaml, yml, YAML, etc.)
		s = strings.TrimSpace(s)
		// Drop first line if it’s just a language tag
		firstNL := strings.IndexByte(s, '\n')
		if firstNL != -1 {
			firstLine := strings.TrimSpace(s[:firstNL])
			if firstLine == "yaml" || firstLine == "yml" || strings.EqualFold(firstLine, "yaml") || strings.EqualFold(firstLine, "yml") {
				s = s[firstNL+1:]
			}
		}
		// Cut trailing ```
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}

func GenerateFrontend(vr VibeRequest) (*[]VibeResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	attachmentsYAML, err := yaml.Marshal(vr.Attachements)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attachments to yaml: %w", err)
	}

	sys := `You are a developer who outputs exactly TWO files as a YAML array of objects:
- type: "markdown" | "html"
- filename: string
- content: string
Files required (exactly these two):
1) README.md (type: markdown)
2) index.html (type: html)

NON-NEGOTIABLE BEHAVIOR:
- DO NOT ASSUME. Derive everything from the user prompt, the provided checks, and the provided attachments list (filenames+URLs).
- If required info is missing/ambiguous, implement a graceful runtime error path in the HTML (visible message) and log a clear console error; do NOT fabricate data, fields, URLs, or formats.
- Resolve fields by NAME/KEY from real artifacts, not by index or guesswork.
- Parse inputs robustly when needed (e.g., use safe parsers for CSV/JSON if parsing is part of the task). Do not write naive parsers if a standard library from a CDN exists.
- Only use public CDNs for any libs (e.g., jsDelivr/unpkg/cdnjs) AND DO NOT use integrity hashes at ALL.
- Validate presence of required DOM elements before writing into them; fail gracefully if missing.
- Never invent attachment filenames or paths; only use those explicitly provided in the attachments list.
- If constraints cannot be met with given info, the page must render a clear user-facing error box and console.error an explanation.
- If there are possible user input fields/query string/params etc. fall back to sensible defaults if not provided.

OUTPUT RULES:
- Return a YAML array with exactly two objects (README.md, index.html), each having only: type, filename, content.
- No extra keys, comments, prose, or backticks.`

	userPrompt := fmt.Sprintf(
		`TASK:
%q

EVALUATION CHECKS (must design for these; do not assume anything not stated):
%q

ATTACHMENTS (authoritative list; only use these if needed):
---
%s
---

IMPLEMENTATION CONSTRAINTS:
- Treat attachments as the single source of truth for sample data/assets. If the task needs "a file named X", locate it by exact filename in the list; if not present, implement a visible error state instead of guessing.
- When reading structured data (CSV/JSON/etc.), detect columns/keys by exact header/key names found in the actual file content; never rely on hard-coded column indices or imagined keys.
- If a selector/ID/element is required by the task or checks, ensure it exists before using it; otherwise show a visible error and log details.
- Use only standard, CDN-loadable libraries if a parser/utility is needed. If a library is used, load it from a public CDN and handle load failures gracefully.
- Make behavior deterministic and auditable: log (console) what file(s) were used, what keys/columns were resolved, and any data that was ignored because it didn’t match requirements.
- If any requirement cannot be satisfied with the provided information, render a clear in-page error message describing exactly what is missing.
- If there are possible user input fields/query string/params etc. fall back to sensible defaults if not provided.

OUTPUT FORMAT (strict):
- YAML array with exactly two items:
  - README.md (type: markdown) — professional, how to open locally, mention MIT license with a link to LICENSE (do not include LICENSE file).
  - index.html (type: html) — the full working page.
- No extra keys, comments, or backticks.`,
		vr.Prompt, vr.Checks, string(attachmentsYAML),
	)

	resp, err := OpenAI.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: "gpt-5-mini",
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfSystem: &openai.ChatCompletionSystemMessageParam{
						Content: openai.ChatCompletionSystemMessageParamContentUnion{
							OfString: openai.String(sys),
						},
					},
				},
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(userPrompt),
						},
					},
				},
			},
		},
		option.WithRequestTimeout(320*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("openai_error: %w", err)
	}

	raw := resp.Choices[0].Message.Content
	clean := extractRawYAML(raw)

	var parsed []VibeResponse
	if err := yaml.Unmarshal([]byte(clean), &parsed); err != nil {
		log.Printf("yaml unmarshal error: %v; content:\n%s", err, raw)
		return nil, fmt.Errorf("failed_to_parse_yaml: %w", err)
	}

	return &parsed, nil
}

func ModifyFrontend(vr VibeRequest, existing []VibeResponse) (*[]VibeResponse, error) {
	if err := InitOpenAI(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	attachmentsYAML, err := yaml.Marshal(vr.Attachements)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attachments to yaml: %w", err)
	}
	existingYAML, err := yaml.Marshal(existing)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing files to yaml: %w", err)
	}

	sys := `You modify an existing two-file frontend bundle. Output exactly TWO files as a YAML array of objects:
- type: "markdown" | "html"
- filename: string (must remain README.md or index.html)
- content: string (entire new contents)

RULES:
- Edit the given files to satisfy the new brief/checks. Do not add or remove files.
- Keep filenames identical (README.md, index.html).
- Do not assume; if info is missing, implement a visible in-page error and console.error.
- Use only attachments provided; never invent paths.
- Validate DOM presence before writing; fail visibly otherwise.
- Full file contents only; no diffs; no comments; no backticks.`

	userPrompt := fmt.Sprintf(
		`CURRENT FILES (authoritative):
---
%s
---

NEW TASK/CONSTRAINTS:
Brief:
%q

Checks (design for these; don't invent anything not stated):
%q

Attachments:
---
%s
---

Please return YAML with exactly two objects (README.md markdown, index.html html) with updated contents to satisfy the brief & checks.`,
		string(existingYAML),
		vr.Prompt,
		vr.Checks,
		string(attachmentsYAML),
	)

	resp, err := OpenAI.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: "gpt-5-mini",
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfSystem: &openai.ChatCompletionSystemMessageParam{
						Content: openai.ChatCompletionSystemMessageParamContentUnion{
							OfString: openai.String(sys),
						},
					},
				},
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(userPrompt),
						},
					},
				},
			},
		},
		option.WithRequestTimeout(320*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("openai_error: %w", err)
	}

	raw := resp.Choices[0].Message.Content
	clean := extractRawYAML(raw)

	var parsed []VibeResponse
	if err := yaml.Unmarshal([]byte(clean), &parsed); err != nil {
		log.Printf("yaml unmarshal error: %v; content:\n%s", err, raw)
		return nil, fmt.Errorf("failed_to_parse_yaml: %w", err)
	}
	return &parsed, nil
}
