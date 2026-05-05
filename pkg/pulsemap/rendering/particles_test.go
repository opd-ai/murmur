// Package rendering tests verify ambient particle field.
package rendering

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAmbientParticleField(t *testing.T) {
	field := NewAmbientParticleField()
	assert.NotNil(t, field)
	assert.Equal(t, 80, field.maxParticles)
	assert.Equal(t, 2.0, field.spawnRate)
	assert.Equal(t, 0, field.ParticleCount())
}

func TestAmbientParticleFieldSetMaxParticles(t *testing.T) {
	field := NewAmbientParticleField()

	field.SetMaxParticles(150)
	assert.Equal(t, 150, field.maxParticles)
}

func TestAmbientParticleFieldSetSpawnRate(t *testing.T) {
	field := NewAmbientParticleField()

	field.SetSpawnRate(5.0)
	assert.Equal(t, 5.0, field.spawnRate)
}

func TestAmbientParticleFieldParticleCount(t *testing.T) {
	field := NewAmbientParticleField()
	assert.Equal(t, 0, field.ParticleCount())
}
