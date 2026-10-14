package ui

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rancher-logs/api"
	"rancher-logs/config"
	"rancher-logs/logger"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	logTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			PaddingLeft(1)
	errorLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	normalStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	footerStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1)
	searchHighlight = lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("0")).
			Bold(true)
	searchInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)
	savedMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
)

type logLineMsg string
type logErrorMsg string
type logDoneMsg struct{}

type LogModel struct {
	viewport    viewport.Model
	lines       []string
	rawLines    []string
	title       string
	podName     string
	namespace   string
	reader      io.ReadCloser
	scanner     *bufio.Scanner
	ready       bool
	err         string
	width       int
	height      int
	xOffset     int
	jsonMode    bool
	searching   bool
	searchQuery string
	searchHits  []int
	searchIdx   int
	statusMsg   string
	maxLines    int
	cfg         *config.Config
	client      *api.Client
}

func NewLogModel(podName, namespace string, reader io.ReadCloser, cfg *config.Config, client *api.Client) LogModel {
	sc := bufio.NewScanner(reader)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	maxLines := cfg.MaxLines
	if maxLines < 200 || maxLines > 5000 {
		maxLines = config.DefaultMaxLines
	}
	return LogModel{
		title:     fmt.Sprintf("📋 Logs: %s/%s", namespace, podName),
		podName:   podName,
		namespace: namespace,
		reader:    reader,
		scanner:   sc,
		lines:     make([]string, 0, maxLines),
		rawLines:  make([]string, 0, maxLines),
		maxLines:  maxLines,
		cfg:       cfg,
		client:    client,
	}
}

func (m LogModel) Init() tea.Cmd {
	return m.readNextLine()
}

func (m LogModel) readNextLine() tea.Cmd {
	sc := m.scanner
	return func() tea.Msg {
		if sc.Scan() {
			return logLineMsg(sc.Text())
		}
		if err := sc.Err(); err != nil {
			logger.Errorf("log read error: %v", err)
			return logErrorMsg(fmt.Sprintf("Read error: %v", err))
		}
		return logDoneMsg{}
	}
}

func colorize(line string) string {
	upper := strings.ToUpper(line)
	if strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERR") ||
		strings.Contains(upper, "FATAL") || strings.Contains(upper, "EXCEPTION") {
		return errorLineStyle.Render(line)
	}
	if strings.Contains(upper, "WARN") || strings.Contains(upper, "WARNING") {
		return warnLineStyle.Render(line)
	}
	return normalStyle.Render(line)
}

func (m *LogModel) refreshViewport() {
	var content []string

	if m.searchQuery == "" {
		content = m.lines
	} else {
		content = make([]string, len(m.rawLines))
		query := strings.ToLower(m.searchQuery)
		m.searchHits = m.searchHits[:0]

		for i, raw := range m.rawLines {
			if strings.Contains(strings.ToLower(raw), query) {
				m.searchHits = append(m.searchHits, i)
				content[i] = searchHighlight.Render(raw)
			} else {
				content[i] = m.lines[i]
			}
		}
	}

	if m.xOffset > 0 {
		shifted := make([]string, len(content))
		for i, line := range content {
			shifted[i] = shiftLeft(line, m.xOffset)
		}
		m.viewport.SetContent(strings.Join(shifted, "\n"))
	} else {
		m.viewport.SetContent(strings.Join(content, "\n"))
	}
}

func shiftLeft(s string, offset int) string {
	runes := []rune(s)
	if offset >= len(runes) {
		return ""
	}
	return string(runes[offset:])
}

func (m *LogModel) jumpToHit() {
	if len(m.searchHits) == 0 {
		return
	}
	if m.searchIdx >= len(m.searchHits) {
		m.searchIdx = 0
	}
	if m.searchIdx < 0 {
		m.searchIdx = len(m.searchHits) - 1
	}
	targetLine := m.searchHits[m.searchIdx]
	vpHeight := m.viewport.Height
	offset := targetLine - vpHeight/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.SetYOffset(offset)
}

func prettyJSON(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return s, false
	}
	// Find first '{' or '[' to handle lines with prefix (e.g. timestamp + json)
	idxObj := strings.Index(s, "{")
	idxArr := strings.Index(s, "[")
	idx := -1
	if idxObj >= 0 && (idxArr < 0 || idxObj < idxArr) {
		idx = idxObj
	} else if idxArr >= 0 {
		idx = idxArr
	}
	if idx < 0 {
		return s, false
	}

	prefix := s[:idx]
	jsonPart := s[idx:]

	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(jsonPart), "", "  "); err != nil {
		return s, false
	}
	if prefix != "" {
		return prefix + buf.String(), true
	}
	return buf.String(), true
}

func (m *LogModel) formatLine(raw string) string {
	if m.jsonMode {
		if pretty, ok := prettyJSON(raw); ok {
			return colorize(pretty)
		}
	}
	return colorize(raw)
}

func (m *LogModel) rebuildLines() {
	m.lines = make([]string, len(m.rawLines))
	for i, raw := range m.rawLines {
		m.lines[i] = m.formatLine(raw)
	}
}

func (m *LogModel) saveLogsToFile() string {
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	dir := filepath.Dir(exe)

	timestamp := time.Now().Format("20060102_150405")
	ext := "txt"
	if m.jsonMode {
		ext = "json"
	}
	filename := fmt.Sprintf("logs_%s_%s_%s.%s", m.namespace, m.podName, timestamp, ext)
	filePath := filepath.Join(dir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		logger.Errorf("failed to create log file: %v", err)
		return fmt.Sprintf("Error saving: %v", err)
	}
	defer f.Close()

	// Fetch complete log from API (all lines, not just buffered)
	if m.client != nil {
		allReader, fetchErr := m.client.FetchAllLogs(m.namespace, m.podName)
		if fetchErr == nil {
			defer allReader.Close()
			sc := bufio.NewScanner(allReader)
			sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
			count := 0
			for sc.Scan() {
				raw := sc.Text()
				if m.jsonMode {
					if pretty, ok := prettyJSON(raw); ok {
						fmt.Fprintln(f, pretty)
					} else {
						fmt.Fprintln(f, raw)
					}
				} else {
					fmt.Fprintln(f, raw)
				}
				count++
			}
			logger.Info(fmt.Sprintf("Logs saved to %s (%d lines, complete)", filePath, count))
			return fmt.Sprintf("✓ Saved (complete): %s (%d lines)", filename, count)
		}
		logger.Errorf("FetchAllLogs failed, falling back to buffer: %v", fetchErr)
	}

	// Fallback: save buffered lines
	for _, raw := range m.rawLines {
		if m.jsonMode {
			if pretty, ok := prettyJSON(raw); ok {
				fmt.Fprintln(f, pretty)
			} else {
				fmt.Fprintln(f, raw)
			}
		} else {
			fmt.Fprintln(f, raw)
		}
	}

	logger.Info(fmt.Sprintf("Logs saved to %s (%d lines, buffer)", filePath, len(m.rawLines)))
	return fmt.Sprintf("✓ Saved: %s (%d lines)", filename, len(m.rawLines))
}

func (m LogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 3
		footerHeight := 2
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
		return m, nil

	case logLineMsg:
		raw := string(msg)
		m.rawLines = append(m.rawLines, raw)
		m.lines = append(m.lines, m.formatLine(raw))
		m.refreshViewport()
		if m.searchQuery == "" {
			m.viewport.GotoBottom()
		}
		return m, m.readNextLine()

	case logErrorMsg:
		m.err = string(msg)
		return m, nil

	case logDoneMsg:
		m.rawLines = append(m.rawLines, "--- Stream ended ---")
		m.lines = append(m.lines, footerStyle.Render("--- Stream ended ---"))
		m.refreshViewport()
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "enter":
				m.searching = false
				m.refreshViewport()
				if len(m.searchHits) > 0 {
					m.searchIdx = 0
					m.jumpToHit()
					m.statusMsg = fmt.Sprintf("Found %d matches (n/N: next/prev)", len(m.searchHits))
				} else {
					m.statusMsg = "No matches found"
				}
				return m, nil
			case "esc":
				m.searching = false
				m.searchQuery = ""
				m.searchHits = nil
				m.statusMsg = ""
				m.refreshViewport()
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
				}
				return m, nil
			}
		}

		m.statusMsg = ""

		switch msg.String() {
		case "ctrl+c":
			if m.reader != nil {
				m.reader.Close()
			}
			return m, tea.Quit
		case "q", "esc":
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.searchHits = nil
				m.statusMsg = ""
				m.refreshViewport()
				return m, nil
			}
			if m.reader != nil {
				m.reader.Close()
			}
			return m, tea.Quit
		case "/":
			m.searching = true
			m.searchQuery = ""
			m.statusMsg = ""
			return m, nil
		case "n":
			if len(m.searchHits) > 0 {
				m.searchIdx++
				m.jumpToHit()
				m.statusMsg = fmt.Sprintf("Match %d/%d", m.searchIdx+1, len(m.searchHits))
			}
			return m, nil
		case "N":
			if len(m.searchHits) > 0 {
				m.searchIdx--
				m.jumpToHit()
				m.statusMsg = fmt.Sprintf("Match %d/%d", m.searchIdx+1, len(m.searchHits))
			}
			return m, nil
		case "ctrl+s":
			result := m.saveLogsToFile()
			m.statusMsg = result
			return m, nil
		case "tab":
			m.jsonMode = !m.jsonMode
			m.rebuildLines()
			m.refreshViewport()
			if m.jsonMode {
				m.statusMsg = "Mode: JSON (pretty-print)"
			} else {
				m.statusMsg = "Mode: TXT (raw)"
			}
			return m, nil
		case "left":
			if m.xOffset > 0 {
				m.xOffset -= 10
				if m.xOffset < 0 {
					m.xOffset = 0
				}
				m.refreshViewport()
			}
			return m, nil
		case "right":
			m.xOffset += 10
			m.refreshViewport()
			return m, nil
		case "home":
			m.xOffset = 0
			m.refreshViewport()
			return m, nil
		case "+", "=":
			if m.maxLines < 5000 {
				m.maxLines += 200
				if m.maxLines > 5000 {
					m.maxLines = 5000
				}
				m.cfg.MaxLines = m.maxLines
				config.Save(m.cfg)
				m.statusMsg = fmt.Sprintf("Linhas iniciais: %d (próxima conexão)", m.maxLines)
			}
			return m, nil
		case "-":
			if m.maxLines > 200 {
				m.maxLines -= 200
				if m.maxLines < 200 {
					m.maxLines = 200
				}
				m.cfg.MaxLines = m.maxLines
				config.Save(m.cfg)
				m.statusMsg = fmt.Sprintf("Linhas iniciais: %d (próxima conexão)", m.maxLines)
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m LogModel) View() string {
	if !m.ready {
		return "\n  Loading..."
	}

	header := logTitleStyle.Render(m.title)

	var footer string
	if m.searching {
		footer = searchInputStyle.Render(fmt.Sprintf("  Search: %s▋", m.searchQuery))
	} else if m.statusMsg != "" {
		footer = savedMsgStyle.Render(fmt.Sprintf("  %s", m.statusMsg))
	} else if m.err != "" {
		footer = errorLineStyle.Render(fmt.Sprintf("  ⚠ %s", m.err))
	} else {
		mode := "TXT"
		if m.jsonMode {
			mode = "JSON"
		}
		footer = footerStyle.Render(fmt.Sprintf("↑/↓ • ←/→ • Tab:[%s] • +/-:ini[%d] • total:%d • /: search • Ctrl+S: save • Esc: back", mode, m.maxLines, len(m.rawLines)))
	}

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}
