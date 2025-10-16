package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

func Headers() map[string]string {
	return map[string]string{
		"Accept":               "application/vnd.github+json",
		"Authorization":        fmt.Sprintf("Bearer %s", os.Getenv("GITHUB_KEY")),
		"X-GitHub-Api-Version": GITHUB_API_VERSION,
	}
}

const GITHUB_API_VERSION = "2022-11-28"

type Repository struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	HTMLURL    string `json:"html_url"`
	CommitsURL string `json:"commits_url"`
}

type Commit struct {
	SHA     string `json:"sha"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Author struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type PagesBuild struct {
	Status string `json:"status"`
	Commit string `json:"commit"`
}

type ContentResp struct {
	SHA      string `json:"sha"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func InitGit() error {
	repos, err := GetRepositories()
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("git_check_failed")
	}

	return nil
}

func GetRepositories() ([]Repository, error) {
	resp, err := HTTPGetClient(fmt.Sprintf("https://api.github.com/users/%s/repos", os.Getenv("GITHUB_USER")), Headers())

	if err != nil {
		return nil, err
	}

	var repos []Repository

	if err := json.Unmarshal(resp, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func CreateRepository(name string) error {
	body := map[string]any{
		"name":    name,
		"private": false,
	}

	_, err := HTTPPostPutClient("https://api.github.com/user/repos", Headers(), body, "POST")
	if err != nil {
		return err
	}

	return nil
}

func CreateFile(repo string, path string, content string, message string) error {
	body := map[string]any{
		"message": message,
		"committer": map[string]string{
			"name":  os.Getenv("GITHUB_USER"),
			"email": os.Getenv("GITHUB_EMAIL"),
		},
		"content": ToBase64(content),
	}

	_, err := HTTPPostPutClient(fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s", os.Getenv("GITHUB_USER"), repo, path), Headers(), body, "PUT")

	if err != nil {
		return err
	}

	return nil
}

func CreateFileBytes(repo, path string, data []byte, message string) error {
	body := map[string]any{
		"message": message,
		"committer": map[string]string{
			"name":  os.Getenv("GITHUB_USER"),
			"email": os.Getenv("GITHUB_EMAIL"),
		},
		"content": ToBase64Bytes(data),
	}

	_, err := HTTPPostPutClient(
		fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
			os.Getenv("GITHUB_USER"), repo, path),
		Headers(), body, "PUT",
	)
	return err
}

func GetFileWithSHA(repo, path string) (sha string, decoded string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", os.Getenv("GITHUB_USER"), repo, path)
	resp, err := HTTPGetClient(url, Headers())
	if err != nil {
		return "", "", err
	}

	var cr ContentResp
	if err := json.Unmarshal(resp, &cr); err != nil {
		return "", "", err
	}
	if cr.Encoding != "base64" {
		return "", "", fmt.Errorf("unexpected encoding: %s", cr.Encoding)
	}
	data, decErr := FromBase64(cr.Content) // implement using base64.StdEncoding.DecodeString
	if decErr != nil {
		return "", "", decErr
	}
	return cr.SHA, string(data), nil
}

func UpdateFile(repo, path, newContent, message, sha string) error {
	body := map[string]any{
		"message": message,
		"committer": map[string]string{
			"name":  os.Getenv("GITHUB_USER"),
			"email": os.Getenv("GITHUB_EMAIL"),
		},
		"content": ToBase64(newContent),
		"sha":     sha,
	}
	_, err := HTTPPostPutClient(
		fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
			os.Getenv("GITHUB_USER"), repo, path),
		Headers(), body, "PUT",
	)
	return err
}

func SetupPages(repo string) error {
	_, err := HTTPPostPutClient(fmt.Sprintf("https://api.github.com/repos/%s/%s/pages", os.Getenv("GITHUB_USER"), repo), Headers(), map[string]any{
		"source": map[string]string{
			"branch": "main",
			"path":   "/",
		},
	}, "POST")

	if err != nil {
		return err
	}

	return nil
}

func SetupRepo(repo string) error {
	if err := CreateFile(
		repo,
		"LICENSE",
		CreateLicense(fmt.Sprintf("%s <%s>",
			os.Getenv("GITHUB_NAME"),
			os.Getenv("GITHUB_EMAIL"))),
		"init: add license"); err != nil {
		return err
	}

	if err := SetupPages(repo); err != nil {
		return err
	}

	return nil
}

func DeleteRepository(repo string) error {
	return HTTPDeleteClient(fmt.Sprintf(
		"https://api.github.com/repos/%s/%s", os.Getenv("GITHUB_USER"), repo), Headers())
}

func GetLastCommitHash(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=1", os.Getenv("GITHUB_USER"), repo)
	resp, err := HTTPGetClient(url, Headers())
	if err != nil {
		return "", err
	}

	var commits []Commit
	if err := json.Unmarshal(resp, &commits); err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found")
	}

	return commits[0].SHA, nil
}

func PagesBuildComplete(repo string, lastHash string) error {
	for i := 0; i < 24; i++ {
		time.Sleep(5 * time.Second)
		resp, err := HTTPGetClient(fmt.Sprintf("https://api.github.com/repos/%s/%s/pages/builds", os.Getenv("GITHUB_USER"), repo), Headers())
		if err != nil {
			return err
		}

		var builds []PagesBuild
		if err := json.Unmarshal(resp, &builds); err != nil {
			return err
		}

		if len(builds) == 0 {
			continue
		}

		latest := builds[0]
		if latest.Commit == lastHash && latest.Status == "built" {
			return nil
		}
	}

	return fmt.Errorf("pages_build_timeout")
}

func Round1(req UserRequest) error {
	err := InitGit()
	if err != nil {
		return err
	}

	name := req.Task

	repos, err := GetRepositories()
	if err != nil {
		return err
	}

	hasRepo := false

	for _, repo := range repos {
		if repo.Name == name {
			hasRepo = true
			break
		}
	}

	if hasRepo {
		if err := DeleteRepository(name); err != nil {
			return err
		}
	}

	if err := CreateRepository(name); err != nil {
		return err
	}

	if err := SetupRepo(name); err != nil {
		return err
	}

	vr := VibeRequest{
		Prompt:       req.Brief,
		Checks:       StringArrToString(req.Checks),
		Attachements: []VibeAttachement{},
	}

	for _, att := range req.Attachments {
		du, err := DecodeDataURL(att.URL)
		if err != nil {
			return fmt.Errorf("decode_data_url(%s): %w", att.Name, err)
		}

		dst := fmt.Sprintf("%s-%s", GenerateUUID(), att.Name)

		if err := CreateFileBytes(name, dst, du.Data, "feat: add attachment "+att.Name); err != nil {
			return fmt.Errorf("create_file_bytes(%s): %w", dst, err)
		}

		vr.Attachements = append(vr.Attachements, VibeAttachement{
			Filename: att.Name,
			URL:      fmt.Sprintf("./%s", dst),
		})
	}

	vibed, err := GenerateFrontend(vr)
	if err != nil {
		return err
	}

	for _, file := range *vibed {
		if err := CreateFile(name,
			file.Filename,
			file.Content,
			fmt.Sprintf("feat: add %s", file.Filename)); err != nil {
			return err
		}
	}

	lastHash, err := GetLastCommitHash(name)
	if err != nil {
		return err
	}

	if err := PagesBuildComplete(name, lastHash); err != nil {
		log.Printf("Pages build did not complete: %v", err)
	}

	lastHash, err = GetLastCommitHash(name)
	if err != nil {
		return err
	}

	evalReq := EvaluatorRequest{
		Email:     req.Email,
		Task:      req.Task,
		Round:     1,
		Nonce:     req.Nonce,
		RepoURL:   fmt.Sprintf("https://github.com/%s/%s", os.Getenv("GITHUB_USER"), name),
		CommitSHA: lastHash,
		PagesURL:  fmt.Sprintf("https://%s.github.io/%s/", os.Getenv("GITHUB_USER"), name),
	}

	if err := SatisfyEvaluator(evalReq, req.EvaluationURL); err != nil {
		return err
	}

	return nil
}

func Round2(req UserRequest) error {
	// Ensure repo exists
	name := req.Task

	// Load current bundle (and SHAs)
	readmeSHA, readmeContent, err := GetFileWithSHA(name, "README.md")
	if err != nil {
		return fmt.Errorf("get README.md: %w", err)
	}
	indexSHA, indexContent, err := GetFileWithSHA(name, "index.html")
	if err != nil {
		return fmt.Errorf("get index.html: %w", err)
	}

	existing := []VibeResponse{
		{Type: "markdown", Filename: "README.md", Content: readmeContent},
		{Type: "html", Filename: "index.html", Content: indexContent},
	}

	// Build VR (with attachments written to repo first, like Round1)
	vr := VibeRequest{
		Prompt:       req.Brief,
		Checks:       StringArrToString(req.Checks),
		Attachements: []VibeAttachement{},
	}

	for _, att := range req.Attachments {
		du, err := DecodeDataURL(att.URL)
		if err != nil {
			return fmt.Errorf("decode_data_url(%s): %w", att.Name, err)
		}
		dst := fmt.Sprintf("%s-%s", GenerateUUID(), att.Name)
		if err := CreateFileBytes(name, dst, du.Data, "feat: add attachment "+att.Name); err != nil {
			return fmt.Errorf("create_file_bytes(%s): %w", dst, err)
		}
		vr.Attachements = append(vr.Attachements, VibeAttachement{
			Filename: att.Name,
			URL:      "./" + dst,
		})
	}

	// Ask the model to modify based on the current files
	modified, err := ModifyFrontend(vr, existing)
	if err != nil {
		return err
	}
	files := *modified

	// Validate before committing
	if verrs := ValidateVibeBundle(files); len(verrs) > 0 {
		for _, e := range verrs {
			log.Printf("bundle validation error: %v", e)
		}
		return fmt.Errorf("bundle_validation_failed")
	}

	// Update files with SHAs
	shaMap := map[string]string{
		"README.md":  readmeSHA,
		"index.html": indexSHA,
	}
	for _, f := range files {
		oldSHA, ok := shaMap[f.Filename]
		if !ok {
			return fmt.Errorf("unexpected filename in round2: %s", f.Filename)
		}
		if err := UpdateFile(name, f.Filename, f.Content, fmt.Sprintf("chore: update %s for round 2", f.Filename), oldSHA); err != nil {
			return err
		}
	}

	// Notify evaluator
	lastHash, err := GetLastCommitHash(name)
	if err != nil {
		return err
	}

	// Optional: wait for Pages build again
	if err := PagesBuildComplete(name, lastHash); err != nil {
		log.Printf("Pages build did not complete (round2): %v", err)
	}

	lastHash, err = GetLastCommitHash(name)
	if err != nil {
		return err
	}

	evalReq := EvaluatorRequest{
		Email:     req.Email,
		Task:      req.Task,
		Round:     2,
		Nonce:     req.Nonce,
		RepoURL:   fmt.Sprintf("https://github.com/%s/%s", os.Getenv("GITHUB_USER"), name),
		CommitSHA: lastHash,
		PagesURL:  fmt.Sprintf("https://%s.github.io/%s/", os.Getenv("GITHUB_USER"), name),
	}

	if err := SatisfyEvaluator(evalReq, req.EvaluationURL); err != nil {
		return err
	}

	return nil
}
