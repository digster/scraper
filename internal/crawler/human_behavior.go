package crawler

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

// BezierPoint represents a point on a Bezier curve
type BezierPoint struct {
	X, Y float64
}

// cubicBezier calculates a point on a cubic Bezier curve
func cubicBezier(p0, p1, p2, p3 BezierPoint, t float64) BezierPoint {
	// B(t) = (1-t)^3*P0 + 3*(1-t)^2*t*P1 + 3*(1-t)*t^2*P2 + t^3*P3
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	return BezierPoint{
		X: uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X,
		Y: uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y,
	}
}

// GenerateBezierCurve creates a natural-looking mouse movement path using cubic Bezier curves
func GenerateBezierCurve(start, end BezierPoint, steps int) []BezierPoint {
	// Generate random control points for natural curve
	// Control points are offset from the direct line to create curvature
	distX := end.X - start.X
	distY := end.Y - start.Y

	// First control point - closer to start
	ctrl1 := BezierPoint{
		X: start.X + distX*0.25 + (rand.Float64()-0.5)*math.Abs(distY)*0.5,
		Y: start.Y + distY*0.25 + (rand.Float64()-0.5)*math.Abs(distX)*0.5,
	}

	// Second control point - closer to end
	ctrl2 := BezierPoint{
		X: start.X + distX*0.75 + (rand.Float64()-0.5)*math.Abs(distY)*0.5,
		Y: start.Y + distY*0.75 + (rand.Float64()-0.5)*math.Abs(distX)*0.5,
	}

	points := make([]BezierPoint, steps)
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		points[i] = cubicBezier(start, ctrl1, ctrl2, end, t)
	}
	return points
}

// MoveMouseNaturally moves the mouse along a Bezier curve with variable speed
func MoveMouseNaturally(ctx context.Context, startX, startY, targetX, targetY int) error {
	// Calculate number of steps based on distance
	dist := math.Sqrt(float64((targetX-startX)*(targetX-startX) + (targetY-startY)*(targetY-startY)))
	steps := int(math.Max(20, math.Min(50, dist/10))) + rand.Intn(10)

	points := GenerateBezierCurve(
		BezierPoint{float64(startX), float64(startY)},
		BezierPoint{float64(targetX), float64(targetY)},
		steps,
	)

	for i, p := range points {
		// Variable delay - faster in middle, slower at edges (ease-in-out)
		progress := float64(i) / float64(len(points)-1)
		easeValue := 0.5 - 0.5*math.Cos(progress*math.Pi) // Sine ease-in-out
		baseDelay := 5 + rand.Intn(15)
		delay := time.Duration(float64(baseDelay)*(1+easeValue*0.5)) * time.Millisecond

		err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchMouseEvent(input.MouseMoved, p.X, p.Y).Do(ctx)
			}),
		)
		if err != nil {
			return fmt.Errorf("mouse move failed: %w", err)
		}
		time.Sleep(delay)
	}
	return nil
}

// TypeWithNaturalDelay types text with human-like variable delays between keystrokes
func TypeWithNaturalDelay(ctx context.Context, selector, text string) error {
	// First focus the element
	if err := chromedp.Run(ctx, chromedp.Focus(selector)); err != nil {
		return fmt.Errorf("failed to focus element: %w", err)
	}

	for _, char := range text {
		// Base delay 50-150ms
		delay := time.Duration(50+rand.Intn(100)) * time.Millisecond

		// 10% chance of a longer "thinking" pause
		if rand.Float64() < 0.1 {
			delay += time.Duration(200+rand.Intn(300)) * time.Millisecond
		}

		// 5% chance of typing a wrong character and correcting (typo simulation)
		if rand.Float64() < 0.05 {
			wrongChar := string(rune('a' + rand.Intn(26)))
			if err := chromedp.Run(ctx, chromedp.SendKeys(selector, wrongChar, chromedp.ByQuery)); err != nil {
				return fmt.Errorf("failed to type character: %w", err)
			}
			time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)
			if err := chromedp.Run(ctx, chromedp.SendKeys(selector, kb.Backspace, chromedp.ByQuery)); err != nil {
				return fmt.Errorf("failed to backspace: %w", err)
			}
			time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
		}

		if err := chromedp.Run(ctx, chromedp.SendKeys(selector, string(char), chromedp.ByQuery)); err != nil {
			return fmt.Errorf("failed to type character: %w", err)
		}
		time.Sleep(delay)
	}
	return nil
}

// ScrollNaturally scrolls with momentum simulation (ease-out effect)
func ScrollNaturally(ctx context.Context, deltaY int) error {
	steps := 10 + rand.Intn(5)
	totalScrolled := 0.0
	targetScroll := float64(deltaY)

	for i := 0; i < steps; i++ {
		// Ease-out effect - faster at start, slower at end
		progress := float64(i+1) / float64(steps)
		easeOut := 1 - math.Pow(1-progress, 3) // Cubic ease-out

		targetSoFar := targetScroll * easeOut
		stepScroll := targetSoFar - totalScrolled
		totalScrolled = targetSoFar

		scrollScript := fmt.Sprintf("window.scrollBy(0, %f)", stepScroll)
		if err := chromedp.Run(ctx, chromedp.Evaluate(scrollScript, nil)); err != nil {
			return fmt.Errorf("scroll failed: %w", err)
		}

		// Variable delay between scroll steps
		delay := time.Duration(20+rand.Intn(30)) * time.Millisecond
		time.Sleep(delay)
	}
	return nil
}

// ClickWithOffset clicks with a small random offset from the element center
func ClickWithOffset(ctx context.Context, selector string) error {
	// Get element bounds
	var rect struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}

	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return {x: rect.x, y: rect.y, width: rect.width, height: rect.height};
		})()
	`, selector)

	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &rect)); err != nil {
		return fmt.Errorf("failed to get element bounds: %w", err)
	}

	if rect.Width == 0 || rect.Height == 0 {
		return fmt.Errorf("element not found or has no dimensions: %s", selector)
	}

	// Calculate center with random offset (within 30% of element size)
	offsetX := (rand.Float64() - 0.5) * rect.Width * 0.3
	offsetY := (rand.Float64() - 0.5) * rect.Height * 0.3

	targetX := int(rect.X + rect.Width/2 + offsetX)
	targetY := int(rect.Y + rect.Height/2 + offsetY)

	// Click at the offset position
	if err := chromedp.Run(ctx,
		chromedp.MouseClickXY(float64(targetX), float64(targetY)),
	); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	return nil
}

// RandomActionDelay returns a random delay between 100-500ms for use between actions
func RandomActionDelay() time.Duration {
	return time.Duration(100+rand.Intn(400)) * time.Millisecond
}

// RandomTypingDelay returns a random delay for typing (50-150ms base)
func RandomTypingDelay() time.Duration {
	delay := time.Duration(50+rand.Intn(100)) * time.Millisecond
	// Occasional longer pause
	if rand.Float64() < 0.1 {
		delay += time.Duration(200+rand.Intn(300)) * time.Millisecond
	}
	return delay
}

// RandomScrollDelay returns a random delay for scroll steps
func RandomScrollDelay() time.Duration {
	return time.Duration(20+rand.Intn(30)) * time.Millisecond
}

// HumanBehavior provides a convenience struct for applying human behavior settings
type HumanBehavior struct {
	config AntiBotConfig
}

// NewHumanBehavior creates a new HumanBehavior helper
func NewHumanBehavior(config AntiBotConfig) *HumanBehavior {
	return &HumanBehavior{config: config}
}

// ApplyMouseMovement moves mouse naturally if enabled, otherwise does nothing
func (h *HumanBehavior) ApplyMouseMovement(ctx context.Context, startX, startY, targetX, targetY int) error {
	if !h.config.NaturalMouseMovement {
		return nil
	}
	return MoveMouseNaturally(ctx, startX, startY, targetX, targetY)
}

// ApplyTyping types with natural delays if enabled, otherwise types directly
func (h *HumanBehavior) ApplyTyping(ctx context.Context, selector, text string) error {
	if !h.config.RandomTypingDelays {
		return chromedp.Run(ctx, chromedp.SendKeys(selector, text, chromedp.ByQuery))
	}
	return TypeWithNaturalDelay(ctx, selector, text)
}

// ApplyScroll scrolls naturally if enabled, otherwise scrolls directly
func (h *HumanBehavior) ApplyScroll(ctx context.Context, deltaY int) error {
	if !h.config.NaturalScrolling {
		script := fmt.Sprintf("window.scrollBy(0, %d)", deltaY)
		return chromedp.Run(ctx, chromedp.Evaluate(script, nil))
	}
	return ScrollNaturally(ctx, deltaY)
}

// ApplyClick clicks with offset if enabled, otherwise clicks directly
func (h *HumanBehavior) ApplyClick(ctx context.Context, selector string) error {
	if !h.config.RandomClickOffset {
		return chromedp.Run(ctx, chromedp.Click(selector, chromedp.ByQuery))
	}
	return ClickWithOffset(ctx, selector)
}

// ApplyActionDelay applies a random delay if enabled
func (h *HumanBehavior) ApplyActionDelay() {
	if h.config.RandomActionDelays {
		time.Sleep(RandomActionDelay())
	}
}
