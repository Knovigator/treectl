package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var SelectedProfile string
var BackendURLOverride string
var AppHostOverride string

var loginEmail string
var loginPassword string
var readPasswordFromStdin bool

var loginEndpointCandidates = []string{"/auth/sign_in", "/auth/signin"}
var builtInProfileOrder = []string{"dev", "staging", "prod"}

type authTokens struct {
	AccessToken string
	Client      string
	UID         string
	Expiry      string
}

type profileConfig struct {
	Name          string `json:"name"`
	BackendURL    string `json:"backend_url"`
	AppHost       string `json:"app_host"`
	AccessToken   string `json:"access_token,omitempty"`
	Client        string `json:"client,omitempty"`
	UID           string `json:"uid,omitempty"`
	Expiry        string `json:"expiry,omitempty"`
	CurrentUserID string `json:"current_user_id,omitempty"`
	ActiveSpaceID string `json:"active_space_id,omitempty"`
}

type loginBootstrap struct {
	Host        string `json:"host"`
	CurrentUser *struct {
		ID      string `json:"id"`
		SpaceID string `json:"space_id"`
	} `json:"currentUser"`
}

// LoginCmd represents the login command
var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your account",
	Long:  `Authenticate and log in to your tree account to access protected features.`,
	Run:   runLogin,
}

var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Inspect and switch treectl profiles",
	Long:  `List, inspect, and switch the saved treectl profiles.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	Run:   runProfileList,
}

var profileUseCmd = &cobra.Command{
	Use:   "use [profile]",
	Short: "Select the active profile",
	Args:  cobra.ExactArgs(1),
	Run:   runProfileUse,
}

var profileShowCmd = &cobra.Command{
	Use:   "show [profile]",
	Short: "Show the resolved profile configuration",
	Args:  cobra.MaximumNArgs(1),
	Run:   runProfileShow,
}

func init() {
	LoginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "Email address for login")
	LoginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password for login")
	LoginCmd.Flags().BoolVar(&readPasswordFromStdin, "password-stdin", false, "Read password from stdin")

	ProfileCmd.AddCommand(profileListCmd)
	ProfileCmd.AddCommand(profileUseCmd)
	ProfileCmd.AddCommand(profileShowCmd)
}

func runLogin(cmd *cobra.Command, args []string) {
	profileName := resolveProfileName()
	profile, err := resolveProfile(profileName)
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	email, err := resolveEmail()
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	password, err := resolvePassword()
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	tokens, err := performLogin(profile.BackendURL, email, password)
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	bootstrap, err := fetchBootstrap(profile.BackendURL, tokens)
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}

	profile.AccessToken = tokens.AccessToken
	profile.Client = tokens.Client
	profile.UID = tokens.UID
	profile.Expiry = tokens.Expiry
	profile.AppHost = normalizeAppHost(bootstrap.Host)
	profile.CurrentUserID = bootstrap.CurrentUser.ID
	profile.ActiveSpaceID = bootstrap.CurrentUser.SpaceID

	err = saveProfile(profile, true)
	if err != nil {
		fmt.Println("Error saving profile:", err)
		return
	}

	fmt.Printf("Login successful. Profile: %s Backend: %s\n", profile.Name, profile.BackendURL)
}

func runProfileList(cmd *cobra.Command, args []string) {
	activeProfile := resolveProfileName()
	profileNames := allProfileNames()

	for _, profileName := range profileNames {
		profile, err := resolveProfile(profileName)
		if err != nil {
			fmt.Printf("  %s (invalid: %v)\n", profileName, err)
			continue
		}

		marker := " "
		if profileName == activeProfile {
			marker = "*"
		}

		loginState := "signed-out"
		if profile.AccessToken != "" && profile.Client != "" && profile.UID != "" {
			loginState = "signed-in"
		}

		fmt.Printf("%s %s\t%s\t%s\n", marker, profile.Name, profile.BackendURL, loginState)
	}
}

func runProfileUse(cmd *cobra.Command, args []string) {
	profileName := normalizeProfileName(args[0])
	if _, err := resolveProfile(profileName); err != nil {
		fmt.Println("Error:", err)
		return
	}

	err := setActiveProfile(profileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Active profile set to %s\n", profileName)
}

func runProfileShow(cmd *cobra.Command, args []string) {
	profileName := resolveProfileName()
	if len(args) == 1 {
		profileName = normalizeProfileName(args[0])
	}

	profile, err := resolveProfile(profileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	redactedProfile := profile
	if redactedProfile.AccessToken != "" {
		redactedProfile.AccessToken = redactValue(redactedProfile.AccessToken)
	}

	prettyJSON, err := json.MarshalIndent(redactedProfile, "", "  ")
	if err != nil {
		fmt.Println("Error formatting profile:", err)
		return
	}

	fmt.Println(string(prettyJSON))
}

func builtInProfiles() map[string]profileConfig {
	return map[string]profileConfig{
		"dev": {
			Name:       "dev",
			BackendURL: "http://localhost:5001",
			AppHost:    "http://localhost:5173",
		},
		"staging": {
			Name:       "staging",
			BackendURL: "https://knov-staging-jajw.onrender.com",
			AppHost:    "https://staging-frontend-vi5w.onrender.com",
		},
		"prod": {
			Name:       "prod",
			BackendURL: "https://knov-prod.onrender.com",
			AppHost:    "https://prod-frontend-kitu.onrender.com",
		},
	}
}

func resolveProfileName() string {
	if strings.TrimSpace(SelectedProfile) != "" {
		return normalizeProfileName(SelectedProfile)
	}

	if profileName := strings.TrimSpace(os.Getenv("TREECTL_PROFILE")); profileName != "" {
		return normalizeProfileName(profileName)
	}

	if profileName := strings.TrimSpace(viper.GetString("active_profile")); profileName != "" {
		return normalizeProfileName(profileName)
	}

	return "dev"
}

func normalizeProfileName(profileName string) string {
	return strings.ToLower(strings.TrimSpace(profileName))
}

func resolveProfile(profileName string) (profileConfig, error) {
	resolvedProfileName := normalizeProfileName(profileName)
	profile := builtInProfiles()[resolvedProfileName]
	profile.Name = resolvedProfileName

	storedProfile := loadStoredProfile(resolvedProfileName)
	profile = mergeProfile(profile, storedProfile)

	backendURLOverride := strings.TrimSpace(BackendURLOverride)
	if backendURLOverride == "" {
		backendURLOverride = strings.TrimSpace(os.Getenv("TREECTL_BACKEND_URL"))
	}
	if backendURLOverride != "" {
		profile.BackendURL = normalizeBaseURL(backendURLOverride)
	}

	appHostOverride := strings.TrimSpace(AppHostOverride)
	if appHostOverride == "" {
		appHostOverride = strings.TrimSpace(os.Getenv("TREECTL_APP_HOST"))
	}
	if appHostOverride != "" {
		profile.AppHost = normalizeAppHost(appHostOverride)
	}

	if profile.BackendURL == "" {
		return profileConfig{}, fmt.Errorf("profile %q does not define a backend_url", resolvedProfileName)
	}

	profile.BackendURL = normalizeBaseURL(profile.BackendURL)
	profile.AppHost = normalizeAppHost(profile.AppHost)

	return profile, nil
}

func loadStoredProfile(profileName string) profileConfig {
	profileKeyPrefix := fmt.Sprintf("profiles.%s.", normalizeProfileName(profileName))

	return profileConfig{
		Name:          normalizeProfileName(profileName),
		BackendURL:    strings.TrimSpace(viper.GetString(profileKeyPrefix + "backend_url")),
		AppHost:       strings.TrimSpace(viper.GetString(profileKeyPrefix + "app_host")),
		AccessToken:   strings.TrimSpace(viper.GetString(profileKeyPrefix + "access_token")),
		Client:        strings.TrimSpace(viper.GetString(profileKeyPrefix + "client")),
		UID:           strings.TrimSpace(viper.GetString(profileKeyPrefix + "uid")),
		Expiry:        strings.TrimSpace(viper.GetString(profileKeyPrefix + "expiry")),
		CurrentUserID: strings.TrimSpace(viper.GetString(profileKeyPrefix + "current_user_id")),
		ActiveSpaceID: strings.TrimSpace(viper.GetString(profileKeyPrefix + "active_space_id")),
	}
}

func mergeProfile(base profileConfig, override profileConfig) profileConfig {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.BackendURL != "" {
		base.BackendURL = override.BackendURL
	}
	if override.AppHost != "" {
		base.AppHost = override.AppHost
	}
	if override.AccessToken != "" {
		base.AccessToken = override.AccessToken
	}
	if override.Client != "" {
		base.Client = override.Client
	}
	if override.UID != "" {
		base.UID = override.UID
	}
	if override.Expiry != "" {
		base.Expiry = override.Expiry
	}
	if override.CurrentUserID != "" {
		base.CurrentUserID = override.CurrentUserID
	}
	if override.ActiveSpaceID != "" {
		base.ActiveSpaceID = override.ActiveSpaceID
	}

	return base
}

func allProfileNames() []string {
	profileNames := append([]string{}, builtInProfileOrder...)
	profileMap := viper.GetStringMap("profiles")
	for profileName := range profileMap {
		normalizedProfileName := normalizeProfileName(profileName)
		if normalizedProfileName == "" || slices.Contains(profileNames, normalizedProfileName) {
			continue
		}

		profileNames = append(profileNames, normalizedProfileName)
	}

	slices.Sort(profileNames)
	return profileNames
}

func requireAuthenticatedProfile() (profileConfig, error) {
	profile, err := resolveProfile(resolveProfileName())
	if err != nil {
		return profileConfig{}, err
	}

	if profile.AccessToken == "" || profile.Client == "" || profile.UID == "" {
		return profileConfig{}, fmt.Errorf("missing credentials for profile %q; run treectl login --profile %s", profile.Name, profile.Name)
	}

	return profile, nil
}

func setActiveProfile(profileName string) error {
	profile, err := resolveProfile(profileName)
	if err != nil {
		return err
	}

	return saveProfile(profile, true)
}

func resolveEmail() (string, error) {
	if strings.TrimSpace(loginEmail) != "" {
		return strings.TrimSpace(loginEmail), nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading email: %w", err)
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}

	return email, nil
}

func resolvePassword() (string, error) {
	if readPasswordFromStdin {
		passwordBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("error reading password from stdin: %w", err)
		}

		password := strings.TrimSpace(string(passwordBytes))
		if password == "" {
			return "", fmt.Errorf("password cannot be empty")
		}

		return password, nil
	}

	if loginPassword != "" {
		return loginPassword, nil
	}

	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("error reading password: %w", err)
	}

	password := strings.TrimSpace(string(passwordBytes))
	fmt.Println()

	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	return password, nil
}

func performLogin(backendURL, email, password string) (authTokens, error) {
	client := resty.New()
	client.SetTimeout(10 * time.Second)

	formBody := fmt.Sprintf("email=%s&password=%s", url.QueryEscape(email), url.QueryEscape(password))
	var lastError error

	for _, endpoint := range loginEndpointCandidates {
		resp, err := client.R().
			SetHeader("Content-Type", "application/x-www-form-urlencoded").
			SetBody(formBody).
			Post(backendURL + endpoint)

		if err != nil {
			lastError = fmt.Errorf("error making request to %s: %w", endpoint, err)
			continue
		}

		if resp.StatusCode() == http.StatusNotFound {
			lastError = fmt.Errorf("%s returned 404", endpoint)
			continue
		}

		if resp.StatusCode() != http.StatusOK {
			return authTokens{}, fmt.Errorf("login failed: %s", formatResponseError(resp))
		}

		tokens := authTokens{
			AccessToken: strings.TrimSpace(resp.Header().Get("access-token")),
			Client:      strings.TrimSpace(resp.Header().Get("client")),
			UID:         strings.TrimSpace(resp.Header().Get("uid")),
			Expiry:      strings.TrimSpace(resp.Header().Get("expiry")),
		}

		if tokens.AccessToken == "" {
			return authTokens{}, fmt.Errorf("access token not found in response headers")
		}
		if tokens.Client == "" {
			return authTokens{}, fmt.Errorf("client not found in response headers")
		}
		if tokens.UID == "" {
			return authTokens{}, fmt.Errorf("uid not found in response headers")
		}

		return tokens, nil
	}

	if lastError != nil {
		return authTokens{}, lastError
	}

	return authTokens{}, fmt.Errorf("login failed: no auth endpoint succeeded")
}

func fetchBootstrap(backendURL string, tokens authTokens) (*loginBootstrap, error) {
	client := resty.New()
	client.SetTimeout(10 * time.Second)

	resp, err := client.R().
		SetHeader("access-token", tokens.AccessToken).
		SetHeader("client", tokens.Client).
		SetHeader("uid", tokens.UID).
		Get(backendURL + "/api/v1/gon")
	if err != nil {
		return nil, fmt.Errorf("error fetching bootstrap: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("bootstrap fetch failed: %s", formatResponseError(resp))
	}

	var bootstrap loginBootstrap
	if err := json.Unmarshal(resp.Body(), &bootstrap); err != nil {
		return nil, fmt.Errorf("error decoding bootstrap payload: %w", err)
	}

	if bootstrap.CurrentUser == nil || bootstrap.CurrentUser.ID == "" {
		return nil, fmt.Errorf("bootstrap payload did not include an authenticated currentUser")
	}

	return &bootstrap, nil
}

func saveProfile(profile profileConfig, setActive bool) error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return err
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")

	profileKeyPrefix := fmt.Sprintf("profiles.%s.", normalizeProfileName(profile.Name))
	viper.Set(profileKeyPrefix+"backend_url", normalizeBaseURL(profile.BackendURL))
	viper.Set(profileKeyPrefix+"app_host", normalizeAppHost(profile.AppHost))
	viper.Set(profileKeyPrefix+"access_token", profile.AccessToken)
	viper.Set(profileKeyPrefix+"client", profile.Client)
	viper.Set(profileKeyPrefix+"uid", profile.UID)
	viper.Set(profileKeyPrefix+"expiry", profile.Expiry)
	viper.Set(profileKeyPrefix+"current_user_id", profile.CurrentUserID)
	viper.Set(profileKeyPrefix+"active_space_id", profile.ActiveSpaceID)

	if setActive {
		viper.Set("active_profile", normalizeProfileName(profile.Name))
	}

	statErr := error(nil)
	if _, statErr = os.Stat(configPath); statErr == nil {
		return viper.WriteConfig()
	}

	if !os.IsNotExist(statErr) {
		return statErr
	}

	return viper.SafeWriteConfigAs(configPath)
}

func configFilePath() (string, error) {
	configPath := viper.ConfigFileUsed()
	if configPath != "" {
		return configPath, nil
	}

	return xdg.ConfigFile("treectl/config.toml")
}

func normalizeBaseURL(rawURL string) string {
	return strings.TrimRight(strings.TrimSpace(rawURL), "/")
}

func normalizeAppHost(rawHost string) string {
	trimmedHost := strings.TrimSpace(rawHost)
	if trimmedHost == "" {
		return ""
	}

	if strings.HasPrefix(trimmedHost, "http://") || strings.HasPrefix(trimmedHost, "https://") {
		return strings.TrimRight(trimmedHost, "/")
	}

	scheme := "https"
	if strings.Contains(trimmedHost, "localhost") || strings.Contains(trimmedHost, "127.0.0.1") {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s", scheme, strings.TrimRight(trimmedHost, "/"))
}

func formatResponseError(resp *resty.Response) string {
	body := strings.TrimSpace(string(resp.Body()))
	if body == "" {
		return fmt.Sprintf("status %d", resp.StatusCode())
	}

	var responseBody map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &responseBody); err == nil {
		formattedJSON, marshalErr := json.Marshal(responseBody)
		if marshalErr == nil {
			return fmt.Sprintf("status %d: %s", resp.StatusCode(), string(formattedJSON))
		}
	}

	return fmt.Sprintf("status %d: %s", resp.StatusCode(), body)
}

func redactValue(value string) string {
	if len(value) <= 8 {
		return "********"
	}

	return value[:4] + "..." + value[len(value)-4:]
}
