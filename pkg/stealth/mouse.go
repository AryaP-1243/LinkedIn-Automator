package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
)

type Point struct {
	X float64
	Y float64
}

type MouseController struct {
	config *config.MouseMovementConfig
	log    *logger.Logger
	rand   *rand.Rand
}

func NewMouseController(cfg *config.MouseMovementConfig) *MouseController {
	return &MouseController{
		config: cfg,
		log:    logger.WithComponent("mouse"),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *MouseController) GeneratePath(start, end Point) []Point {
	if !m.config.Enabled {
		return []Point{start, end}
	}

	// Calculate distance
	dist := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))

	// Balanced density for smoothness without being too slow
	numSteps := int(dist / 3.0) // Increased divisor for fewer steps = faster
	if numSteps < 30 {
		numSteps = 30 // Reduced minimum
	}
	if numSteps > 200 {
		numSteps = 200 // Reduced cap for faster movement
	}

	path := m.generateBezierPath(start, end, numSteps)

	if m.config.OvershootEnabled && m.rand.Float64() < 0.3 {
		path = m.addOvershoot(path, end)
	}

	if m.config.MicroMovements {
		path = m.addMicroMovements(path)
	}

	return path
}

func (m *MouseController) generateBezierPath(start, end Point, numSteps int) []Point {
	controlPoints := m.generateControlPoints(start, end)
	// numSteps is passed in
	path := make([]Point, 0, numSteps)

	for i := 0; i <= numSteps; i++ {
		t := float64(i) / float64(numSteps)
		t = m.applyEasing(t)
		point := m.bezierPoint(controlPoints, t)
		path = append(path, point)
	}

	return path
}

func (m *MouseController) generateControlPoints(start, end Point) []Point {
	complexity := m.config.BezierComplexity
	if complexity < 2 {
		complexity = 2
	}

	points := make([]Point, complexity+2)
	points[0] = start
	points[len(points)-1] = end

	dx := end.X - start.X
	dy := end.Y - start.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// Guard against zero-distance to avoid division by zero and NaN coordinates
	if distance == 0 {
		return []Point{start, end}
	}

	for i := 1; i < len(points)-1; i++ {
		progress := float64(i) / float64(len(points)-1)

		deviation := distance * 0.3 * (m.rand.Float64() - 0.5)

		perpX := -dy / distance * deviation
		perpY := dx / distance * deviation

		points[i] = Point{
			X: start.X + dx*progress + perpX,
			Y: start.Y + dy*progress + perpY,
		}
	}

	return points
}

// REMOVED calculateSteps as it's no longer used by GeneratePath directly
// reusing logic inside GeneratePath for better control

func (m *MouseController) applyEasing(t float64) float64 {
	return t * t * (3 - 2*t)
}

func (m *MouseController) bezierPoint(points []Point, t float64) Point {
	n := len(points) - 1
	x, y := 0.0, 0.0
	for i, p := range points {
		b := bernstein(n, i, t)
		x += p.X * b
		y += p.Y * b
	}
	return Point{X: x, Y: y}
}

func bernstein(n, k int, t float64) float64 {
	return float64(binomial(n, k)) * math.Pow(t, float64(k)) * math.Pow(1-t, float64(n-k))
}

func binomial(n, k int) int {
	if k > n-k {
		k = n - k
	}
	result := 1
	for i := 0; i < k; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}

func (m *MouseController) addOvershoot(path []Point, target Point) []Point {
	if len(path) == 0 {
		return path
	}

	last := path[len(path)-1]
	dx := target.X - last.X
	dy := target.Y - last.Y

	overshootDistance := 5 + m.rand.Float64()*15

	overshoot := Point{
		X: target.X + dx*overshootDistance/100,
		Y: target.Y + dy*overshootDistance/100,
	}

	path = append(path, overshoot)

	// Generate correction path with fixed steps for smoothness
	correctionPath := m.generateBezierPath(overshoot, target, 20)
	path = append(path, correctionPath[1:]...)

	return path
}

func (m *MouseController) addMicroMovements(path []Point) []Point {
	result := make([]Point, 0, len(path))

	for i, p := range path {
		jitterX := (m.rand.Float64() - 0.5) * 2
		jitterY := (m.rand.Float64() - 0.5) * 2

		result = append(result, Point{
			X: p.X + jitterX,
			Y: p.Y + jitterY,
		})

		if i > 0 && i < len(path)-1 && m.rand.Float64() < 0.1 {
			pauseJitterX := (m.rand.Float64() - 0.5) * 0.5
			pauseJitterY := (m.rand.Float64() - 0.5) * 0.5
			result = append(result, Point{
				X: p.X + pauseJitterX,
				Y: p.Y + pauseJitterY,
			})
		}
	}

	return result
}

func (m *MouseController) GetMovementDuration(path []Point) time.Duration {
	if len(path) < 2 {
		return 0
	}

	totalDistance := 0.0
	for i := 1; i < len(path); i++ {
		dx := path[i].X - path[i-1].X
		dy := path[i].Y - path[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	speed := m.config.MinSpeed + m.rand.Float64()*(m.config.MaxSpeed-m.config.MinSpeed)
	baseDuration := totalDistance / (speed * 500)

	variation := 1 + (m.rand.Float64()-0.5)*0.3
	finalDuration := baseDuration * variation

	return time.Duration(finalDuration * float64(time.Second))
}

func (m *MouseController) GenerateHoverPath(around Point, duration time.Duration) []Point {
	numPoints := int(duration.Seconds() * 10)
	if numPoints < 3 {
		numPoints = 3
	}

	path := make([]Point, numPoints)
	radius := 5.0

	for i := 0; i < numPoints; i++ {
		angle := float64(i) * 2 * math.Pi / float64(numPoints)
		r := radius * (0.5 + m.rand.Float64()*0.5)
		path[i] = Point{
			X: around.X + r*math.Cos(angle) + (m.rand.Float64()-0.5)*2,
			Y: around.Y + r*math.Sin(angle) + (m.rand.Float64()-0.5)*2,
		}
	}

	return path
}
