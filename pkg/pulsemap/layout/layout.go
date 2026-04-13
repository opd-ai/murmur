// Package layout provides the force-directed graph engine for the Pulse Map.
// Per PULSE_MAP.md, the layout uses Fruchterman-Reingold with Barnes-Hut
// optimization for graphs exceeding 500 nodes.
package layout

// BarnesHutThreshold is the node count above which Barnes-Hut is used.
const BarnesHutThreshold = 500

// TODO: Implement force-directed layout per PULSE_MAP.md.
