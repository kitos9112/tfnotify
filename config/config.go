package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/suzuki-shunsuke/go-ci-env/cienv"
	"github.com/suzuki-shunsuke/go-findconfig/findconfig"
	"gopkg.in/yaml.v2"
)

// Config is for tfnotify config structure
type Config struct {
	CI        string            `yaml:"ci"`
	Notifier  Notifier          `yaml:"notifier"`
	Terraform Terraform         `yaml:"terraform"`
	Vars      map[string]string `yaml:"-"`

	path string
}

// Notifier is a notification notifier
type Notifier struct {
	Github   GithubNotifier   `yaml:"github"`
	Gitlab   GitlabNotifier   `yaml:"gitlab"`
	Slack    SlackNotifier    `yaml:"slack"`
	Typetalk TypetalkNotifier `yaml:"typetalk"`
}

// GithubNotifier is a notifier for GitHub
type GithubNotifier struct {
	Token      string     `yaml:"token"`
	BaseURL    string     `yaml:"base_url"`
	Repository Repository `yaml:"repository"`
}

// GitlabNotifier is a notifier for GitLab
type GitlabNotifier struct {
	Token      string     `yaml:"token"`
	BaseURL    string     `yaml:"base_url"`
	Repository Repository `yaml:"repository"`
}

// Repository represents a GitHub repository
type Repository struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

// SlackNotifier is a notifier for Slack
type SlackNotifier struct {
	Token   string `yaml:"token"`
	Channel string `yaml:"channel"`
	Bot     string `yaml:"bot"`
}

// TypetalkNotifier is a notifier for Typetalk
type TypetalkNotifier struct {
	Token   string `yaml:"token"`
	TopicID string `yaml:"topic_id"`
}

// Terraform represents terraform configurations
type Terraform struct {
	Default      Default `yaml:"default"`
	Fmt          Fmt     `yaml:"fmt"`
	Plan         Plan    `yaml:"plan"`
	Apply        Apply   `yaml:"apply"`
	UseRawOutput bool    `yaml:"use_raw_output,omitempty"`
}

// Default is a default setting for terraform commands
type Default struct {
	Template string `yaml:"template"`
}

// Fmt is a terraform fmt config
type Fmt struct {
	Template string `yaml:"template"`
}

// Plan is a terraform plan config
type Plan struct {
	Template            string              `yaml:"template"`
	WhenAddOrUpdateOnly WhenAddOrUpdateOnly `yaml:"when_add_or_update_only,omitempty"`
	WhenDestroy         WhenDestroy         `yaml:"when_destroy,omitempty"`
	WhenNoChanges       WhenNoChanges       `yaml:"when_no_changes,omitempty"`
	WhenPlanError       WhenPlanError       `yaml:"when_plan_error,omitempty"`
}

// WhenAddOrUpdateOnly is a configuration to notify the plan result contains new or updated in place resources
type WhenAddOrUpdateOnly struct {
	Label string `yaml:"label,omitempty"`
	Color string `yaml:"label_color,omitempty"`
}

// WhenDestroy is a configuration to notify the plan result contains destroy operation
type WhenDestroy struct {
	Label    string `yaml:"label,omitempty"`
	Template string `yaml:"template,omitempty"`
	Color    string `yaml:"label_color,omitempty"`
}

// WhenNoChange is a configuration to add a label when the plan result contains no change
type WhenNoChanges struct {
	Label string `yaml:"label,omitempty"`
	Color string `yaml:"label_color,omitempty"`
}

// WhenPlanError is a configuration to notify the plan result returns an error
type WhenPlanError struct {
	Label string `yaml:"label,omitempty"`
	Color string `yaml:"label_color,omitempty"`
}

// Apply is a terraform apply config
type Apply struct {
	Template string `yaml:"template"`
}

// LoadFile binds the config file to Config structure
func (cfg *Config) LoadFile(path string) error {
	cfg.path = path
	_, err := os.Stat(cfg.path)
	if err != nil {
		return fmt.Errorf("%s: no config file", cfg.path)
	}
	raw, _ := ioutil.ReadFile(cfg.path)
	return yaml.Unmarshal(raw, cfg)
}

func (cfg *Config) Complement() {
	var platform cienv.Platform
	if cfg.CI == "" {
		platform = cienv.Get()
		if platform != nil {
			cfg.CI = platform.CI()
		}
	} else {
		platform = cienv.GetByName(cfg.CI)
	}
	if platform == nil {
		return
	}
	if cfg.isDefinedGithub() {
		if cfg.Notifier.Github.Repository.Owner == "" {
			cfg.Notifier.Github.Repository.Owner = platform.RepoOwner()
		}
		if cfg.Notifier.Github.Repository.Name == "" {
			cfg.Notifier.Github.Repository.Name = platform.RepoName()
		}
	}
}

// Validation validates config file
func (cfg *Config) Validation() error {
	switch strings.ToLower(cfg.CI) {
	case "":
		return errors.New("ci: need to be set")
	case "circleci", "circle-ci":
		// ok pattern
	case "gitlabci", "gitlab-ci":
		// ok pattern
	case "travis", "travisci", "travis-ci":
		// ok pattern
	case "codebuild":
		// ok pattern
	case "teamcity":
		// ok pattern
	case "drone":
		// ok pattern
	case "jenkins":
		// ok pattern
	case "github-actions":
		// ok pattern
	case "cloud-build", "cloudbuild":
		// ok pattern
	default:
		return fmt.Errorf("%s: not supported yet", cfg.CI)
	}
	if cfg.isDefinedGithub() {
		platform := cienv.GetByName(cfg.CI)

		if platform != nil {
			if cfg.Notifier.Github.Repository.Owner == "" {
				cfg.Notifier.Github.Repository.Owner = platform.RepoOwner()
			}
			if cfg.Notifier.Github.Repository.Name == "" {
				cfg.Notifier.Github.Repository.Name = platform.RepoName()
			}
		}

		if cfg.Notifier.Github.Repository.Owner == "" {
			return errors.New("repository owner is missing")
		}
		if cfg.Notifier.Github.Repository.Name == "" {
			return errors.New("repository name is missing")
		}
	}
	if cfg.isDefinedGitlab() {
		if cfg.Notifier.Gitlab.Repository.Owner == "" {
			return errors.New("repository owner is missing")
		}
		if cfg.Notifier.Gitlab.Repository.Name == "" {
			return errors.New("repository name is missing")
		}
	}
	if cfg.isDefinedSlack() {
		if cfg.Notifier.Slack.Channel == "" {
			return errors.New("slack channel id is missing")
		}
	}
	if cfg.isDefinedTypetalk() {
		if cfg.Notifier.Typetalk.TopicID == "" {
			return errors.New("Typetalk topic id is missing") //nolint:stylecheck
		}
	}
	notifier := cfg.GetNotifierType()
	if notifier == "" {
		return errors.New("notifier is missing")
	}
	return nil
}

func (cfg *Config) isDefinedGithub() bool {
	// not empty
	return cfg.Notifier.Github != (GithubNotifier{})
}

func (cfg *Config) isDefinedGitlab() bool {
	// not empty
	return cfg.Notifier.Gitlab != (GitlabNotifier{})
}

func (cfg *Config) isDefinedSlack() bool {
	// not empty
	return cfg.Notifier.Slack != (SlackNotifier{})
}

func (cfg *Config) isDefinedTypetalk() bool {
	// not empty
	return cfg.Notifier.Typetalk != (TypetalkNotifier{})
}

// GetNotifierType return notifier type described in Config
func (cfg *Config) GetNotifierType() string {
	if cfg.isDefinedGithub() {
		return "github"
	}
	if cfg.isDefinedGitlab() {
		return "gitlab"
	}
	if cfg.isDefinedSlack() {
		return "slack"
	}
	if cfg.isDefinedTypetalk() {
		return "typetalk"
	}
	return ""
}

// Find returns config path
func (cfg *Config) Find(file string) (string, error) {
	if file != "" {
		if _, err := os.Stat(file); err == nil {
			return file, nil
		}
		return "", errors.New("config for tfnotify is not found at all")
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get a current directory path: %w", err)
	}
	if p := findconfig.Find(wd, findconfig.Exist, "tfnotify.yaml", "tfnotify.yml", ".tfnotify.yaml", ".tfnotify.yml"); p != "" {
		return p, nil
	}
	return "", errors.New("config for tfnotify is not found at all")
}
