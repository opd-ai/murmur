# Pulse Map

**Category:** Interface — Primary Visualization and Interaction Surface
**Version:** 0.4
**Status:** Draft

---

## Overview

The Pulse Map is MURMUR's primary user interface. It is a real-time, interactive visualization of the network as a living topology — a map of nodes, connections, and activity rendered as an animated, explorable two-dimensional space. Every user, every connection, every Wave, every anonymous mechanic, and every social signal in MURMUR has a visual representation on the Pulse Map. It is both the way users see the network and the way users interact with it.

The Pulse Map replaces the feed-based paradigm of conventional social platforms with a spatial paradigm. Users do not scroll through a timeline; they navigate a landscape. Content is not presented as a sequence; it is distributed across a topology. Attention is not captured by algorithmic ranking; it is guided by visual salience — size, brightness, motion, proximity, and color. The Pulse Map makes the network's social structure visible and tangible in a way that text lists and card feeds cannot.

The Pulse Map operates on both the Surface Layer and the Anonymous Layer. Surface Layer users see a map of identified nodes and public connections. Anonymous Layer users (Hybrid mode and above) see an overlaid map of Specter nodes, anonymous connections, and anonymous mechanics. Fortress-mode users see only the Anonymous Layer map. The two layers are visually distinct but spatially co-located, creating a sense of two worlds occupying the same space.

---

## Coordinate System and Layout

### Topology-Driven Positioning

Node positions on the Pulse Map are not arbitrary. They are computed from the network's actual connection topology using a force-directed layout algorithm. The algorithm treats nodes as charged particles that repel each other and connections as springs that pull connected nodes together. The result is a layout where densely connected clusters of nodes are positioned close together, and weakly connected nodes drift to the periphery.

The force-directed layout runs continuously on each client, updating positions as nodes join, leave, form connections, and sever them. The layout is deterministic given the same topology — two clients observing the same network state will compute approximately the same layout (minor floating-point variations notwithstanding). This means all users see roughly the same spatial arrangement of the network, creating a shared sense of place.

### Force Model Parameters

The layout algorithm uses four forces.

**Repulsion.** Every pair of nodes exerts a repulsive force on each other, proportional to the inverse square of their distance. This prevents nodes from overlapping and distributes them across the available space. The repulsion constant is tuned so that a network of 1,000 nodes fills the viewport comfortably at default zoom.

**Spring Attraction.** Every connection acts as a spring pulling the two connected nodes toward each other. The spring's rest length is proportional to the connection's age (older connections have shorter rest lengths, pulling nodes closer over time). The spring constant is uniform across all connections. This force creates clusters of tightly connected nodes.

**Center Gravity.** A weak gravitational force pulls all nodes toward the center of the viewport. This prevents the layout from drifting unboundedly and ensures the network remains centered. The gravitational constant is very weak — it provides a gentle centering tendency without overriding the topology-driven forces.

**Damping.** A velocity damping force decelerates all nodes, preventing oscillation and ensuring the layout converges to a stable equilibrium. The damping coefficient is set high enough to prevent visible jitter but low enough to allow smooth transitions when the topology changes.

### Layout Update Cycle

The force-directed simulation runs at 30 ticks per second. Each tick computes the net force on every visible node, updates velocities, applies damping, and updates positions. The simulation stabilizes within 2–5 seconds after a topology change (a new node joining, a connection forming or breaking).

To manage computational cost, the simulation is hierarchical. Nodes beyond the current viewport are aggregated into cluster representatives — single virtual nodes representing groups of 10–50 nodes, with mass proportional to the cluster size. The force computation uses cluster representatives for distant nodes and individual nodes for nearby ones. This reduces the computational complexity from O(n²) to approximately O(n log n), making the layout feasible for networks of up to 100,000 nodes on commodity hardware.

### Viewport and Navigation

The Pulse Map viewport is a rectangular window into the two-dimensional layout space. Users navigate the map using standard pan and zoom gestures: drag to pan, pinch or scroll-wheel to zoom. Double-tap or double-click centers the viewport on the tapped/clicked node.

The zoom range spans from a macro view (showing the entire network as a galaxy of dots) to a micro view (showing individual node details, sigils, and active Waves). The zoom level determines the level of detail rendered, as described in the Visual Detail Levels section.

A minimap is displayed in the corner of the viewport, showing the full network layout at macro scale with the current viewport highlighted as a rectangle. Users can click or tap on the minimap to jump to a different region of the network.

### Ego-Centric View

By default, the Pulse Map is ego-centric: the current user's node is positioned at the center of the viewport, and the layout radiates outward from there. The user's direct connections are in the innermost ring, their connections' connections in the next ring, and so on. This ego-centric framing gives users a natural sense of their place in the network — they are at the center of their social world.

Users can toggle to a network-centric view, where the layout is centered on the network's topological center (the node or cluster with the highest closeness centrality). This view gives a more objective picture of the network's structure but sacrifices the personal framing.

### Layer Separation

When a user is in Hybrid mode (seeing both Surface Layer and Anonymous Layer), the two layers are rendered in the same coordinate space but visually separated by depth cues. Surface Layer nodes are rendered with full opacity and warm color tones. Anonymous Layer Specter nodes are rendered with reduced opacity and cool color tones (blues, purples, silvers). The two layers overlap spatially — a Specter node occupies the same approximate position as its owner's Surface node (because its connections mirror the owner's topology), but is visually distinct.

Users can adjust the layer blend using a slider control. At one extreme, only the Surface Layer is visible. At the other extreme, only the Anonymous Layer is visible. In the middle, both layers are visible with adjustable relative opacity. This blending control lets users focus on the layer that interests them while maintaining awareness of the other.

Fortress-mode users have no blend slider. Their Pulse Map shows only the Anonymous Layer.

---

## Node Rendering

### Node Anatomy

Each node on the Pulse Map represents a single identity — a main identity on the Surface Layer, or a Specter identity on the Anonymous Layer. A node is rendered as a composite visual element with several components.

**Core.** The node's central element is a filled circle. The circle's radius encodes the node's current social significance as computed locally by the observer: more active, more connected, or higher Resonance nodes have larger radii. The radius formula for Surface Layer nodes is `r = r_base + r_scale * ln(1 + connections + activity_30d)`, where `r_base` is 4 pixels, `r_scale` is 3 pixels, `connections` is the node's connection count as known to the observer, and `activity_30d` is the number of Waves published by the node in the trailing 30 days as observed by the observer. For Specter nodes, the radius formula substitutes Specter Resonance for the connection+activity metric: `r = r_base + r_scale * ln(1 + resonance)`.

**Sigil.** The node's procedurally generated sigil (as described in the Identity System specification) is rendered inside the core circle when the zoom level is sufficient to display it (micro and meso zoom levels). At macro zoom, the sigil is not rendered and the core is a plain colored circle.

**Ring.** A thin ring surrounds the core circle, providing a visual border. The ring's color indicates the node's current mode: no ring for Open mode, a single blue ring for Hybrid mode, and a double silver ring for Fortress mode. On the Surface Layer, the ring color indicates connection trust level when viewing a specific connection (green for verified, amber for pending, gray for unverified).

**Halo.** An optional outer glow effect that indicates active status. A node that has published a Wave within the last 60 minutes has a soft radial glow extending beyond the ring. The halo's intensity decays linearly from the most recent Wave publication, fading to invisible after 60 minutes of inactivity.

**Marks.** Specter Marks placed on the node are rendered as small sigil icons arrayed around the node's ring, slightly outside the border. Each Mark sigil is the marking Specter's sigil rendered at a miniature scale.

**Phantom Gifts.** Active Phantom Gift effects are rendered as animated overlays on the node, as described in the Anonymous Mechanics specification. Gift effects layer on top of the core, ring, and halo but behind Mark sigils.

### Node Color

Node color encodes identity information. On the Surface Layer, each node's core color is derived from the node's public key hash — the same hash used to generate the sigil. The color derivation extracts the first 3 bytes of the hash and maps them to a hue (byte 0 mapped to 0–360 degrees), saturation (byte 1 mapped to 40–80%), and lightness (byte 2 mapped to 40–60%). This ensures every node has a unique, deterministic color that matches its sigil's color palette.

Specter nodes use a restricted color palette: all Specter nodes are rendered in shades of blue, purple, silver, and white. The specific shade is derived from the Specter public key hash, but the hue is constrained to the 200–280 degree range (cool tones). This restricted palette visually distinguishes the Anonymous Layer from the Surface Layer at a glance.

### Node Labels

Node labels are rendered below the core circle. On the Surface Layer, the label displays the node's chosen display name (if set) or the first 8 characters of the public key fingerprint (if no display name is set). On the Anonymous Layer, the label displays the Specter's two-word pseudonym.

Labels are visible only at meso and micro zoom levels. At macro zoom, labels are hidden to reduce visual clutter. Labels use a semi-transparent dark background pill to ensure readability against varying map backgrounds.

### Node Selection and Interaction

Tapping or clicking a node selects it, triggering a selection animation (a brief scale-up pulse and a persistent highlight ring). Selecting a node opens the Node Detail Panel, a slide-in panel displaying the node's profile information, recent Waves, connections, and available interaction options (send Wave, send Phantom Gift, place Specter Mark, initiate Whisper Chain, join mini-game, etc.). The available options depend on the observer's mode and the target node's layer.

Long-pressing or right-clicking a node opens a quick-action radial menu with the most common interactions. The radial menu items are context-sensitive: different options appear for Surface Layer nodes vs. Specter nodes, for connected vs. unconnected nodes, and for the user's own node vs. others.

---

## Connection Rendering

### Connection Lines

Connections between nodes are rendered as curved lines (quadratic Bézier curves) between the connected nodes' core circles. The curve's control point is offset perpendicular to the straight-line path between the nodes, creating a gentle arc. The offset direction alternates for connections on the same node, preventing overlap.

Connection lines are rendered with low opacity (20–40% alpha) to prevent the map from becoming a dense tangle of lines at scale. At macro zoom, connections are rendered only for the user's direct connections and a sampled subset of the network's connections. At meso zoom, connections within the viewport are rendered. At micro zoom, all connections involving visible nodes are rendered.

### Connection Color and Style

Connection color encodes the connection type. Surface Layer connections between two identified users are rendered in warm tones (amber to gold). Specter connections between two anonymous identities are rendered in cool tones (blue to violet). Cross-layer connections (which do not exist — the two layers have separate connection graphs) are not applicable.

Connection line style encodes the connection's age. New connections (less than 7 days old) are rendered as dashed lines with a subtle animation (dashes flowing along the line like a signal traveling through a wire). Established connections (7–90 days old) are rendered as solid lines. Old connections (more than 90 days old) are rendered as solid lines with increased opacity and a subtle glow, visually reinforcing the network's long-term social bonds.

### Connection Activity Indicators

When a Wave propagates along a connection (from publisher to connected node), a brief pulse animation travels along the connection line in the direction of propagation. The pulse is a bright dot that moves from the source node to the destination node over 0.3 seconds, leaving a brief afterglow trail. This animation makes Wave propagation visually tangible — users can see information flowing through the network in real time.

During periods of high activity (many Waves propagating simultaneously), the pulse animations create a visual effect of the network "lighting up" — connections flickering and pulsing with flowing information. This effect is the origin of the name "Pulse Map."

---

## Visual Detail Levels

The Pulse Map renders different levels of detail depending on the current zoom level. There are three primary detail levels.

### Macro View

At the widest zoom, the entire network (or a large portion of it) is visible. Nodes are rendered as small colored dots without sigils, labels, or halos. Connections are rendered as faint lines, with only a sparse sample visible to prevent visual overload. Cluster structure is visible — dense regions of dots appear as bright clouds, sparse regions as scattered points. At this level, the Pulse Map resembles a star chart or galaxy visualization.

The macro view is useful for understanding the network's overall structure: identifying major clusters, peripheral regions, and the relative positions of communities. Users navigate the macro view to find regions of interest before zooming in.

Active Masked Events, mini-games, and Phantom Councils are visible at macro scale as special overlay icons — small animated glyphs positioned at the centroid of their participants' nodes. These icons serve as beacons, drawing users' attention to active social events they might want to explore.

### Meso View

At medium zoom, a neighborhood of 50–200 nodes is visible. Nodes are rendered with core circles, rings, halos, and labels. Sigils are visible but small. Connections are rendered with color and style encoding. Wave propagation pulses are visible. Mark sigils and Phantom Gift effects are visible but subtle.

The meso view is the primary navigation and browsing view. Users explore the network at this level, reading node labels, observing activity patterns, and identifying nodes they want to interact with.

### Micro View

At the closest zoom, 5–20 nodes are visible. Nodes are rendered at full detail: large core circles with clearly visible sigils, full labels with display names, all Marks, all Phantom Gift effects, and halos with full intensity. Connections are rendered with full detail including activity indicators.

The micro view is the interaction and reading view. Users zoom to this level to read Wave content, view node profiles, and perform interactions. Wave content is displayed as floating text cards attached to the publishing node (or in the Node Detail Panel for the selected node).

---

## Wave Visualization

### Wave Publication Animation

When a node publishes a Wave, the Pulse Map plays a publication animation centered on that node. The animation begins with a brief scale-up pulse of the node's core (expanding to 1.5x radius and returning over 0.3 seconds) accompanied by a radial shockwave — a ring of light that expands outward from the node at a constant rate, fading as it expands. The shockwave's color matches the node's core color.

The shockwave is purely cosmetic — it does not represent the actual gossip propagation pattern (which is determined by the gossip protocol and network topology, not a uniform radial expansion). However, it provides clear visual feedback that a Wave has been published and draws attention to the publishing node.

### Wave Content Display

Wave content is displayed in floating content cards. A content card is a semi-transparent rounded rectangle containing the Wave's text content, the publisher's label (display name or pseudonym), the publication timestamp, and interaction controls (amplify, reply, mute).

Content cards are visible at meso and micro zoom. At meso zoom, only the most recent content card per node is displayed, and it fades after 30 seconds. At micro zoom, the selected node's full Wave history (up to the 30-day content window) is displayed in the Node Detail Panel, and the most recent 3 Waves are displayed as stacked content cards near the node.

Reply Waves are displayed as content cards with a thin line connecting them to their parent Wave's card, creating a visual thread structure. Threads can be expanded or collapsed by tapping the connection line.

### Amplification Animation

When a Wave receives an amplification, a small star-burst animation plays at the amplifying node's position, and a bright pulse travels along the connection from the amplifier to the Wave's publisher. The publisher's node briefly brightens (a 0.2-second flash of increased halo intensity). If many amplifications arrive in quick succession, the effect compounds — the publisher's node appears to glow intensely as it receives recognition from the network.

### Echo Visualization

Echoes (re-broadcasts of a Wave beyond its original propagation range) are visualized as a secondary shockwave emanating from the echoing node. The secondary shockwave is thinner and fainter than the original publication shockwave, and its color blends the original publisher's color with the echoing node's color. This makes Echo propagation visually distinct from original publication — users can see Waves rippling outward through chains of Echoes.

---

## Specter Layer Visualization

### Specter Node Appearance

Specter nodes are rendered with the same anatomy as Surface Layer nodes (core, sigil, ring, halo, marks, gifts) but with a distinctive visual treatment that marks them as anonymous. Specter cores have a subtle translucency effect — the background of the Pulse Map is faintly visible through the node, as if the Specter exists in a layer slightly offset from the surface. Specter nodes also have a continuous, very slow particle emission effect: tiny luminous particles drift upward and outward from the node at irregular intervals, giving the impression of a node that is gently dissolving or being perpetually recreated.

The particle emission rate scales with Specter Resonance. Low-Resonance Specters emit particles rarely (one every 2–3 seconds). High-Resonance Specters emit particles continuously, creating a visible aura of luminous motes. This makes high-Resonance Specters visually striking and immediately identifiable as powerful anonymous identities.

### Shroud Relay Visualization

When the user's traffic is being routed through Shroud Nodes (in Hybrid+ or Fortress mode), the Pulse Map displays a subtle routing indicator: a faint, animated path from the user's node through 3 relay icons (represented as small shield glyphs) to the network. The path pulses gently to indicate active routing. This visualization reassures the user that their traffic is being anonymized without revealing the actual relay nodes' identities (the shield glyphs are generic, not specific to any node).

### Mini-Game Visualization

Active mini-games are rendered as distinctive visual events on the Pulse Map, each with a unique visual signature.

**Cipher Puzzles** appear as a pulsing glyph — a rotating cryptographic symbol at the centroid of the participants' nodes. The glyph brightens each time a participant submits a solution attempt. Mosaic Puzzles show multiple smaller glyphs converging toward a central point as contributors add pieces.

**Specter Hunts** manifest as scattered glowing fragments across the relevant Pulse Map region. Claimed fragments fade from bright gold to dim embers. A faint trail connects claimed fragments to the claiming Specter's recent path, creating a visible treasure-hunt narrative.

**Territory Drift** is rendered as translucent watermarks — the controlling Specter's sigil faintly visible in the territory's background. Contested territories show overlapping, flickering watermarks from competing Specters. Territory boundaries are drawn as soft, shifting gradient lines.

**Oracle Pools** appear as a swirling vortex icon at the centroid of participants. The vortex color shifts as predictions accumulate, and resolves into a bright flash when the outcome is determined.

**Sigil Forge** events display a small anvil-and-flame icon that pulses with each new entry submission. When voting begins, the entries orbit the icon as small thumbnails visible at meso zoom.

**Shadow Play** sessions appear as a dark, shimmering dome (similar to Masked Events but with a distinctive purple-black palette and subtle lightning effects), indicating an active social deduction game.

At macro zoom, all active mini-games are collapsed into small animated type-specific icons, providing an at-a-glance view of where game activity is happening across the network.

### Masked Event Visualization

Active Masked Events are rendered as a translucent dome or bubble on the Pulse Map, positioned at the centroid of the event creator's approximate location in the topology. The dome is a semi-transparent circle with a shimmering, iridescent surface effect. The dome's radius is proportional to the number of participants (more participants produce a larger dome).

Inside the dome, Masked participants are represented as small, identical, featureless dots — all the same size, color (a neutral white-silver), and shape. The dots move gently in a random Brownian motion pattern inside the dome. No labels, sigils, or distinguishing features are shown. The visual reinforces the mechanic's promise: inside a Masked Event, everyone is equal and unrecognizable.

Outside the dome, Specters who are not participating can observe the dome's size and activity level (the dome pulses with increased brightness when Masked Waves are being published inside) but cannot see individual dots or read content. The dome is a visible social event — a gathering happening in plain sight but impenetrable to outsiders.

### Council Visualization

Phantom Councils are represented on the Pulse Map as a constellation pattern connecting the Council's member nodes. The constellation lines are rendered as thin, glowing threads in a unique color derived from the Council's ID hash. The threads are visible to all Anonymous Layer observers, making the Council's membership and structure public knowledge. The threads pulse gently when Council Waves are being exchanged on the private topic, indicating that the Council is active without revealing the content of its deliberations.

The constellation pattern is rendered only at meso zoom and closer. At macro zoom, the Council is represented by a single glyph (a small crown or star icon) positioned at the centroid of the member nodes.

### Whisper Chain Visualization

Whisper Chain activity is deliberately not visualized on the Pulse Map. Because Whisper Chains are private, point-to-point communications, rendering their paths or endpoints would compromise privacy. The only visual indication of Whisper Chain activity is a subtle incoming-message indicator on the recipient's node: a brief, very small pulse effect (distinct from the Wave publication shockwave) that is visible only to the recipient at micro zoom. Other observers cannot distinguish a Whisper Chain delivery from normal node activity.

---

## Background and Atmosphere

### Map Background

The Pulse Map background is a dark, low-saturation field. The default background is a very dark blue-gray gradient with subtle procedural noise texture, evoking a night sky or deep ocean. The background is deliberately dark to provide contrast for the bright node and connection elements and to create an atmospheric, immersive quality.

The background is not static. A very slow, large-scale procedural animation shifts the noise pattern over time (one full cycle per 10 minutes), creating a sense of the space itself being alive. The animation is subtle enough to be subconscious — users do not notice it directly but feel a sense of organic vitality in the environment.

### Ambient Particle Field

A sparse ambient particle field fills the background. Tiny, dim particles drift slowly across the viewport in random directions at very low speeds. The particle density is approximately 1 particle per 10,000 square pixels. The particles are not associated with any node or mechanic — they are purely atmospheric, contributing to the sense of the Pulse Map as a living, breathing space rather than a static diagram.

### Activity Heat Map

An optional overlay renders a heat map of recent activity across the network. The heat map colors regions of the background based on the density of Wave publications in the trailing 60 minutes, using a blue-to-red gradient (blue for low activity, red for high activity). The heat map is rendered as a low-resolution, heavily blurred layer behind the nodes, creating a soft glow effect that highlights active regions without cluttering the visualization.

The heat map is toggled via a button in the viewport controls. It is off by default to maintain the clean, atmospheric aesthetic.

---

## Performance Optimization

### Rendering Pipeline

The Pulse Map is rendered using a GPU-accelerated 2D rendering pipeline. On desktop platforms, the renderer uses WebGL 2.0 (for the web client) or native GPU APIs (for native clients). On mobile platforms, the renderer uses WebGL 2.0 or Metal/Vulkan through a cross-platform abstraction layer.

The rendering pipeline processes elements in a fixed order: background, heat map overlay, connection lines, node cores and rings, sigils, halos and particle effects, Phantom Gift effects, Mark sigils, content cards, and UI overlay. Each layer is rendered to a separate framebuffer and composited with appropriate blend modes, allowing independent update rates for different layers (e.g., the background animates at 10 fps while nodes and connections animate at 60 fps).

### Level of Detail Culling

Nodes and connections outside the current viewport are not rendered. Nodes beyond a distance threshold from the viewport center are replaced by cluster representatives (a single dot representing multiple nodes) for both layout computation and rendering. Content cards, sigils, and labels are rendered only within the current detail level's threshold. These culling strategies ensure that rendering cost scales with viewport size, not network size.

### Batched Rendering

Nodes of the same type (same detail level, same layer) are rendered in batched draw calls. Connection lines are rendered in a single batched draw call using instanced rendering. Particle effects (Specter node emissions, ambient particles, and shockwave particles) are rendered using a GPU particle system with a single draw call per particle type. These batching strategies minimize draw call overhead and maintain 60 fps on target hardware.

### Target Hardware

The Pulse Map is designed to maintain 60 fps on the following minimum hardware profiles: desktop computers with integrated GPUs from 2020 onward, smartphones from 2021 onward, and tablets from 2020 onward. Networks of up to 10,000 visible nodes (within the viewport at meso zoom) should render at 60 fps on this hardware. Networks of up to 100,000 total nodes (with most culled by the viewport) should maintain responsive pan and zoom at 30+ fps.

### Data Update Throttling

The Pulse Map's visual state is updated from network data at a throttled rate to prevent rendering from being overwhelmed by high-frequency gossip events. Node position updates from the layout simulation are applied at 30 Hz. Node state updates (activity, halo, marks, gifts) are applied at 10 Hz. Connection state updates are applied at 5 Hz. Wave content card updates are applied at 2 Hz. These throttle rates are configurable in application settings for users with high-performance or low-performance hardware.

---

## Interaction Model

### Gesture Vocabulary

The Pulse Map supports the following gesture vocabulary, consistent across desktop and mobile platforms.

**Pan.** Click-drag (desktop) or one-finger drag (mobile) moves the viewport across the map. Momentum scrolling is supported: a fast drag followed by release results in the viewport continuing to move with decelerating velocity.

**Zoom.** Scroll wheel (desktop) or two-finger pinch (mobile) adjusts the zoom level. Zoom is centered on the cursor position (desktop) or the pinch midpoint (mobile). Zoom transitions are animated with ease-out timing over 0.2 seconds.

**Select.** Click (desktop) or tap (mobile) on a node selects it and opens the Node Detail Panel. Click or tap on empty space deselects the current node and closes the panel.

**Quick Action.** Right-click (desktop) or long-press (mobile) on a node opens the radial quick-action menu.

**Center.** Double-click (desktop) or double-tap (mobile) on a node centers the viewport on that node and zooms to micro view.

**Find Self.** A dedicated button in the viewport controls centers the viewport on the user's own node and resets to ego-centric view.

### Search and Navigation

A search bar at the top of the viewport allows users to search for nodes by display name, public key fingerprint, or Specter pseudonym. Search results are displayed as a dropdown list. Selecting a result pans and zooms the viewport to center on the matching node.

A bookmarks feature allows users to save nodes for quick navigation. Bookmarked nodes are displayed in a collapsible sidebar list. Selecting a bookmark pans and zooms to the bookmarked node.

### Filters

Users can apply visual filters to the Pulse Map to focus on specific aspects of the network. Available filters include activity filters (show only nodes active in the last hour, day, or week), connection filters (show only the user's direct connections, 2-hop neighborhood, or full network), layer filters (show only Surface Layer, only Anonymous Layer, or both), and mechanic filters (highlight nodes with active Marks, active Phantom Gifts, or active mini-game participation).

Filters are applied as visual adjustments — filtered-out nodes are dimmed to very low opacity (5%) rather than hidden entirely, maintaining spatial context while directing attention to the filtered subset. Active filters are displayed as pill-shaped indicators below the search bar.

### Accessibility

The Pulse Map includes accessibility accommodations for users with visual impairments or motor difficulties. A high-contrast mode replaces the atmospheric dark background with a true black background and increases node and connection brightness to maximum contrast. A colorblind mode replaces the default color encoding with a palette optimized for the three most common forms of color vision deficiency (deuteranopia, protanopia, and tritanopia), selected via a setting.

A screen-reader mode replaces the visual map with a navigable list interface. Nodes are listed in order of proximity to the user (ego-centric) or by activity level. Each list item includes the node's label, connection count, activity summary, and available actions. The list interface provides equivalent functionality to the visual map for users who cannot use the spatial visualization. Keyboard navigation is fully supported in both the visual map (arrow keys to pan, plus/minus to zoom, Tab to cycle through nodes, Enter to select) and the list interface.

---

## Notification Integration

### Pulse Notifications

Significant events are surfaced as Pulse Notifications — brief, non-intrusive visual indicators on the Pulse Map. A Pulse Notification appears as a small animated icon at the edge of the viewport, pointing in the direction of the event's location on the map. Tapping or clicking the notification pans the viewport to the event's location.

Events that trigger Pulse Notifications include: a new Wave from a connected node, a new amplification on the user's Wave, a new Phantom Gift received, a new Specter Mark placed on the user's node, a new mini-game event from a connected Specter, a new Masked Event announced by a connected Specter, and a new Whisper Chain message received.

### Notification Priority

Notifications are prioritized to prevent overload. High-priority notifications (Whisper Chain messages, direct replies to the user's Waves, Phantom Gifts received) are displayed immediately with a brief attention pulse (a flash at the viewport edge). Medium-priority notifications (new Waves from connections, amplifications, new events) are batched and displayed in a queue, one every 5 seconds. Low-priority notifications (activity from distant nodes, network topology changes) are displayed only in the notification log, not as active viewport indicators.

Users can configure notification priority levels and toggle individual notification types in settings.

---

## State Persistence

### Viewport State

The Pulse Map persists the user's viewport state across sessions. When the application is closed and reopened, the viewport restores to the same position and zoom level. If the user was in ego-centric view, the viewport re-centers on the user's node (which may have moved due to topology changes while offline). If the user was in network-centric view, the viewport restores to the same approximate region (which may have shifted due to layout recomputation).

### Layout Cache

The force-directed layout state (node positions and velocities) is cached locally. When the application starts, the layout is initialized from the cache rather than recomputed from scratch. The layout simulation then runs for a brief settling period (1–2 seconds) to adjust for topology changes that occurred while offline. This produces a smooth startup experience — the map appears immediately in approximately the correct configuration and gently adjusts to the current state.

### Filter and Preference State

Active filters, bookmarks, layer blend settings, zoom preferences, and accessibility settings are persisted in local application state. These preferences are not synced across devices — they are per-device settings.

---

## Visual Language Summary

The Pulse Map's visual language is designed to be learnable through observation rather than requiring explicit instruction. The key visual mappings are as follows.

**Node size** encodes social significance: larger nodes are more connected, more active, or higher in Resonance.

**Node color** encodes identity: each node has a unique color derived from its public key, with Surface Layer nodes in warm tones and Anonymous Layer nodes in cool tones.

**Node opacity** encodes layer: Surface Layer nodes are fully opaque, Anonymous Layer nodes are slightly translucent.

**Halo glow** encodes recent activity: glowing nodes have published recently, dim nodes have been quiet.

**Connection style** encodes connection age: dashed lines are new, solid lines are established, glowing solid lines are old.

**Connection pulses** encode information flow: bright dots traveling along connections show Waves propagating in real time.

**Particle emission** encodes Specter Resonance: more particles mean higher Resonance.

**Marks** encode anonymous attention: more Mark sigils around a node mean more anonymous identities have noticed it.

**Phantom Gift effects** encode anonymous generosity: visible cosmetic effects indicate that the node has been gifted by an anonymous benefactor.

**Mini-game glyphs** encode competition: animated type-specific icons between or around nodes indicate active mini-game events.

**Event domes** encode gathering: translucent bubbles on the map indicate anonymous social events in progress.

**Council constellations** encode coordination: glowing thread patterns connecting nodes indicate a persistent anonymous group.

Together, these visual mappings create a rich, legible visual language that communicates the network's social state without requiring users to read text or consult documentation. The Pulse Map is designed to be understood intuitively — users learn what the visual elements mean by observing how they correlate with social activity over time.