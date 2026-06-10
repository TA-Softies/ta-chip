package ui

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"ta-chip/internal/checks"
	"ta-chip/internal/config"
	"ta-chip/internal/submit"
	"ta-chip/version"
)

//go:embed set_lockscreen.ps1
var lockscreenScript string

type screen int

const (
	screenBanner screen = iota
	screenAutoChecks
	screenRounder
	screenHardware
	screenKeyboardTest
	screenSoftwareReview
	screenDomainTest
	screenRemarks
	screenReview
	screenDone
)

const (
	wallpaperRowIdx  = 2
	defenderRowIdx   = 7
	activationRowIdx = 8
	audioRowIdx      = 9
	cameraRowIdx     = 10
)

type autoCheckResults struct {
	hostname      string
	timeStatus    string
	timeDetail    string
	wallpaper     string
	wallDetail    string
	office        string
	officeDetail  string
	teams         string
	teamsDetail   string
	internet      string
	internetDetail string
	df            checks.DeepFreezeResult
	domainMem     string
	domainDetail  string
	domainLogin   bool
	domainLoginDetail string
	diskFree      float64
	diskTotal     float64
	lastReboot    string
	winVersion    string
	ram           string
	defender      string
	defenderDetail string
	activation    string
	activationDetail string
	hw            checks.HardwareInfo
}

type softwareRow struct {
	label  string
	status string
	detail string
}

type checksCompleteMsg struct{ results autoCheckResults }
type submitDoneMsg struct {
	err error
	row int
}
type wallpaperFixMsg struct {
	status string
	detail string
}

// Model is the root Bubble Tea model.
type Model struct {
	cfg     *config.Config
	screen  screen
	spinner spinner.Model

	autoResults  autoCheckResults
	checksReady  bool

	rounderInput textinput.Model

	hwIndex    int
	hwSelected int
	hwResults  []HardwareResult

	keyTest       KeyTestModel
	keyTestStatus string

	swRows     []softwareRow
	swFocusRow int

	domainOverride string

	remarks textarea.Model

	submitting      bool
	fixingWallpaper bool
	audioBeepedOnce bool
	submitErr      error
	submitRow      int

	startTime time.Time

	width  int
	height int
}

func New(cfg *config.Config) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	ri := textinput.New()
	ri.Placeholder = "Your name"
	ri.Focus()
	ri.CharLimit = 64

	rm := textarea.New()
	rm.Placeholder = "Any remarks? (optional — press Tab to skip)"
	rm.SetWidth(60)
	rm.SetHeight(4)
	rm.CharLimit = 512

	return &Model{
		cfg:           cfg,
		screen:        screenBanner,
		spinner:       sp,
		rounderInput:  ri,
		hwResults:     make([]HardwareResult, len(hardwareItems)),
		keyTest:       newKeyTestModel(),
		keyTestStatus: "V",
		remarks:       rm,
		startTime:     time.Now(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.EnterAltScreen)
}

func runAutoChecks(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		r := autoCheckResults{}

		// Fast synchronous checks
		r.hostname = checks.GetHostname()
		r.diskFree, r.diskTotal = checks.GetDiskSpace()
		r.lastReboot = checks.GetLastReboot()
		r.winVersion = checks.GetWindowsVersion()
		r.ram = checks.GetRAM()
		r.domainMem, r.domainDetail = checks.CheckDomainMembership(cfg.DomainName)

		// Slow checks in parallel
		var mu sync.Mutex
		var wg sync.WaitGroup

		set := func(f func()) {
			wg.Add(1)
			go func() { defer wg.Done(); f() }()
		}

		set(func() {
			s, d := checks.CheckTimeDate(cfg.NTPToleranceSecs)
			mu.Lock(); r.timeStatus, r.timeDetail = s, d; mu.Unlock()
		})
		set(func() {
			s, d := checks.CheckWallpaper(cfg.ExpectedWallpaper)
			mu.Lock(); r.wallpaper, r.wallDetail = s, d; mu.Unlock()
		})
		set(func() {
			s, d := checks.CheckOffice()
			mu.Lock(); r.office, r.officeDetail = s, d; mu.Unlock()
		})
		set(func() {
			s, d := checks.CheckTeams()
			mu.Lock(); r.teams, r.teamsDetail = s, d; mu.Unlock()
		})
		set(func() {
			s, d := checks.CheckInternet()
			mu.Lock(); r.internet, r.internetDetail = s, d; mu.Unlock()
		})
		set(func() {
			df := checks.CheckDeepFreeze()
			mu.Lock(); r.df = df; mu.Unlock()
		})
		set(func() {
			ok, d := checks.TestDomainLogin(cfg.DomainName, cfg.DomainTestUser, cfg.DomainTestPassword)
			mu.Lock(); r.domainLogin, r.domainLoginDetail = ok, d; mu.Unlock()
		})
		set(func() {
			s, d := checks.CheckDefender()
			mu.Lock(); r.defender, r.defenderDetail = s, d; mu.Unlock()
		})
		set(func() {
			s, d := checks.CheckActivation()
			mu.Lock(); r.activation, r.activationDetail = s, d; mu.Unlock()
		})
		set(func() {
			hw := checks.GetHardwareInfo()
			mu.Lock(); r.hw = hw; mu.Unlock()
		})

		wg.Wait()
		return checksCompleteMsg{results: r}
	}
}

func runWallpaperFix(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		// Write the embedded PS1 to a temp file
		tmp, err := os.CreateTemp("", "set_lockscreen_*.ps1")
		if err != nil {
			return wallpaperFixMsg{"X", "cannot write fix script"}
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.WriteString(lockscreenScript); err != nil {
			tmp.Close()
			return wallpaperFixMsg{"X", "cannot write fix script"}
		}
		tmp.Close()

		// Pass exe dir as -ScriptDir for local image fallback
		exeDir := ""
		if exe, err := os.Executable(); err == nil {
			exeDir = filepath.Dir(exe)
		}

		cmd := exec.Command("powershell.exe",
			"-ExecutionPolicy", "Bypass",
			"-File", tmp.Name(),
			"-ScriptDir", exeDir,
		)
		if err := cmd.Run(); err != nil {
			return wallpaperFixMsg{"X", "lockscreen update failed"}
		}

		status, detail := checks.CheckWallpaper(cfg.ExpectedWallpaper)
		return wallpaperFixMsg{status, "fixed: " + detail}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.MouseMsg:
		if m.screen == screenKeyboardTest {
			switch msg.Button {
			case tea.MouseButtonLeft:
				m.keyTest.handleMousePress("LMB")
			case tea.MouseButtonMiddle:
				m.keyTest.handleMousePress("MMB")
			case tea.MouseButtonRight:
				m.keyTest.handleMousePress("RMB")
			}
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case checksCompleteMsg:
		m.autoResults = msg.results
		m.checksReady = true
		m.buildSoftwareRows()
		return m, nil

	case wallpaperFixMsg:
		m.fixingWallpaper = false
		if len(m.swRows) > wallpaperRowIdx {
			m.swRows[wallpaperRowIdx].status = msg.status
			m.swRows[wallpaperRowIdx].detail = msg.detail
		}
		return m, nil

	case submitDoneMsg:
		m.submitting = false
		m.submitErr = msg.err
		m.submitRow = msg.row
		m.screen = screenDone
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenRounder:
		m.rounderInput, cmd = m.rounderInput.Update(msg)
	case screenRemarks:
		m.remarks, cmd = m.remarks.Update(msg)
	}
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {

	case screenBanner:
		if msg.Type == tea.KeyEnter || msg.Type == tea.KeySpace {
			m.screen = screenAutoChecks
			return m, runAutoChecks(m.cfg)
		}

	case screenAutoChecks:
		if m.checksReady && msg.Type == tea.KeyEnter {
			m.screen = screenRounder
		}

	case screenRounder:
		if msg.Type == tea.KeyEnter {
			if strings.TrimSpace(m.rounderInput.Value()) != "" {
				m.screen = screenHardware
				m.hwIndex = 0
				m.hwSelected = 0
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.rounderInput, cmd = m.rounderInput.Update(msg)
		return m, cmd

	case screenHardware:
		switch msg.Type {
		case tea.KeyUp:
			if m.hwSelected > 0 {
				m.hwSelected--
			}
		case tea.KeyDown:
			if m.hwSelected < 2 {
				m.hwSelected++
			}
		case tea.KeyEnter:
			statuses := []string{"V", "Y", "X"}
			m.hwResults[m.hwIndex] = HardwareResult{
				Item:   hardwareItems[m.hwIndex],
				Status: statuses[m.hwSelected],
			}
			m.hwIndex++
			m.hwSelected = 0
			if m.hwIndex >= len(hardwareItems) {
				m.screen = screenKeyboardTest
			}
		case tea.KeyRunes:
			switch strings.ToUpper(string(msg.Runes)) {
			case "V":
				m.hwSelected = 0
			case "Y":
				m.hwSelected = 1
			case "X":
				m.hwSelected = 2
			}
		}

	case screenKeyboardTest:
		if msg.Type == tea.KeyEnter {
			keyCount := m.keyTest.pressedCount()
			mouseCount := m.keyTest.mouseCount()
			if keyCount > 10 && mouseCount >= 2 {
				m.keyTestStatus = "V"
			} else if keyCount > 0 || mouseCount > 0 {
				m.keyTestStatus = "Y"
			} else {
				m.keyTestStatus = "X"
			}
			m.screen = screenSoftwareReview
			if m.autoResults.hw.Audio != "" && !m.audioBeepedOnce {
				m.audioBeepedOnce = true
				go checks.PlayBeep()
			}
			return m, nil
		}
		m.keyTest.handleKeyPress(msg)

	case screenSoftwareReview:
		switch msg.Type {
		case tea.KeyUp:
			if m.swFocusRow > 0 {
				m.swFocusRow--
			}
		case tea.KeyDown:
			if m.swFocusRow < len(m.swRows)-1 {
				m.swFocusRow++
			}
		case tea.KeyEnter, tea.KeyTab:
			m.screen = screenDomainTest
		case tea.KeyRunes:
			switch strings.ToUpper(string(msg.Runes)) {
			case "V":
				m.swRows[m.swFocusRow].status = "V"
			case "Y":
				m.swRows[m.swFocusRow].status = "Y"
			case "X":
				m.swRows[m.swFocusRow].status = "X"
			case "F":
				if m.swFocusRow == wallpaperRowIdx && !m.fixingWallpaper {
					m.fixingWallpaper = true
					return m, runWallpaperFix(m.cfg)
				}
			case "B":
				if m.swFocusRow == audioRowIdx {
					go checks.PlayBeep()
				}
			case "L":
				if m.swFocusRow == cameraRowIdx {
					go exec.Command("explorer.exe", "ms-camera:").Start()
				}
			}
		}

	case screenDomainTest:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyTab:
			m.screen = screenRemarks
		case tea.KeyRunes:
			switch strings.ToUpper(string(msg.Runes)) {
			case "V":
				m.domainOverride = "V"
			case "Y":
				m.domainOverride = "Y"
			case "X":
				m.domainOverride = "X"
			}
		}

	case screenRemarks:
		if msg.Type == tea.KeyTab || (msg.Type == tea.KeyEnter && msg.Alt) {
			m.screen = screenReview
			return m, nil
		}
		var cmd tea.Cmd
		m.remarks, cmd = m.remarks.Update(msg)
		return m, cmd

	case screenReview:
		switch msg.Type {
		case tea.KeyEnter:
			m.submitting = true
			return m, m.doSubmit()
		case tea.KeyEsc:
			m.screen = screenRemarks
		}

	case screenDone:
		if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
			return m, tea.Quit
		}
	}

	return m, nil
}

func deviceRowStatus(s string) string {
	if s == "" {
		return "X"
	}
	return "Y"
}

func (m *Model) buildSoftwareRows() {
	r := m.autoResults
	m.swRows = []softwareRow{
		{"Boot to Windows", "V", "Running"},
		{"Time & Date", r.timeStatus, r.timeDetail},
		{"Lockscreen Wallpaper", r.wallpaper, r.wallDetail},
		{"Microsoft Office", r.office, r.officeDetail},
		{"Microsoft Teams", r.teams, r.teamsDetail},
		{"Internet", r.internet, r.internetDetail},
		{"DeepFreeze Frozen", r.df.Frozen, r.df.Detail},
		{"Windows Defender", r.defender, r.defenderDetail},
		{"Windows Activation", r.activation, r.activationDetail},
		{"Audio", deviceRowStatus(r.hw.Audio), r.hw.Audio},
		{"Camera", deviceRowStatus(r.hw.Camera), r.hw.Camera},
	}
}

func (m *Model) domainStatus() string {
	if m.domainOverride != "" {
		return m.domainOverride
	}
	if m.autoResults.domainMem == "V" && m.autoResults.domainLogin {
		return "V"
	}
	if m.autoResults.domainMem == "V" {
		return "Y"
	}
	return "X"
}

func (m *Model) doSubmit() tea.Cmd {
	data := submit.InspectionData{
		PCLocation:     m.autoResults.hostname,
		Rounder:        m.rounderInput.Value(),
		ShiftTime:      m.startTime.Format("15:04"),
		Display:        m.hwResults[0].Status,
		MouseKeyboard:  m.keyTestStatus,
		KensingtonLock: m.hwResults[1].Status,
		Conduiting:     m.hwResults[2].Status,
		Tidiness:       m.hwResults[3].Status,
		BootToWindows:  m.swRows[0].status,
		TimeDate:       m.swRows[1].status,
		Wallpaper:      m.swRows[2].status,
		Domain:         m.domainStatus(),
		MSOffice:       m.swRows[3].status,
		MSTeams:        m.swRows[4].status,
		Internet:       m.swRows[5].status,
		DFFrozen:       m.swRows[6].status,
		DFPolicy:       m.autoResults.df.PolicyName,
		DiskSpace:      fmt.Sprintf("%.1f GB free of %.1f GB", m.autoResults.diskFree, m.autoResults.diskTotal),
		LastReboot:     m.autoResults.lastReboot,
		WinVersion:     m.autoResults.winVersion,
		RAM:            m.autoResults.ram,
		Monitor:        m.autoResults.hw.Monitor,
		Keyboard:       m.autoResults.hw.Keyboard,
		Mouse:          m.autoResults.hw.Mouse,
		Defender:       m.swRows[defenderRowIdx].status,
		Activation:     m.swRows[activationRowIdx].status,
		Audio:          m.swRows[audioRowIdx].status,
		Camera:         m.swRows[cameraRowIdx].status,
		Remarks:        m.remarks.Value(),
	}
	cfg := m.cfg
	return func() tea.Msg {
		resp, err := submit.Submit(cfg.AppScriptURL, data)
		row := 0
		if resp != nil {
			row = resp.Row
		}
		return submitDoneMsg{err: err, row: row}
	}
}

// renderPage wraps content with a consistent header and footer.
func (m *Model) renderPage(stepLabel, content, hint string) string {
	w := m.width
	if w < 80 {
		w = 120
	}

	// Header
	title := "  TA CHIP  ·  PC Health Inspector"
	step := stepLabel + "  "
	gap := w - len(title) - len(step) - 4
	if gap < 1 {
		gap = 1
	}
	headerText := title + strings.Repeat(" ", gap) + step
	header := styleHeaderBar.Copy().Width(w).Render(headerText)

	// Body — pad top so content appears vertically centered-ish
	body := "\n" + content

	// Footer
	footer := ""
	if hint != "" {
		footer = "\n" + styleFooterBar.Render("  "+hint)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body+footer)
}

func (m *Model) View() string {
	switch m.screen {
	case screenBanner:
		return m.viewBanner()
	case screenAutoChecks:
		return m.viewAutoChecks()
	case screenRounder:
		return m.viewRounder()
	case screenHardware:
		return m.viewHardware()
	case screenKeyboardTest:
		return m.viewKeyboardTest()
	case screenSoftwareReview:
		return m.viewSoftwareReview()
	case screenDomainTest:
		return m.viewDomainTest()
	case screenRemarks:
		return m.viewRemarks()
	case screenReview:
		return m.viewReview()
	case screenDone:
		return m.viewDone()
	}
	return ""
}

func (m *Model) viewBanner() string {
	content := fmt.Sprintf("%s\n\n  %s  %s",
		styleBanner.Render(banner),
		styleDim.Render("Version:"),
		styleLabel.Render(version.Version),
	)
	return m.renderPage("", content, "Press Enter or Space to start")
}

func (m *Model) viewAutoChecks() string {
	if !m.checksReady {
		content := fmt.Sprintf("\n\n  %s  %s\n",
			m.spinner.View(),
			styleLabel.Render("Running automated checks..."),
		)
		return m.renderPage("Auto Checks", content, "")
	}
	r := m.autoResults
	lines := []string{
		"",
		fmt.Sprintf("  %-24s %s  %s", "PC Location", statusStyle("V"), styleLabel.Render(r.hostname)),
		fmt.Sprintf("  %-24s %s  %s", "Time & Date", statusStyle(r.timeStatus), styleDim.Render(r.timeDetail)),
		fmt.Sprintf("  %-24s %s  %s", "Lockscreen Wallpaper", statusStyle(r.wallpaper), styleDim.Render(r.wallDetail)),
		fmt.Sprintf("  %-24s %s  %s", "Microsoft Office", statusStyle(r.office), styleDim.Render(r.officeDetail)),
		fmt.Sprintf("  %-24s %s  %s", "Microsoft Teams", statusStyle(r.teams), styleDim.Render(r.teamsDetail)),
		fmt.Sprintf("  %-24s %s  %s", "Internet", statusStyle(r.internet), styleDim.Render(r.internetDetail)),
		fmt.Sprintf("  %-24s %s  %s", "DeepFreeze", statusStyle(r.df.Frozen), styleDim.Render(r.df.Detail)),
		fmt.Sprintf("  %-24s %s  %s", "Domain ("+m.cfg.DomainName+")", statusStyle(r.domainMem), styleDim.Render(r.domainDetail)),
		"",
		styleDim.Render(fmt.Sprintf("  %-14s %s", "Disk Space", fmt.Sprintf("%.1f GB free of %.1f GB", r.diskFree, r.diskTotal))),
		styleDim.Render(fmt.Sprintf("  %-14s %s", "Last Reboot", r.lastReboot)),
		styleDim.Render(fmt.Sprintf("  %-14s %s", "Windows", r.winVersion)),
		styleDim.Render(fmt.Sprintf("  %-14s %s", "RAM", r.ram)),
		styleDim.Render(fmt.Sprintf("  %-14s %s", "Monitor", r.hw.Monitor)),
		styleDim.Render(fmt.Sprintf("  %-14s %s", "Keyboard", r.hw.Keyboard)),
		styleDim.Render(fmt.Sprintf("  %-14s %s", "Mouse", r.hw.Mouse)),
	}
	return m.renderPage("Auto Checks", strings.Join(lines, "\n"), "Press Enter to continue")
}

func (m *Model) viewRounder() string {
	content := fmt.Sprintf("\n\n  %s\n\n  %s",
		styleHeader.Render("Who is doing rounds today?"),
		m.rounderInput.View(),
	)
	return m.renderPage("Rounder", content, "Enter to continue")
}

func (m *Model) viewHardware() string {
	item := hardwareItems[m.hwIndex]
	step := fmt.Sprintf("Hardware %d / %d", m.hwIndex+1, len(hardwareItems))
	content := "\n" + renderHardwareScreen(item, m.hwSelected)
	return m.renderPage(step, content, "↑↓ / V Y X to select  •  Enter to confirm")
}

func (m *Model) viewKeyboardTest() string {
	content := "\n" + renderKeyboardTestScreen(m.keyTest)
	return m.renderPage("Keyboard & Mouse Test", content, "")
}

func (m *Model) viewSoftwareReview() string {
	var sb strings.Builder
	sb.WriteString("\n")

	for i, row := range m.swRows {
		cursor := "  "
		label := row.label
		detail := row.detail

		// Row-specific action hints
		fixHint := ""
		switch i {
		case wallpaperRowIdx:
			if m.fixingWallpaper {
				fixHint = "  " + m.spinner.View() + styleDim.Render(" fixing...")
			} else if row.status == "X" {
				fixHint = "  " + styleDim.Render("[F to fix]")
			}
		case audioRowIdx:
			if i == m.swFocusRow {
				fixHint = "  " + styleDim.Render("[B to beep]")
			}
		case cameraRowIdx:
			if i == m.swFocusRow {
				fixHint = "  " + styleDim.Render("[L to open camera]")
			}
		}

		if i == m.swFocusRow {
			cursor = styleSelected.Render("▶")
			label = styleLabel.Render(label)
			detail = styleDim.Render(detail)
		} else {
			detail = styleDim.Render(detail)
		}
		sb.WriteString(fmt.Sprintf("%s %-26s %s  %s%s\n",
			cursor,
			label,
			statusStyle(row.status),
			detail,
			fixHint,
		))
	}
	return m.renderPage("Software Review", sb.String(), "↑↓ navigate  •  V / Y / X override  •  Enter to continue")
}

func (m *Model) viewDomainTest() string {
	r := m.autoResults
	loginResult := "X  Failed"
	if r.domainLogin {
		loginResult = "V  Success"
	}

	override := ""
	if m.domainOverride != "" {
		override = "  Override: " + statusStyle(m.domainOverride)
	}

	content := fmt.Sprintf("\n\n"+
		"  %-26s %s  %s\n"+
		"  %-26s %s\n\n"+
		"  %s",
		"Domain Membership", statusStyle(r.domainMem), styleDim.Render(r.domainDetail),
		"Login Test ("+m.cfg.DomainTestUser+")", styleDim.Render(loginResult),
		styleDim.Render("V / Y / X to override"+override),
	)
	return m.renderPage("Domain Check", content, "Enter to continue")
}

func (m *Model) viewRemarks() string {
	content := fmt.Sprintf("\n\n  %s\n\n  %s",
		styleHeader.Render("Remarks"),
		m.remarks.View(),
	)
	return m.renderPage("Remarks", content, "Alt+Enter or Tab to continue  •  Tab to skip")
}

func (m *Model) viewReview() string {
	r := m.autoResults

	sw := func(i int) string { return m.swRows[i].status }
	hw := func(i int) string {
		if i < len(m.hwResults) {
			return m.hwResults[i].Status
		}
		return "-"
	}

	if m.submitting {
		content := fmt.Sprintf("\n\n  %s  %s", m.spinner.View(), styleLabel.Render("Submitting to Google Sheets..."))
		return m.renderPage("Review", content, "")
	}

	lines := []string{
		"",
		fmt.Sprintf("  %-28s %s", "PC Location", styleLabel.Render(r.hostname)),
		fmt.Sprintf("  %-28s %s", "Rounder", styleLabel.Render(m.rounderInput.Value())),
		fmt.Sprintf("  %-28s %s", "Started", styleDim.Render(m.startTime.Format("15:04"))),
		"",
		fmt.Sprintf("  %-28s %s", "Display", statusStyle(hw(0))),
		fmt.Sprintf("  %-28s %s  %s", "Mouse & Keyboard", statusStyle(m.keyTestStatus),
			styleDim.Render(fmt.Sprintf("(%d keys, %d/3 mouse)", m.keyTest.pressedCount(), m.keyTest.mouseCount()))),
		fmt.Sprintf("  %-28s %s", "Kensington Lock", statusStyle(hw(1))),
		fmt.Sprintf("  %-28s %s", "Conduiting", statusStyle(hw(2))),
		fmt.Sprintf("  %-28s %s", "Tidiness", statusStyle(hw(3))),
		"",
		fmt.Sprintf("  %-28s %s", "Boot to Windows", statusStyle(sw(0))),
		fmt.Sprintf("  %-28s %s  %s", "Time & Date", statusStyle(sw(1)), styleDim.Render(r.timeDetail)),
		fmt.Sprintf("  %-28s %s", "Lockscreen Wallpaper", statusStyle(sw(2))),
		fmt.Sprintf("  %-28s %s", "Domain ("+m.cfg.DomainName+")", statusStyle(m.domainStatus())),
		fmt.Sprintf("  %-28s %s", "Microsoft Office", statusStyle(sw(3))),
		fmt.Sprintf("  %-28s %s", "Microsoft Teams", statusStyle(sw(4))),
		fmt.Sprintf("  %-28s %s", "Internet", statusStyle(sw(5))),
		fmt.Sprintf("  %-28s %s", "DeepFreeze Frozen", statusStyle(sw(6))),
		fmt.Sprintf("  %-28s %s", "DeepFreeze Policy", styleDim.Render(r.df.PolicyName)),
		fmt.Sprintf("  %-28s %s", "Windows Defender", statusStyle(sw(defenderRowIdx))),
		fmt.Sprintf("  %-28s %s", "Windows Activation", statusStyle(sw(activationRowIdx))),
		fmt.Sprintf("  %-28s %s", "Audio", statusStyle(sw(audioRowIdx))),
		fmt.Sprintf("  %-28s %s", "Camera", statusStyle(sw(cameraRowIdx))),
		"",
		styleDim.Render(fmt.Sprintf("  %-28s %s", "Disk Space", fmt.Sprintf("%.1f GB free of %.1f GB", r.diskFree, r.diskTotal))),
		styleDim.Render(fmt.Sprintf("  %-28s %s", "Windows", r.winVersion)),
		styleDim.Render(fmt.Sprintf("  %-28s %s", "RAM", r.ram)),
		"",
		fmt.Sprintf("  %-28s %s", "Remarks", styleDim.Render(m.remarks.Value())),
	}
	return m.renderPage("Review", strings.Join(lines, "\n"), "Enter to submit  •  Esc to go back")
}

func (m *Model) viewDone() string {
	if m.submitErr != nil {
		content := fmt.Sprintf("\n\n  %s\n\n  %s",
			styleError.Render("Submission failed"),
			styleDim.Render(m.submitErr.Error()),
		)
		return m.renderPage("Done", content, "Press Enter to exit")
	}
	content := fmt.Sprintf("\n\n  %s\n\n  %s",
		styleSuccess.Render("Done! Results submitted to Google Sheets."),
		styleDim.Render(fmt.Sprintf("Row %d written.", m.submitRow)),
	)
	return m.renderPage("Done", content, "Press Enter to exit")
}
