package game

import (
	"math"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// uiRect draws a filled rectangle with an optional border.
func uiRect(ctx *whisky.Context, r geom.Rect, fill geom.Color, border geom.Color, borderWidth float64) {
	ctx.DrawRect(r, fill)
	if borderWidth > 0 {
		// Top
		ctx.DrawRect(geom.Rect{X: r.X, Y: r.Y, W: r.W, H: borderWidth}, border)
		// Bottom
		ctx.DrawRect(geom.Rect{X: r.X, Y: r.Y + r.H - borderWidth, W: r.W, H: borderWidth}, border)
		// Left
		ctx.DrawRect(geom.Rect{X: r.X, Y: r.Y, W: borderWidth, H: r.H}, border)
		// Right
		ctx.DrawRect(geom.Rect{X: r.X + r.W - borderWidth, Y: r.Y, W: borderWidth, H: r.H}, border)
	}
}

// uiButton represents a clickable button.
type uiButton struct {
	Text     string
	Rect     geom.Rect
	Enabled  bool
	OnClick  func()
}

// uiMenu is a vertical list of buttons with keyboard and mouse navigation.
type uiMenu struct {
	Buttons      []*uiButton
	Selected     int
	ConfirmDelay float64 // frames remaining before accepting input
}

func newUIMenu() *uiMenu {
	return &uiMenu{Selected: -1}
}

func (m *uiMenu) AddButton(text string, onClick func()) *uiButton {
	btn := &uiButton{Text: text, Enabled: true, OnClick: onClick}
	m.Buttons = append(m.Buttons, btn)
	if m.Selected < 0 {
		m.Selected = 0
	}
	return btn
}

func (m *uiMenu) Update(ctx *whisky.Context, dt float64) {
	if m == nil || len(m.Buttons) == 0 {
		return
	}

	// Cooldown after state change to prevent accidental double-presses.
	if m.ConfirmDelay > 0 {
		m.ConfirmDelay -= dt
		return
	}

	// Keyboard navigation.
	if ctx.Input.JustPressed("menu_up") {
		m.moveSelection(-1)
	}
	if ctx.Input.JustPressed("menu_down") {
		m.moveSelection(1)
	}
	if ctx.Input.JustPressed("menu_confirm") {
		m.activateSelected()
	}

	// Mouse navigation.
	mx, my := ctx.Mouse().Position()
	mousePos := geom.Vec2{X: mx, Y: my}
	for i, btn := range m.Buttons {
		if !btn.Enabled {
			continue
		}
		if pointInRect(mousePos, btn.Rect) {
			m.Selected = i
			if ctx.Mouse().JustPressed(input.MouseButtonLeft) {
				m.activateSelected()
			}
		}
	}
}

func (m *uiMenu) moveSelection(delta int) {
	if len(m.Buttons) == 0 {
		return
	}
	for {
		m.Selected += delta
		if m.Selected < 0 {
			m.Selected = len(m.Buttons) - 1
		}
		if m.Selected >= len(m.Buttons) {
			m.Selected = 0
		}
		if m.Buttons[m.Selected].Enabled {
			break
		}
	}
}

func (m *uiMenu) activateSelected() {
	if m.Selected >= 0 && m.Selected < len(m.Buttons) {
		btn := m.Buttons[m.Selected]
		if btn.Enabled && btn.OnClick != nil {
			btn.OnClick()
		}
	}
}

func (m *uiMenu) LayoutCentered(startY, buttonW, buttonH, gap float64, vw, vh float64) {
	if len(m.Buttons) == 0 {
		return
	}
	totalH := float64(len(m.Buttons))*buttonH + float64(len(m.Buttons)-1)*gap
	x := (vw - buttonW) / 2
	y := startY
	if y <= 0 {
		y = (vh - totalH) / 2
	}
	for _, btn := range m.Buttons {
		btn.Rect = geom.Rect{X: x, Y: y, W: buttonW, H: buttonH}
		y += buttonH + gap
	}
}

func (m *uiMenu) Draw(ctx *whisky.Context) {
	for i, btn := range m.Buttons {
		col := uiButtonColor
		if !btn.Enabled {
			col = uiDisabledColor
		} else if i == m.Selected {
			col = uiButtonHover
		}
		border := geom.RGBA(0.35, 0.38, 0.48, 1)
		if i == m.Selected {
			border = uiHighlightColor
		}
		uiRect(ctx, btn.Rect, col, border, 1)

		// Draw button text label.
		textCol := uiTextColor
		if !btn.Enabled {
			textCol = geom.RGBA(0.50, 0.50, 0.52, 1)
		}
		if btn.Text != "" {
			textScale := 0.55
			glyphW := 7.0 * textScale  // base glyph width
			glyphH := 13.0 * textScale // base glyph height
			textW := float64(len(btn.Text)) * glyphW
			tx := btn.Rect.X + (btn.Rect.W-textW)/2
			ty := btn.Rect.Y + (btn.Rect.H-glyphH)/2
			ctx.DrawText(btn.Text, geom.Vec2{X: tx, Y: ty}, textCol, textScale)
		}
	}
}

// uiSlider represents a horizontal slider for numeric values.
type uiSlider struct {
	Label    string
	Rect     geom.Rect
	Value    float64 // 0..1
	Min, Max float64
	Step     float64
	OnChange func(float64)
	Dragging bool
}

func (s *uiSlider) Update(ctx *whisky.Context) {
	if s == nil {
		return
	}
	mx, my := ctx.Mouse().Position()
	mousePos := geom.Vec2{X: mx, Y: my}
	inRect := pointInRect(mousePos, s.Rect)

	if ctx.Mouse().JustPressed(input.MouseButtonLeft) && inRect {
		s.Dragging = true
	}
	if !ctx.Mouse().ButtonPressed(input.MouseButtonLeft) {
		s.Dragging = false
	}

	if s.Dragging || (ctx.Mouse().JustPressed(input.MouseButtonLeft) && inRect) {
		ratio := (mx - s.Rect.X) / s.Rect.W
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		s.Value = s.Min + ratio*(s.Max-s.Min)
		if s.Step > 0 {
			s.Value = math.Round(s.Value/s.Step) * s.Step
		}
		if s.Value < s.Min {
			s.Value = s.Min
		}
		if s.Value > s.Max {
			s.Value = s.Max
		}
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
	}
}

func (s *uiSlider) Draw(ctx *whisky.Context) {
	// Label to the left of the track.
	if s.Label != "" {
		labelScale := 0.5
		glyphH := 13.0 * labelScale
		lx := s.Rect.X - float64(len(s.Label))*7.0*labelScale - 4
		ly := s.Rect.Y + (s.Rect.H-glyphH)/2
		ctx.DrawText(s.Label, geom.Vec2{X: lx, Y: ly}, uiTextColor, labelScale)
	}
	// Track
	uiRect(ctx, s.Rect, geom.RGBA(0.12, 0.12, 0.16, 1), geom.RGBA(0.30, 0.32, 0.40, 1), 1)
	// Fill
	ratio := (s.Value - s.Min) / (s.Max - s.Min)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	fillW := s.Rect.W * ratio
	if fillW > 2 {
		ctx.DrawRect(geom.Rect{X: s.Rect.X + 1, Y: s.Rect.Y + 1, W: fillW - 2, H: s.Rect.H - 2}, uiSuccessColor)
	}
	// Thumb
	thumbX := s.Rect.X + s.Rect.W*ratio - 2
	ctx.DrawRect(geom.Rect{X: thumbX, Y: s.Rect.Y - 1, W: 5, H: s.Rect.H + 2}, uiTextColor)
}

func pointInRect(p geom.Vec2, r geom.Rect) bool {
	return p.X >= r.X && p.X <= r.X+r.W && p.Y >= r.Y && p.Y <= r.Y+r.H
}

// uiPanel draws a centered modal panel.
func uiPanel(ctx *whisky.Context, x, y, w, h float64) {
	// Shadow
	ctx.DrawRect(geom.Rect{X: x + 4, Y: y + 4, W: w, H: h}, geom.RGBA(0, 0, 0, 0.4))
	// Panel
	uiRect(ctx, geom.Rect{X: x, Y: y, W: w, H: h}, uiPanelColor, geom.RGBA(0.25, 0.27, 0.35, 1), 1)
}

// uiTitle draws a decorative title bar at the top of a panel with centered text.
func uiTitle(ctx *whisky.Context, x, y, w, titleH float64, title string) {
	ctx.DrawRect(geom.Rect{X: x, Y: y, W: w, H: titleH}, geom.RGBA(0.14, 0.16, 0.22, 1))
	ctx.DrawRect(geom.Rect{X: x, Y: y + titleH - 1, W: w, H: 1}, uiHighlightColor)
	if title != "" {
		textScale := 0.6
		glyphW := 7.0 * textScale
		glyphH := 13.0 * textScale
		textW := float64(len(title)) * glyphW
		tx := x + (w-textW)/2
		ty := y + (titleH-glyphH)/2
		ctx.DrawText(title, geom.Vec2{X: tx, Y: ty}, uiTitleColor, textScale)
	}
}
