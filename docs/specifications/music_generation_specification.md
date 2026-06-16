# Technical Specification: Antigravity Jazz Lounge Generation Engine

This document outlines the architecture, algorithmic designs, and state models of the history-aware, emotion-driven 1940s noir lounge jazz ensemble generation system.

---

## 1. Synthesis & Audio Pipeline Architecture

The system generates multi-instrument audio in real-time, utilizing physical synthesizer voices mixed with ambient field recordings to establish the 1940s lounge environment.

### 1.1 Ambient Atmosphere Engine (`ambient.go`)
An ambient soundscape layer runs continuously in the background, adding texture and low-frequency depth:
- **Rain & Thunder**: A rolling environmental track simulating storm elements.
- **Bar Chatter**: Soft, distant crowd voices, clinking glasses, and hushed speech.
- **Vinyl Crackle**: A constant, low-amplitude crackle simulating vintage turntable playback.
- **Glass Clinks**: Sporadic, high-frequency, randomized transient events.
The volume of each ambient element is modulated dynamically by the `JazzLoungeEngine` based on active emotional states (e.g., higher bar chatter in crowded/tense atmospheres, louder rain during intimate/still moments).

### 1.2 Synth Voice Architecture (`voice.go` / `streamers.go`)
Individual instruments (Piano, Synth Melody/Soloist, Bass) are synthesized using a custom voice-additive synthesis engine:
- **Tones**: Formed using sine, triangle, and square waveforms, shaped by specific envelope parameters.
- **Piano Voicing**: Constructed of multiple concurrent voice nodes using exponential decay modeling to emulate hammer strikes and resonance.
- **Soloist**: Utilizes slightly filtered triangle waves for a warm brass/woodwind timbre.
- **Bass**: Employs warm, low-passed sine waves to emulate a double bass.

---

## 2. The Emotional Forces Engine

Instead of repeating a static *Exposition -> Development -> Climax -> Resolution* pattern, the performance is governed by a set of competing psychological forces that rise and fall organically.

### 2.1 Emotional Dimensions (`ensemble.go`)
Six core psychological dimensions are tracked continuously:
1. **Intimacy**: Rises during periods of collective silence or transition; decays under heavy tension or soloist loudness.
2. **Melancholy**: Driven by minor keys and persistent session moods ("Melancholic", "Weary").
3. **Tension**: Increases on altered dominant chords (`7alt`) and high soloist energy; resolves slowly on major tonics (`maj7`, `maj9`).
4. **Warmth**: Rises during transitions and low-tension, intimate sections.
5. **Momentum**: Rises when a soloist is active; decays during silence or comping-only sections.
6. **Mystery**: Elevates when an "AmbiguousChromatic" session style or "Mysterious" mood is active.

### 2.2 System Adaptation
The emotional values dynamically adjust:
- **Perceived Tempo**: Calculated as:
  $$\text{PerceivedTempo} = 0.25 + 0.45 \times \text{Tension} + 0.2 \times \text{Momentum} - 0.15 \times (\text{if Intimacy} > 0.6)$$
  This scales the note occupancy, brush density, and comping patterns without modifying the physical tempo (BPM).
- **Macro Energy**: Tied directly to the perceived tempo, scaling the velocities and volume mixes of the active instruments.

---

## 3. Register Architecture

The vertical alignment of the band is managed dynamically using two parameters:
1. **Register Center ($C_r$)**: The pitch center of gravity.
2. **Register Width ($W_r$)**: The vertical spread.

### 3.1 Mapping Bounds
At any tick, the soloist boundary bounds $[Low, High]$ are scaled as follows:
- When $W_r < 0.4$, the boundaries contract inwards by up to 5 semitones (creating a narrow, concentrated midrange).
- When $W_r > 0.8$, the boundaries expand outwards (reaching the limits of the instrument range for dramatic runs).

$C_r$ migrates slowly over long tick periods (following a low-frequency oscillator formula) and sinks lower during "Introspective" or "Weary" moods, making the instruments gravitate towards lower registers.

---

## 4. Session Harmonic Personality

At the start of every performance, a persistent `HarmonicTaste` is generated to define the band's color habits for the evening.

| Taste Profile | Style | Chord Substitution Chance | Extensions Preference | Chord Colors |
|---|---|---|---|---|
| **ConsonantClassic** | Traditional, stable | $15\%$ | $9\text{ths}$, $6\text{ths}$ | Tonic prolongations, major sonorities |
| **DarkModal** | Heavy pedal tones | $35\%$ | Minor $9\text{ths}$, $11\text{ths}$ | Dark Dorian and Phrygian inflections |
| **AmbiguousChromatic**| Passing harmony | $60\%$ | Diminished, Half-diminished | Chromatic voice leading, tritone subs |
| **BackdoorDominant**  | Deceptive cadences| $70\%$ | $7\text{alt}$, $13\text{b9}$ | Backdoor resolutions, unresolved dominants|

The taste determines the likelihood of applying chord substitutions in `engine.go` and shapes the comping voicings of the piano.

---

## 5. Motif Identity & Memory System

Instead of simple note duplication, the ensemble utilizes a contour-based memory architecture.

### 5.1 Motif Contour Representation
Every `ThematicMotif` stores:
- **Contour Profile**: An array of steps mapping intervals as $+1$ (ascending), $-1$ (descending), or $0$ (stay same).
- **Rhythmic Signatures**: An array of tick spacings.
- **Emotional Quality**: The mood tag active during the motif's creation.

### 5.2 Transformation & Recall
When a soloist recalls a motif from the inventory (60% chance):
- **Contour Preservation**: There is a 40% chance the instrument recreates the motif by taking the contour directions and walking current chord tones rather than playing the literal pitches.
- **Literal Recall**: Otherwise, the motif is transposed to the soloist's current register with minor random pitch variations to allow developmental variation.

---

## 6. Ensemble Obsession & Fixation

At random intervals, the band develops a temporary collective fixation on a specific musical idea, representing the social phenomenon of mutual discovery.

- **Obsession Strengths**: Starts at $0.7 - 1.0$ and decays by $0.003$ per tick.
- **Obsession Types**:
  1. **Interval**: Fixates on a specific musical interval (e.g., minor 6th). Soloists have a high probability of making melodic leaps matching this interval.
  2. **Rhythmic Gesture**: Fixates on a specific note-duration spacing. Soloists override standard swing/comping patterns to play notes separated by this duration.
  3. **Register Area**: Gravitates towards a specific MIDI center, driving bass and piano voicings to narrow in on that region.
