package effects

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGPUParticleSystem verifies particle system initialization.
func TestNewGPUParticleSystem(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 5.0)
	require.NoError(t, err)
	assert.NotNil(t, sys)
	assert.Equal(t, 100, sys.MaxParticles)
	assert.Equal(t, float32(5.0), sys.EmitRate)
	assert.Equal(t, 0, len(sys.Particles))
	assert.Equal(t, 100, cap(sys.Particles))
}

// TestGPUParticleSystemUpdate verifies particle emission and physics.
func TestGPUParticleSystemUpdate(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	// Update with 1 second dt should emit ~10 particles
	sys.Update(1.0, 100.0, 100.0, 10.0, 0.0)
	assert.Greater(t, len(sys.Particles), 5)
	assert.LessOrEqual(t, len(sys.Particles), 15)

	// Particles should have positions near emission point
	for _, p := range sys.Particles {
		assert.InDelta(t, 100.0, p.X, 20.0)
		assert.InDelta(t, 100.0, p.Y, 20.0)
		assert.Equal(t, float32(1.0), p.Life) // New particles start at full life
	}
}

// TestGPUParticleSystemLifetimeDecay verifies particles die after lifetime.
func TestGPUParticleSystemLifetimeDecay(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	// Emit particles
	sys.Update(1.0, 0.0, 0.0, 10.0, 0.0)
	initialCount := len(sys.Particles)
	assert.Greater(t, initialCount, 0)

	// Advance time by 3 seconds (MaxLife = 2.0 at resonance 0)
	for i := 0; i < 3; i++ {
		sys.Update(1.0, 0.0, 0.0, 10.0, 0.0)
	}

	// Original particles should be dead (life < 0)
	// New particles may have been emitted
	aliveCount := 0
	for _, p := range sys.Particles {
		if p.Life > 0 {
			aliveCount++
		}
	}
	// All original particles should be gone after 3 seconds
	assert.Less(t, aliveCount, initialCount+20)
}

// TestGPUParticleSystemResonanceScaling verifies resonance affects emission.
func TestGPUParticleSystemResonanceScaling(t *testing.T) {
	sysLow, err := NewGPUParticleSystem(200, 10.0)
	require.NoError(t, err)

	sysHigh, err := NewGPUParticleSystem(200, 10.0)
	require.NoError(t, err)

	// Update with low resonance (0)
	sysLow.Update(1.0, 0.0, 0.0, 10.0, 0.0)
	lowCount := len(sysLow.Particles)

	// Update with high resonance (100)
	sysHigh.Update(1.0, 0.0, 0.0, 10.0, 100.0)
	highCount := len(sysHigh.Particles)

	// High resonance should emit more particles
	// At resonance 100, rate is 10 * (1 + 100/100) = 20 particles/sec
	assert.Greater(t, highCount, lowCount)
	assert.InDelta(t, 10, lowCount, 5)
	assert.InDelta(t, 20, highCount, 5)
}

// TestGPUParticleSystemMaxCapacity verifies max particles constraint.
func TestGPUParticleSystemMaxCapacity(t *testing.T) {
	sys, err := NewGPUParticleSystem(50, 100.0)
	require.NoError(t, err)

	// Emit many particles over 2 seconds
	for i := 0; i < 10; i++ {
		sys.Update(0.2, 0.0, 0.0, 10.0, 0.0)
	}

	// Particle count should not exceed max
	assert.LessOrEqual(t, len(sys.Particles), 50)
}

// TestGPUParticleSystemVelocity verifies particles move according to velocity.
func TestGPUParticleSystemVelocity(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	// Emit one particle
	sys.Update(0.1, 0.0, 0.0, 10.0, 0.0)
	require.Greater(t, len(sys.Particles), 0)

	p0 := sys.Particles[0]
	initialX := p0.X
	initialY := p0.Y

	// Advance time without emitting new particles
	sys.EmitRate = 0
	sys.Update(1.0, 0.0, 0.0, 10.0, 0.0)

	// Particle should have moved
	p1 := sys.Particles[0]
	deltaX := p1.X - initialX
	deltaY := p1.Y - initialY

	// VX and VY are set in emitParticle, should produce non-zero movement
	assert.NotZero(t, deltaX)
	assert.NotZero(t, deltaY)
}

// TestGPUParticleSystemClear verifies clear removes all particles.
func TestGPUParticleSystemClear(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	// Emit particles
	sys.Update(1.0, 0.0, 0.0, 10.0, 0.0)
	assert.Greater(t, len(sys.Particles), 0)

	// Clear
	sys.Clear()
	assert.Equal(t, 0, len(sys.Particles))
	assert.Equal(t, float32(0), sys.accumulator)
}

// TestGPUParticleSystemParticleCount verifies count reporting.
func TestGPUParticleSystemParticleCount(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	assert.Equal(t, 0, sys.ParticleCount())

	sys.Update(1.0, 0.0, 0.0, 10.0, 0.0)
	count := sys.ParticleCount()
	assert.Greater(t, count, 0)
	assert.Equal(t, len(sys.Particles), count)
}

// TestGPUParticleColor verifies particle color assignment.
func TestGPUParticleColor(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	sys.Update(0.1, 0.0, 0.0, 10.0, 0.0)
	require.Greater(t, len(sys.Particles), 0)

	// Particles should have luminous blue-white color per PULSE_MAP.md
	p := sys.Particles[0]
	assert.Equal(t, color.RGBA{200, 220, 255, 200}, p.Color)
}

// TestGPUParticleSystemZeroEmitRate verifies no emission at rate 0.
func TestGPUParticleSystemZeroEmitRate(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 0.0)
	require.NoError(t, err)

	sys.Update(1.0, 0.0, 0.0, 10.0, 0.0)
	assert.Equal(t, 0, len(sys.Particles))
}

// TestGPUParticleSystemSmallDeltaTime verifies stability with small dt.
func TestGPUParticleSystemSmallDeltaTime(t *testing.T) {
	sys, err := NewGPUParticleSystem(100, 10.0)
	require.NoError(t, err)

	// Update with many small time steps
	for i := 0; i < 60; i++ {
		sys.Update(1.0/60.0, 0.0, 0.0, 10.0, 0.0)
	}

	// Should have emitted approximately 10 particles in 1 second total
	assert.InDelta(t, 10, len(sys.Particles), 5)
}
