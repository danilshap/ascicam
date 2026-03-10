package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type runTarget int

const (
	runNone runTarget = iota
	runPreview
	runPhotoTXT
	runRecordGIF
)

type settingsField int

const (
	fieldDevice settingsField = iota
	fieldWidth
	fieldFPS
	fieldMode
	fieldPalette
	fieldContrast
	fieldBrightness
	fieldMirror
	fieldInvert
	fieldStatus
	fieldCaptureFullscreen
)

type tuiAction struct {
	title       string
	description string
	target      runTarget
}

type tuiModel struct {
	cfg          Config
	width        int
	height       int
	focusPane    int
	actionIndex  int
	settingIndex int
	editing      bool
	editField    settingsField
	editTarget   runTarget
	input        textinput.Model
	status       string
	runTarget    runTarget
	cancelled    bool
}

var tuiActions = []tuiAction{
	{title: "Live Preview", description: "Run the camera feed in the terminal", target: runPreview},
	{title: "Save ASCII Photo", description: "Capture one frame into a .txt file", target: runPhotoTXT},
	{title: "Record GIF Session", description: "Build an animated .gif up to 5 seconds", target: runRecordGIF},
}

func RunTUI(cfg Config) error {
	notice := ""
	for {
		model := newTUIModel(cfg, notice)
		program := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := program.Run()
		if err != nil {
			return err
		}

		m := finalModel.(tuiModel)
		if m.cancelled || m.runTarget == runNone {
			return nil
		}

		runCfg, err := m.buildRunConfig()
		if err != nil {
			return err
		}
		outcome, err := Run(runCfg)
		if err != nil {
			return err
		}

		cfg = runCfg
		cfg.UseTUI = true
		cfg.Output.PhotoPath = ""
		cfg.Output.RecordPath = ""
		notice = outcome.Notice
	}
}

func newTUIModel(cfg Config, notice string) tuiModel {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 256
	input.Width = 36

	status := "Tab switches panes. Enter activates. Space toggles booleans."
	if notice != "" {
		status = notice
	}

	return tuiModel{
		cfg:       cfg,
		focusPane: 0,
		input:     input,
		status:    status,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.updateEditor(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		case "tab", "shift+tab", "left", "right", "h", "l":
			m.focusPane = 1 - m.focusPane
			return m, nil
		case "up", "k":
			if m.focusPane == 0 {
				m.actionIndex = maxInt(0, m.actionIndex-1)
			} else {
				m.settingIndex = maxInt(0, m.settingIndex-1)
			}
			return m, nil
		case "down", "j":
			if m.focusPane == 0 {
				m.actionIndex = minInt(len(tuiActions)-1, m.actionIndex+1)
			} else {
				m.settingIndex = minInt(int(fieldCaptureFullscreen), m.settingIndex+1)
			}
			return m, nil
		case " ":
			if m.focusPane == 1 {
				return m.toggleSetting()
			}
			return m, nil
		case "enter":
			if m.focusPane == 0 {
				return m.activateAction()
			}
			return m.editSetting()
		}
	}

	return m, nil
}

func (m tuiModel) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.editing = false
			m.status = "Edit cancelled."
			return m, nil
		case "enter":
			return m.commitEditor()
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m tuiModel) activateAction() (tea.Model, tea.Cmd) {
	action := tuiActions[m.actionIndex]
	switch action.target {
	case runPreview:
		m.runTarget = runPreview
		return m, tea.Quit
	case runPhotoTXT, runRecordGIF:
		m.editing = true
		m.editTarget = action.target
		m.input.SetValue(defaultOutputPath(action.target))
		m.input.CursorEnd()
		m.status = action.description
		return m, textinput.Blink
	default:
		return m, nil
	}
}

func (m tuiModel) editSetting() (tea.Model, tea.Cmd) {
	field := settingsField(m.settingIndex)
	switch field {
	case fieldMode:
		if m.cfg.Render.Mode == ModeGray {
			m.cfg.Render.Mode = ModeEdges
		} else {
			m.cfg.Render.Mode = ModeGray
		}
		m.status = "Render mode updated."
		return m, nil
	case fieldMirror:
		m.cfg.Mirror = !m.cfg.Mirror
		m.status = "Mirror toggled."
		return m, nil
	case fieldInvert:
		m.cfg.Render.Invert = !m.cfg.Render.Invert
		m.status = "Palette inversion toggled."
		return m, nil
	case fieldStatus:
		m.cfg.Render.ShowStatus = !m.cfg.Render.ShowStatus
		m.status = "Status bar visibility updated."
		return m, nil
	case fieldCaptureFullscreen:
		m.cfg.Output.CaptureFullscreen = !m.cfg.Output.CaptureFullscreen
		m.status = "Capture fullscreen updated."
		return m, nil
	default:
		m.editing = true
		m.editField = field
		m.input.SetValue(m.settingValue(field))
		m.input.CursorEnd()
		m.status = "Editing setting."
		return m, textinput.Blink
	}
}

func (m tuiModel) toggleSetting() (tea.Model, tea.Cmd) {
	return m.editSetting()
}

func (m tuiModel) commitEditor() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())

	if m.editTarget != runNone {
		if value == "" {
			m.status = "Path cannot be empty."
			return m, nil
		}
		switch m.editTarget {
		case runPhotoTXT:
			m.cfg.Output = OutputOptions{PhotoPath: ensureDefaultExt(value, ".txt")}
		case runRecordGIF:
			m.cfg.Output = OutputOptions{RecordPath: ensureDefaultExt(value, ".gif")}
		}
		m.runTarget = m.editTarget
		m.editing = false
		m.editTarget = runNone
		return m, tea.Quit
	}

	if err := m.applySettingValue(m.editField, value); err != nil {
		m.status = err.Error()
		return m, nil
	}
	if err := m.cfg.NormalizeAndValidate(); err != nil {
		m.status = err.Error()
		return m, nil
	}

	m.editing = false
	m.status = "Setting updated."
	return m, nil
}

func (m *tuiModel) applySettingValue(field settingsField, value string) error {
	switch field {
	case fieldDevice:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("device must be an integer")
		}
		m.cfg.DeviceID = parsed
	case fieldWidth:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("width must be an integer")
		}
		m.cfg.RequestedWidth = parsed
	case fieldFPS:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("fps must be an integer")
		}
		m.cfg.FPS = parsed
	case fieldPalette:
		m.cfg.Palette = value
	case fieldContrast:
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("contrast must be a number")
		}
		m.cfg.Render.Contrast = parsed
	case fieldBrightness:
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("brightness must be a number")
		}
		m.cfg.Render.Brightness = parsed
	}
	return nil
}

func (m tuiModel) buildRunConfig() (Config, error) {
	cfg := m.cfg
	cfg.UseTUI = false
	return cfg, cfg.NormalizeAndValidate()
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return "Loading terminal UI..."
	}

	header := m.headerView()
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.actionsView(), m.settingsView())
	footer := m.footerView()

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	return appDocStyle.Render(content)
}

func (m tuiModel) headerView() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("ascicam")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("109")).Render("terminal camera studio")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("tab switch pane  •  enter edit/run  •  q quit")
	return lipgloss.JoinVertical(lipgloss.Left, title+"  "+subtitle, help)
}

func (m tuiModel) actionsView() string {
	var lines []string
	for i, action := range tuiActions {
		style := itemStyle
		if m.focusPane == 0 && i == m.actionIndex {
			style = selectedItemStyle
		}
		lines = append(lines, style.Render(action.title))
		lines = append(lines, descStyle.Render(action.description))
	}
	return panelStyle.Width(m.panelWidth(0)).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Actions"),
			strings.Join(lines, "\n"),
		),
	)
}

func (m tuiModel) settingsView() string {
	fields := []settingsField{
		fieldDevice, fieldWidth, fieldFPS, fieldMode, fieldPalette,
		fieldContrast, fieldBrightness, fieldMirror, fieldInvert,
		fieldStatus, fieldCaptureFullscreen,
	}

	var lines []string
	for i, field := range fields {
		label := settingLabel(field)
		value := m.settingValue(field)
		line := lipgloss.JoinHorizontal(lipgloss.Top,
			settingKeyStyle.Width(16).Render(label),
			settingValueStyle.Render(value),
		)
		if m.focusPane == 1 && i == m.settingIndex {
			line = selectedSettingStyle.Width(m.panelWidth(1) - 4).Render(line)
		}
		lines = append(lines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		panelTitleStyle.Render("Settings"),
		strings.Join(lines, "\n"),
	)
	if m.editing {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			"",
			panelTitleStyle.Render("Editor"),
			editorHintStyle.Render("Enter saves • Esc cancels"),
			editorBoxStyle.Render(m.input.View()),
		)
	}

	return panelStyle.Width(m.panelWidth(1)).Render(content)
}

func (m tuiModel) footerView() string {
	left := footerStyle.Render(m.status)
	right := footerAccentStyle.Render("preview, photo, gif, record")
	width := maxInt(0, m.width-6)
	return lipgloss.NewStyle().Width(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			left,
			lipgloss.NewStyle().Width(maxInt(0, width-lipgloss.Width(left)-lipgloss.Width(right))).Render(""),
			right,
		),
	)
}

func (m tuiModel) panelWidth(index int) int {
	total := maxInt(60, m.width-6)
	if index == 0 {
		return total/2 - 1
	}
	return total - (total/2 - 1)
}

func (m tuiModel) settingValue(field settingsField) string {
	switch field {
	case fieldDevice:
		return strconv.Itoa(m.cfg.DeviceID)
	case fieldWidth:
		return strconv.Itoa(m.cfg.RequestedWidth)
	case fieldFPS:
		return strconv.Itoa(m.cfg.FPS)
	case fieldMode:
		return string(m.cfg.Render.Mode)
	case fieldPalette:
		return fmt.Sprintf("%q", m.cfg.Palette)
	case fieldContrast:
		return fmt.Sprintf("%.2f", m.cfg.Render.Contrast)
	case fieldBrightness:
		return fmt.Sprintf("%.2f", m.cfg.Render.Brightness)
	case fieldMirror:
		return fmt.Sprintf("%t", m.cfg.Mirror)
	case fieldInvert:
		return fmt.Sprintf("%t", m.cfg.Render.Invert)
	case fieldStatus:
		return fmt.Sprintf("%t", m.cfg.Render.ShowStatus)
	case fieldCaptureFullscreen:
		return fmt.Sprintf("%t", m.cfg.Output.CaptureFullscreen)
	default:
		return ""
	}
}

func settingLabel(field settingsField) string {
	switch field {
	case fieldDevice:
		return "Device"
	case fieldWidth:
		return "Width"
	case fieldFPS:
		return "FPS"
	case fieldMode:
		return "Mode"
	case fieldPalette:
		return "Palette"
	case fieldContrast:
		return "Contrast"
	case fieldBrightness:
		return "Brightness"
	case fieldMirror:
		return "Mirror"
	case fieldInvert:
		return "Invert"
	case fieldStatus:
		return "Show status"
	case fieldCaptureFullscreen:
		return "Capture fullscreen"
	default:
		return ""
	}
}

func defaultOutputPath(target runTarget) string {
	switch target {
	case runPhotoTXT:
		return "frame.txt"
	case runRecordGIF:
		return "session.gif"
	default:
		return ""
	}
}

func ensureDefaultExt(path, ext string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if len(path) >= len(ext) && strings.EqualFold(path[len(path)-len(ext):], ext) {
		return path
	}
	if strings.Contains(filepath.Base(path), ".") {
		return path
	}
	return path + ext
}

var (
	appDocStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Height(20)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("69")).
				Bold(true).
				Padding(0, 1)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			MarginBottom(1).
			PaddingLeft(1)

	settingKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("110"))

	settingValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	selectedSettingStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("15")).
				Padding(0, 1)

	editorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(0, 1)

	editorHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			MarginBottom(1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	footerAccentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("109"))
)
