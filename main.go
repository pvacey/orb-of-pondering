package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const header = `
██████╗  ██████╗ ███╗   ██╗██████╗ ███████╗██████╗ 
██╔══██╗██╔═══██╗████╗  ██║██╔══██╗██╔════╝██╔══██╗
██████╔╝██║   ██║██╔██╗ ██║██║  ██║█████╗  ██████╔╝
██╔═══╝ ██║   ██║██║╚██╗██║██║  ██║██╔══╝  ██╔══██╗
██║     ╚██████╔╝██║ ╚████║██████╔╝███████╗██║  ██║
╚═╝      ╚═════╝ ╚═╝  ╚═══╝╚═════╝ ╚══════╝╚═╝  ╚═╝
`

// --- Model and Commands ---

// A message to trigger a frame update
type tickMsg time.Time

// A message with the answer from the cosmos
type answerMsg struct{ answer string }

// A message for when things go wrong
type errMsg struct{ err error }

// JSON struct for the request payload
type questionPayload struct {
	Question string `json:"question"`
}

// JSON structs for parsing the response
type wisdomResponse struct {
	Wisdom string `json:"wisdom"`
}

// The command to produce the tickMsg at a regular interval
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// The main application model
type model struct {
	frame         int // Current animation frame, used for swirling
	width         int // Terminal width
	height        int // Terminal height
	textInput     textinput.Model
	spinner       spinner.Model
	thinking      bool
	showingAnswer bool
	answer        string
	renderer      *lipgloss.Renderer
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 200
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#222"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("155"))

	return model{
		textInput:     ti,
		spinner:       s,
		thinking:      false,
		showingAnswer: false,
		frame:         rand.Intn(1080), // Randomize starting frame for color
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.thinking {
			return m, nil // Ignore key presses when thinking
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.showingAnswer {
				m.showingAnswer = false
				m.textInput.Focus()
				return m, textinput.Blink
			} else if m.textInput.Value() != "" {
				logToFile(m.textInput.Value())
				m.thinking = true
				m.textInput.Blur()
				return m, tea.Batch(
					tea.Tick(time.Second/10, func(t time.Time) tea.Msg { return spinner.TickMsg{} }),
					getAnswerCmd(m.textInput.Value()),
				)
			}
		}

	case answerMsg:
		m.thinking = false
		m.showingAnswer = true
		m.answer = msg.answer
		m.textInput.Reset()
		return m, nil

	case errMsg:
		m.thinking = false
		m.showingAnswer = true
		m.answer = "The cosmos is silent. Your question remains unanswered."
		m.textInput.Reset()
		log.Printf("Error getting answer: %v", msg.err) // Log error
		return m, nil

	case tickMsg: // For orb animation
		m.frame++
		cmds = append(cmds, tickCmd())
	}

	if m.thinking {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	} else if !m.showingAnswer {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// --- View and Rendering Logic ---

func getAnswerCmd(question string) tea.Cmd {
	return func() tea.Msg {
		answer, err := getAnswer(question)
		if err != nil {
			return errMsg{err}
		}
		return answerMsg{answer}
	}
}

func getAnswer(question string) (string, error) {
	payload := questionPayload{Question: question}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal question: %w", err)
	}

	resp, err := http.Post("https://orb.ponder.guru/", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to get wisdom: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wisdom API returned non-200 status: %d", resp.StatusCode)
	}

	var wisdomResp wisdomResponse
	if err := json.NewDecoder(resp.Body).Decode(&wisdomResp); err != nil {
		return "", fmt.Errorf("failed to decode wisdom response: %w", err)
	}

	if wisdomResp.Wisdom != "" {
		return wisdomResp.Wisdom, nil
	}

	return "", fmt.Errorf("wisdom not found in response")
}

func logToFile(text string) {
	f, err := os.OpenFile("orb_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if _, err := f.WriteString(text + "\n"); err != nil {
		log.Fatal(err)
	}
}

// Deepest, almost black tone for the rim
var darkestBlue = lipgloss.Color("#250042")

// hslToHex converts HSL color values to a hex string.
func hslToHex(h, s, l float64) string {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	s /= 100
	l /= 100

	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	var r, g, b float64
	if h >= 0 && h < 60 {
		r, g, b = c, x, 0
	} else if h >= 60 && h < 120 {
		r, g, b = x, c, 0
	} else if h >= 120 && h < 180 {
		r, g, b = 0, c, x
	} else if h >= 180 && h < 240 {
		r, g, b = 0, x, c
	} else if h >= 240 && h < 300 {
		r, g, b = x, 0, c
	} else {
		r, g, b = c, 0, x
	}

	r = (r + m) * 255
	g = (g + m) * 255
	b = (b + m) * 255

	return fmt.Sprintf("#%02X%02X%02X", int(r), int(g), int(b))
}

// getColorSubtle returns a color from the palette based on an input value.
func getColorSubtle(val float64, palette []lipgloss.Color) lipgloss.Color {
	cycle := math.Mod(val, 1.0)
	switch {
	case cycle < 0.2:
		return palette[4]
	case cycle < 0.4:
		return palette[3]
	case cycle < 0.6:
		return palette[2]
	case cycle < 0.8:
		return palette[1]
	default:
		return palette[0]
	}
}

func applyGradient(text string, palette []lipgloss.Color, frame int, newStyle func() lipgloss.Style) string {
	var builder strings.Builder

	paletteSize := len(palette)
	textLength := utf8.RuneCountInString(text)
	scrollOffset := frame / 3

	for i, runeValue := range text {
		paletteIndex := int(float64(i) / float64(textLength) * float64(paletteSize))
		scrolledIndex := (paletteIndex + scrollOffset) % paletteSize
		color := palette[scrolledIndex]
		style := newStyle().Foreground(color)
		builder.WriteString(style.Render(string(runeValue)))
	}
	return builder.String()
}

func renderOrbPixel(x, y, orbWidth, orbHeight, radius, frame int, palette []lipgloss.Color, newStyle func() lipgloss.Style) string {
	nx := float64(x) - float64(orbWidth)/2.0
	ny := float64(y) - float64(orbHeight)/2.0

	distSq := (nx*nx)/4.0 + (ny * ny)
	dist := math.Sqrt(distSq)

	if dist < float64(radius) {
		swirlValue := (dist * 0.2) + math.Sin(nx/6.0+ny/8.0+float64(frame)/10.0) + math.Cos(ny/10.0+nx/12.0+float64(frame)/15.0)
		color := getColorSubtle(swirlValue, palette)
		if dist > float64(radius)*0.9 {
			color = darkestBlue
		}
		return newStyle().Foreground(color).SetString("█").String()
	}
	return " "
}

func (m model) View() string {
	newStyle := lipgloss.NewStyle
	if m.renderer != nil {
		newStyle = m.renderer.NewStyle
	}
	termWidth := m.width
	if termWidth == 0 {
		termWidth = 60
	} else if termWidth > 100 {
		termWidth = 98
	}

	// Orb dimensions
	orbWidth := termWidth
	radius := orbWidth / 4
	orbHeight := radius * 2
	visibleOrbHeight := int(float64(orbHeight) * 0.6)

	// Palette
	baseHue := math.Mod(float64(m.frame)/3.0, 360)
	palette := make([]lipgloss.Color, 5)
	for i := 0; i < 5; i++ {
		hue := baseHue + float64(i)*15
		sat := 65.0 + float64(i)*3
		light := 65.0 - float64(i)*2
		palette[i] = lipgloss.Color(hslToHex(hue, sat, light))
	}

	// Header setup
	gradientPalette := make([]lipgloss.Color, 10)
	for i := 0; i < 10; i++ {
		hue := baseHue + float64(i)*10
		sat := 70.0
		light := 65.0
		gradientPalette[i] = lipgloss.Color(hslToHex(hue, sat, light))
	}
	headerLines := strings.Split(header, "\n")
	var styledHeaderLines []string
	for _, line := range headerLines {
		styledHeaderLines = append(styledHeaderLines, applyGradient(line, gradientPalette, m.frame, newStyle))
	}
	headerView := lipgloss.JoinVertical(lipgloss.Left, styledHeaderLines...)
	headerView = newStyle().Width(termWidth).Align(lipgloss.Center).Render(headerView)

	// Interactive element setup
	var interactiveElement string
	if m.thinking {
		spinnerView := m.spinner.View() + " consulting the cosmos..."
		interactiveElement = newStyle().Padding(1, 2).Render(spinnerView)
	} else if m.showingAnswer {
		answerView := newStyle().Padding(1, 2).Render(m.answer)
		promptView := newStyle().Padding(0, 2).Foreground(lipgloss.Color("240")).Render("Ask another question [enter]")
		interactiveElement = lipgloss.JoinVertical(lipgloss.Center, answerView, promptView)
	} else {
		m.textInput.Width = orbWidth / 2
		prompt := newStyle().Padding(0, 1).Foreground(lipgloss.Color("#FFF")).Render("What is the knowledge you seek?")
		inputBox := newStyle().Padding(1, 3).Background(lipgloss.Color("#222")).Render(m.textInput.View())
		interactiveElement = lipgloss.JoinVertical(lipgloss.Center, prompt, inputBox)
	}

	textBoxWidth := lipgloss.Width(interactiveElement)
	textBoxHeight := lipgloss.Height(interactiveElement)
	textBoxStartX := orbWidth/2 - textBoxWidth/2
	textBoxStartY := visibleOrbHeight/2 - textBoxHeight/2 + 3
	textBoxLines := strings.Split(interactiveElement, "\n")

	// Orb rendering with textbox overlay
	var lines []string
	for y := 0; y < visibleOrbHeight; y++ {
		isTextBoxLine := y >= textBoxStartY && y < textBoxStartY+textBoxHeight

		if isTextBoxLine {
			leftOrb := ""
			for x := 0; x < textBoxStartX; x++ {
				leftOrb += renderOrbPixel(x, y, orbWidth, orbHeight, radius, m.frame, palette, newStyle)
			}
			textBoxLine := textBoxLines[y-textBoxStartY]
			rightOrb := ""
			for x := textBoxStartX + textBoxWidth; x < orbWidth; x++ {
				rightOrb += renderOrbPixel(x, y, orbWidth, orbHeight, radius, m.frame, palette, newStyle)
			}
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, leftOrb, textBoxLine, rightOrb))
		} else {
			line := ""
			for x := 0; x < orbWidth; x++ {
				line += renderOrbPixel(x, y, orbWidth, orbHeight, radius, m.frame, palette, newStyle)
			}
			lines = append(lines, line)
		}
	}
	ball := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Instructions
	instructions := newStyle().Foreground(lipgloss.Color("#626262")).Render("\nPress Ctrl+C to quit.")

	// Final layout
	return lipgloss.JoinVertical(lipgloss.Left, headerView, ball, instructions)
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "no active PTY")
		return nil, nil
	}
	renderer := bubbletea.MakeRenderer(s)
	m := initialModel()
	m.width = pty.Window.Width
	m.height = pty.Window.Height
	m.renderer = renderer
	m.textInput.TextStyle = renderer.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#222"))
	m.spinner.Style = renderer.NewStyle().Foreground(lipgloss.Color("155"))
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func main() {
	sshFlag := flag.Bool("ssh", false, "run as ssh server")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	if *sshFlag {
		s, err := wish.NewServer(
			wish.WithAddress(":2222"),
			wish.WithHostKeyPath(".ssh/orb_host_key"),
			wish.WithMiddleware(
				bubbletea.Middleware(teaHandler),
				logging.Middleware(),
			),
		)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("starting ssh server on port 2222")
		if err := s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}

	} else {
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
			os.Exit(1)
		}
	}
}
