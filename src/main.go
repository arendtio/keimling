package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	Model, CompactionModel, Mode, Key, SessionPath, VoiceModel, ReviewModel, ApprovalPolicy, Reasoning, BaseURL string
	DelegateFast, DelegateBalanced, DelegateDeep, PilotModel, PilotReasoning                                    string
	Autopilot                                                                                                   bool
	ExternalClient                                                                                              *Client
	Web                                                                                                         bool
	MaxSteps, ContextLimit                                                                                      int
}

func apiKey() (string, error) {
	if s := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); s != "" {
		return s, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(home, ".credentials_openai")
	st, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("API key not found: OPENAI_API_KEY or %s", p)
	}
	if st.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("insecure file permissions for %s: chmod 600 %s", p, p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(b))
	if key == "" {
		return "", fmt.Errorf("API key file is empty")
	}
	return key, nil
}

type fileConfig struct {
	API struct {
		BaseURL string `json:"base_url"`
		Mode    string `json:"mode"`
		Model   string `json:"model"`
		KeyFile string `json:"key_file"`
	} `json:"api"`
	OpenAI struct {
		KeyFile string `json:"key_file"`
	} `json:"openai"`
	Features struct {
		WebSearch               *bool `json:"web_search"`
		DisableExternalServices bool  `json:"disable_external_services"`
	} `json:"features"`
	Presentation string `json:"presentation"`
	Listen       string `json:"listen"`
	Connect      string `json:"connect"`
	AccessSecret string `json:"access_secret"`
}

func readConfiguration(path string) (fileConfig, error) {
	var cfg fileConfig
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return cfg, nil
	}
	if e != nil {
		return cfg, e
	}
	e = json.Unmarshal(b, &cfg)
	return cfg, e
}
func readKeyFile(path string) (string, error) {
	st, e := os.Stat(path)
	if e != nil {
		return "", e
	}
	if st.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("insecure key-file permissions: chmod 600 %s", path)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	return strings.TrimSpace(string(b)), nil
}
func normalizeBaseURL(raw string) (string, error) {
	u, e := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if e != nil {
		return "", e
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid API base URL")
	}
	if u.Path == "" {
		u.Path = "/v1"
	}
	return strings.TrimRight(u.String(), "/"), nil
}
func main() {
	configPath := flag.String("config", "", "JSON configuration path (default ~/.config/keimling/config.json)")
	baseURL := flag.String("base-url", "", "Responses API base URL, e.g. http://localhost:8080/v1")
	keyFile := flag.String("api-key-file", "", "Model provider API key file (optional for custom providers)")
	openaiKeyFile := flag.String("openai-api-key-file", "", "OpenAI key file for external web search and transcription")
	disableExternal := flag.Bool("disable-external-services", false, "Disable OpenAI web search and transcription")
	checkModel := flag.Bool("check-model", true, "Check the model identifier against /models at startup")
	apiMode := flag.String("api-mode", "responses", "Only responses is supported")
	model := flag.String("model", "gpt-6-luna", "OpenAI model (Responses API)")
	compactionModel := flag.String("compaction-model", "", "Optional model for compaction (uses same API endpoint and credentials)")
	reviewModel := flag.String("review-model", "gpt-6-luna", "Model for automatic approval review")
	approvalPolicy := flag.String("approval-policy", "", "Optional path to additional approval policy text")
	reasoning := flag.String("reasoning", "high", "Reasoning effort: low, medium, high, xhigh, max")
	delegateFast := flag.String("delegate-fast-model", "gpt-6-luna", "Fast delegate model")
	delegateBalanced := flag.String("delegate-balanced-model", "gpt-6.1-sol", "Balanced delegate model")
	delegateDeep := flag.String("delegate-deep-model", "gpt-6.1-sol", "Deep delegate model (high reasoning)")
	pilot := flag.Bool("autopilot", false, "Enable autonomous task supervisor")
	pilotModel := flag.String("autopilot-model", "gpt-6-luna", "Autopilot supervisor model")
	pilotReasoning := flag.String("autopilot-reasoning", "high", "Autopilot reasoning effort")
	mode := flag.String("mode", "auto", "ask, auto, or yolo")
	web := flag.Bool("web", true, "Enable OpenAI web search")
	ctxLimit := flag.Int("context-limit", 128000, "Context window token budget for two-stage compaction")
	steps := flag.Int("max-steps", 200, "Maximum API rounds per user task")
	session := flag.String("session", "", "Session file (JSONL)")
	resume := flag.Bool("resume", true, "Resume last session (default: true)")
	voiceModel := flag.String("voice-model", "gpt-transcribe", "Transcription model")
	uiMode := flag.Bool("ui", false, "Force native WebView UI (fail if unavailable)")
	cliMode := flag.Bool("cli", false, "Force shell client")
	serverMode := flag.Bool("server", false, "Run HTTP agent server only")
	listenAddr := flag.String("listen", "", "HTTP listen address (required with --server), e.g. 0.0.0.0:7420")
	connectURL := flag.String("connect", "", "Connect to remote Keimling agent base URL")
	accessSecret := flag.String("access-secret", "", "Shared secret for agent HTTP API/UI")
	flag.Parse()
	if *apiMode != "responses" {
		fmt.Fprintln(os.Stderr, "only --api-mode responses is supported")
		os.Exit(2)
	}
	home, _ := os.UserHomeDir()
	if *configPath == "" {
		*configPath = filepath.Join(home, ".config", "keimling", "config.json")
	}
	fc, e := readConfiguration(*configPath)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	changed := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { changed[f.Name] = true })
	if !changed["listen"] && fc.Listen != "" {
		*listenAddr = fc.Listen
	}
	if !changed["connect"] && fc.Connect != "" {
		*connectURL = fc.Connect
	}
	if !changed["access-secret"] && fc.AccessSecret != "" {
		*accessSecret = fc.AccessSecret
	}
	presentation, perr := resolvePresentation(fc.Presentation, *uiMode, *cliMode, *serverMode)
	if perr != nil {
		fmt.Fprintln(os.Stderr, perr)
		os.Exit(2)
	}
	if presentation == presentationServer {
		if err := validateServerListen(*listenAddr, *accessSecret); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if strings.TrimSpace(*connectURL) != "" {
		if presentation == presentationServer {
			fmt.Fprintln(os.Stderr, "--connect cannot be combined with --server")
			os.Exit(2)
		}
		wantGUI := presentation == presentationUI || presentation == presentationAuto
		if wantGUI {
			if err := runGUIRemote(*connectURL, *accessSecret); err != nil {
				if presentation == presentationUI {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Fprintln(os.Stderr, "GUI unavailable, falling back to CLI:", err)
			} else {
				return
			}
		}
		if err := runRemoteCLI(*connectURL, *accessSecret); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if !changed["base-url"] && fc.API.BaseURL != "" {
		*baseURL = fc.API.BaseURL
	}
	if !changed["model"] && fc.API.Model != "" {
		*model = fc.API.Model
	}
	if !changed["api-key-file"] && fc.API.KeyFile != "" {
		*keyFile = fc.API.KeyFile
	}
	if !changed["openai-api-key-file"] && fc.OpenAI.KeyFile != "" {
		*openaiKeyFile = fc.OpenAI.KeyFile
	}
	if !changed["mode"] && fc.API.Mode != "" {
		*mode = fc.API.Mode
	}
	custom := strings.TrimSpace(*baseURL) != "" && strings.TrimRight(*baseURL, "/") != defaultBaseURL
	if *baseURL == "" {
		*baseURL = defaultBaseURL
	}
	resolvedURL, e := normalizeBaseURL(*baseURL)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	custom = resolvedURL != defaultBaseURL
	parsedBase, _ := url.Parse(resolvedURL)
	if parsedBase.Scheme == "http" && parsedBase.Hostname() != "localhost" && parsedBase.Hostname() != "127.0.0.1" && parsedBase.Hostname() != "::1" {
		fmt.Fprintln(os.Stderr, "WARNING: remote HTTP API endpoint is not encrypted; API keys and prompts may be exposed. Prefer HTTPS.")
	}
	if !changed["web"] && fc.Features.WebSearch != nil {
		*web = *fc.Features.WebSearch
	}
	if !changed["disable-external-services"] {
		*disableExternal = fc.Features.DisableExternalServices
	}
	if *disableExternal {
		*web = false
	}
	if *mode != "ask" && *mode != "auto" && *mode != "yolo" {
		fmt.Fprintln(os.Stderr, "--mode must be ask, auto, or yolo")
		os.Exit(2)
	}
	if *steps < 1 {
		fmt.Fprintln(os.Stderr, "--max-steps must be positive")
		os.Exit(2)
	}
	policyText := ""
	if *approvalPolicy != "" {
		pb, e := os.ReadFile(*approvalPolicy)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(2)
		}
		policyText = string(pb)
	}
	key := ""
	var err error
	if *keyFile != "" {
		key, err = readKeyFile(*keyFile)
	} else if !custom {
		key, err = apiKey()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// External OpenAI credentials are deliberately independent of the model provider.
	// A custom provider never receives this key.
	externalKey := ""
	if !*disableExternal {
		if *openaiKeyFile != "" {
			externalKey, err = readKeyFile(*openaiKeyFile)
		} else if !custom && key != "" {
			externalKey = key
		} else {
			externalKey, err = apiKey()
		}
		if err != nil {
			// Continue without optional services. Missing credentials are not fatal.
			fmt.Fprintln(os.Stderr, "WARNING: OpenAI external services unavailable:", err)
			externalKey = ""
			err = nil
		}
		if externalKey == "" {
			*web = false
		}
	}
	externalClient := NewClient(externalKey)
	if *session == "" {
		*session = filepath.Join(home, ".local", "state", "keimling", "session.jsonl")
	}
	journal, err := OpenJournal(*session, *resume)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer journal.Close()
	c := Config{Model: *model, CompactionModel: *compactionModel, Mode: *mode, Key: key, SessionPath: *session, VoiceModel: *voiceModel, Web: *web, MaxSteps: *steps, ContextLimit: *ctxLimit, ReviewModel: *reviewModel, ApprovalPolicy: policyText, Reasoning: *reasoning, BaseURL: resolvedURL, DelegateFast: *delegateFast, DelegateBalanced: *delegateBalanced, DelegateDeep: *delegateDeep, Autopilot: *pilot, ExternalClient: externalClient, PilotModel: *pilotModel, PilotReasoning: *pilotReasoning}
	a := &Agent{Cfg: c, Client: NewClient(key), Journal: journal, History: journal.History}
	a.Client.BaseURL = resolvedURL
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	a.SessionCtx = sessionCtx
	defer sessionCancel()
	a.LastTaskMessage, a.ResumeAvailable = pendingTask(c.SessionPath)
	if *disableExternal {
		a.Cfg.ExternalClient = nil
	}
	if *checkModel {
		if checkErr := performStartupCheck(a.Client, c.Model); checkErr != nil {
			if strings.Contains(checkErr.Error(), "not found in provider catalog") {
				fmt.Fprintln(os.Stderr, "Model validation failed:", checkErr)
				os.Exit(2)
			}
			fmt.Fprintln(os.Stderr, "WARNING:", checkErr, "(continuing; use --check-model=false to disable check)")
		} else {
			fmt.Println("Model validated:", c.Model)
		}
	}
	if presentation == presentationServer {
		if err := runAppServer(*listenAddr, *accessSecret, a); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if presentation == presentationUI || presentation == presentationAuto {
		if err := runGUILocal(a, *accessSecret); err != nil {
			if presentation == presentationUI {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "GUI unavailable, falling back to CLI:", err)
		} else {
			return
		}
	}
	input := bufio.NewReader(os.Stdin)
	fmt.Printf("keimling | %s | mode=%s | web=%v\nSession: %s\nF1/Space/Enter on empty prompt: record; F1/Space/Enter: stop; Enter: send draft; /newchat, /mode, /status, /help, /exit\n", c.Model, c.Mode, c.Web, c.SessionPath)
	for {
		discardPendingInput()
		line, err := readInteractiveLine(input, a.Cfg.ExternalClient, c.VoiceModel)
		if err != nil {
			fmt.Println()
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "/exit" || line == "/quit" {
			return
		}
		if line == "/help" {
			printHelp()
			continue
		}
		if line == "/mode" {
			fmt.Println("Mode:", a.Cfg.Mode, "(use /mode ask|auto|yolo)")
			continue
		}
		if strings.HasPrefix(line, "/mode ") {
			m := strings.TrimSpace(strings.TrimPrefix(line, "/mode "))
			if m == "ask" || m == "auto" || m == "yolo" {
				a.Cfg.Mode = m
				fmt.Println("mode:", m)
			} else {
				fmt.Println("ask, auto, or yolo")
			}
			continue
		}
		if line == "/compact" || strings.HasPrefix(line, "/compact ") {
			level := "hard"
			extra := strings.TrimSpace(strings.TrimPrefix(line, "/compact"))
			if extra == "soft" {
				fmt.Println("Soft compaction removed; use /compact")
				continue
			}
			ctx, cancel := context.WithCancel(context.Background())
			err := a.compactContext(ctx, level, extra)
			cancel()
			if err != nil {
				fmt.Println("Compaction:", err)
			}
			continue
		}
		if line == "/autopilot" || strings.HasPrefix(line, "/autopilot ") {
			setAutopilot(a, strings.TrimSpace(strings.TrimPrefix(line, "/autopilot")))
			continue
		}
		if line == "/status" {
			printStatus(a)
			continue
		}
		if line == "/newchat" {
			if err := a.NewChat(); err != nil {
				fmt.Println("New chat:", err)
			} else {
				fmt.Println("New session started. Previous session archived.")
			}
			continue
		}
		if strings.HasPrefix(line, "/transcribe ") {
			fmt.Println("Use F1 or /voice to record audio.")
			continue
		}
		if line == "/resume" || line == "/continue" || line == "/continue-new-model" {
			if !a.ResumeAvailable {
				fmt.Println("No unfinished task to resume.")
				continue
			}
			if line == "/continue-new-model" {
				removed, err := a.prepareNewModelContinuation()
				if err != nil {
					fmt.Println("Model handoff:", err)
					continue
				}
				fmt.Printf("Model handoff: portable context ready (%d reasoning blocks excluded; original journal preserved).\n", removed)
			}
			line = "" // resume without writing a synthetic user message
		} else if strings.HasPrefix(line, "/") {
			fmt.Println("Unknown command")
			continue
		}
		signalCtx, signalStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		ctx, cancel := context.WithCancel(signalCtx)
		control := newTaskControl(cancel)
		if line != "" {
			a.LastTaskMessage = line
		}
		a.ResumeAvailable = true
		a.Control = control
		activeControl = control
		go func() { control.done <- a.Run(ctx, line, input); close(control.finished) }()
		// One UI reader at a time. When an approval or PTY needs stdin,
		// the editor yields ownership and is recreated afterwards.
		running := true
		for running {
			select {
			case err = <-control.done:
				running = false
				continue
			default:
			}
			text, readErr := readInteractiveLine(input, a.Cfg.ExternalClient, c.VoiceModel)
			if readErr == errAgentIdle {
				continue
			}
			if readErr == errApprovalPending {
				req := <-control.approvals
				answer := readApproval(ctx)
				control.state.Store(0)
				req.response <- answer
				continue
			}
			if readErr == errUIHandoff {
				// Wait until foreground interaction releases the terminal.
				for control.state.Load() != 0 {
					time.Sleep(25 * time.Millisecond)
				}
				continue
			}
			if readErr != nil {
				cancel()
				err = readErr
				running = false
				break
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if text == "/autopilot" || strings.HasPrefix(text, "/autopilot ") {
				setAutopilot(a, strings.TrimSpace(strings.TrimPrefix(text, "/autopilot")))
				continue
			}
			if text == "/stop" {
				cancel()
				fmt.Println("Stop requested")
				continue
			}
			if text == "/compact" || strings.HasPrefix(text, "/compact ") {
				guidance := strings.TrimSpace(strings.TrimPrefix(text, "/compact"))
				if guidance == "soft" {
					fmt.Println("Soft compaction removed; use /compact")
					continue
				}
				select {
				case control.events <- controlEvent{kind: "compact", text: guidance}:
					fmt.Println("Compaction queued")
				default:
					fmt.Println("Steering queue is full")
				}
				continue
			}
			select {
			case control.events <- controlEvent{kind: "steer", text: text}:
				fmt.Println("Steering queued")
			default:
				fmt.Println("Steering queue is full")
			}
		}
		cancel()
		signalStop()
		activeControl = nil
		a.Control = nil
		if a.TaskDone || a.TaskBlocked || (err == nil && !a.TaskActive) {
			a.ResumeAvailable = false
		}
		if err != nil && err != context.Canceled {
			fmt.Fprintln(os.Stderr, "Agent:", err)
		}
	}
}

func printHelp() {
	fmt.Println("\n\033[1mCommands\033[0m")
	fmt.Println("  /help             Show this help")
	fmt.Println("  /status           Show model, context estimate, tokens and settings")
	fmt.Println("  /mode             Show current approval mode")
	fmt.Println("  /mode auto        Review commands; ask before sensitive changes")
	fmt.Println("  /mode ask         Confirm changing commands")
	fmt.Println("  /mode yolo        Run without approvals (unsafe)")
	fmt.Println("  /compact          Semantic context compaction now")
	fmt.Println("  /compact soft     Deprecated (use /compact)")
	fmt.Println("  /compact <text>   Hard compaction with extra guidance")
	fmt.Println("  /newchat          Archive conversation and start fresh")
	fmt.Println("  /stop             Cancel current work without exiting")
	fmt.Println("  /resume           Resume unfinished task without a new user message")
	fmt.Println("  /continue         Alias for /resume")
	fmt.Println("  /continue-new-model Resume using portable history without encrypted thinking")
	fmt.Println("  /autopilot on|off Enable/disable task supervision (default off)")
	fmt.Println("  Ctrl+G            Toggle autopilot while typing (PTY excluded)")
	fmt.Println("  During work: type a message to steer the next step")
	fmt.Println("  /exit             Quit")
	fmt.Println("\n\033[1mVoice\033[0m")
	fmt.Println("  F1/Space/Enter on empty prompt: start recording")
	fmt.Println("  Enter: transcribe and send | F1/Space: transcribe to draft")
	fmt.Println("\n\033[1mInput\033[0m")
	fmt.Println("  Arrow keys: move cursor/history | Home/End: line boundaries | Tab: complete slash commands")
}
func printStatus(a *Agent) {
	estimated := a.contextOccupancyEstimate()
	statusKind := "heuristic"
	if a.contextUsageCalibrated() {
		statusKind = "usage-calibrated"
	}
	pct := 0.0
	if a.Cfg.ContextLimit > 0 {
		pct = 100 * float64(estimated) / float64(a.Cfg.ContextLimit)
	}
	fmt.Printf("\n\033[1mSession\033[0m\n  Model: %s | Review: %s\n  Mode: %s | Autopilot: %v | Web: %v | Max steps: %d\n", a.Cfg.Model, a.Cfg.ReviewModel, a.Cfg.Mode, a.Cfg.Autopilot, a.Cfg.Web, a.Cfg.MaxSteps)
	fmt.Printf("  Compaction: model %s | endpoint %s\n", a.compactionModel(), a.Client.BaseURL)
	fmt.Printf("  Context: ~%d / %d tokens (%.1f%%, %s)\n", estimated, a.Cfg.ContextLimit, pct, statusKind)
	if u := a.Client.LastMainUsage; u.Available {
		actualPct := 0.0
		if a.Cfg.ContextLimit > 0 {
			actualPct = 100 * float64(u.InputTokens) / float64(a.Cfg.ContextLimit)
		}
		delta := int64(u.EstimatedBefore) - u.InputTokens
		fmt.Printf("  Last main request: input %d / %d (%.1f%% of configured limit) | output %d\n", u.InputTokens, a.Cfg.ContextLimit, actualPct, u.OutputTokens)
		fmt.Printf("  Details: cached input %d | reasoning output %d\n", u.CachedTokens, u.ReasoningTokens)
		fmt.Printf("  Estimate at request: ~%d | delta vs measured input: %+d tokens\n", u.EstimatedBefore, delta)
	} else {
		fmt.Println("  Last main request: not measured yet (or provider supplied no usage)")
	}
	fmt.Printf("  Tokens billed/reported this process: input %d, output %d, total %d\n", a.Client.InputTokens, a.Client.OutputTokens, a.Client.InputTokens+a.Client.OutputTokens)
	fmt.Printf("  History items: %d | Transcription: %s\n\n", len(a.History), a.Cfg.VoiceModel)
}

func setAutopilot(a *Agent, arg string) {
	switch arg {
	case "on":
		a.Cfg.Autopilot = true
	case "off":
		a.Cfg.Autopilot = false
	case "toggle":
		a.Cfg.Autopilot = !a.Cfg.Autopilot
	case "":
	default:
		fmt.Println("Usage: /autopilot on|off")
		return
	}
	fmt.Printf("Autopilot: %v\n", a.Cfg.Autopilot)
}
