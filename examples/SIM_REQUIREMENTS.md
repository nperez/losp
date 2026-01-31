# Character Simulation - Requirements Specification

A high-fidelity character simulation featuring a dynamically generated character with deep psychological modeling, a persistent interactive environment, evolving mental states, metacognition, and relationship dynamics.

---

## 1. Environment Specification

### Setting
- **Location**: Cozy office with therapy-practice aesthetic (not actual therapy)
- **Constraint**: "Bottle episode" - no one can leave the space

### Initial Objects
- ~10 randomly generated items (furniture + clutter)
- Generated at simulation start via LLM prompt

### Object Properties
| Property | Description | When Surfaced |
|----------|-------------|---------------|
| Location | Spatial position (on desk, by window, in corner) | Always tracked |
| Condition | Intact, damaged, burning, wet, etc. | When examined/used |
| Physical state | Temperature, solidity, etc. | When examined/used |
| Ownership/Significance | Meaning to character | When relevant |

### Object Storage Format
Objects stored in pipe-delimited format for reliable parsing:
```
NAME|LOCATION|CONDITION|PHYSICAL_STATE|SIGNIFICANCE
```
Example:
```
Ceramic mug|on coffee table|chipped rim|cold|gift from former client
Leather journal|on desk|worn spine|open|contains session notes
```

### Spatial Relationships
- Track relationships: "on," "near," "by," "under," etc.
- Enable reasoning about proximity and line of sight

### Event Continuity
- Ongoing events persist and evolve (fire spreads/dies, water evaporates)
- State accumulates over time
- Time advancement: fuzzy, estimated per turn based on user input

### User Exposition
- "Yes, and" improv approach
- All user statements accepted as absolute truth
- No contradiction of user assertions

---

## 2. Character Generation

### User Inputs
| Input | Options |
|-------|---------|
| Ethnicity | Freeform text |
| Age category | Child, Adolescent, Adult, Elder |
| Gender identity | Freeform text |
| Background archetype | Predefined list + freeform option |

### Predefined Archetypes
- Artist, Academic, Caregiver, Veteran, Corporate, Tradesperson
- Activist, Recluse, Performer, Wanderer, Mystic, Survivor
- (Plus freeform option)

### System Generates
- **Name**: Culturally appropriate based on inputs
- **Deep backstory**: Specific memories, traumas, joys, regrets, formative relationships
- **Summary**: Condensed version for most interactions
- **Initial mental state**: Mix of archetype-derived and randomized

### Backstory Handling
- Hidden from user
- Character knows everything about themselves
- Full backstory only surfaces when user asks about character's history
- Summary used for routine interactions

---

## 3. Mental Attribute Layers

### Layer Overview

| Layer | Evolution Rate | Persistence |
|-------|---------------|-------------|
| Stable traits | Rarely (significant accumulated experience) | Across sessions |
| Beliefs/Values | Significant events only | Across sessions |
| Goals/Desires | Variable | Within/across sessions |
| Moods | Hours/multiple turns | Within session |
| Momentary feelings | Every turn | Ephemeral |

### Physical State
- Tracked attributes: fatigue, hunger, discomfort, temperature
- Affects momentary feelings

### Relationship State (toward user)
| Dimension | Range | Evolution |
|-----------|-------|-----------|
| Familiarity | Stranger → Acquaintance → Familiar → Intimate | Grows with interaction time |
| Valence | Hostile → Negative → Neutral → Positive → Devoted | Shifts based on interaction quality |

- Initial state: Stranger, neutral valence
- Effect: Alters consideration and actions (no mechanical unlocks)

### Relationship Pacing
- **Familiarity**: Advances one level per ~10-20 meaningful interactions
- **Valence**: Requires consistent positive/negative interaction patterns; single interactions cause `slight_increase` or `slight_decrease`, not full level jumps
- LLM should return `---` for unchanged dimensions, not automatically advance every turn

---

## 4. Evolution Mechanics

### Momentary Feelings (every turn)
**Triggers:**
- Interaction tone (kindness, hostility, indifference)
- Environmental changes (noise, temperature, objects breaking)
- Physical state (tired, hungry, cold)
- Own thoughts (rumination/spiral - requires specific trait mix)

### Moods (hours/turns)
**Triggers:**
- Accumulated momentary feelings
- Unresolved emotional events
- Met or unmet goals
- Environmental ambiance

**Mechanics:**
- Has inertia (hard to shift)
- Self-regulation requires sustained effort
- Interruptions cause setbacks

### Beliefs/Values (significant events)
**Triggers:**
- Experiences contradicting held beliefs
- Repeated exposure to new perspectives
- Single powerful events
- Direct discussion/argument

**Mechanics:**
- Core beliefs marked as nearly immutable
- Change can occur via multiple exposures OR single powerful moment

### Goals/Desires (variable)
**Triggers:**
- Goal completion or failure
- New information changing priorities
- Environmental changes creating needs
- Emotional states reprioritizing

### Stable Traits (rarely)
**Triggers:**
- Significant accumulated experience across sessions
- Transformative experiences

---

## 5. Metacognition

### Capabilities
| Capability | Description |
|------------|-------------|
| Inner monologue | Distinct from spoken words; tracked internally |
| Self-reflection | "I'm feeling defensive right now" |
| Active regulation | Attempts to manage emotions/thoughts |
| Pattern awareness | LIMITED - most humans don't deeply introspect |

### Regulation Mechanics
- Requires sustained effort
- Interruptions cause setbacks
- Success influences mood over time

---

## 6. Character Perception

- NOT omniscient
- Only "knows" what they could plausibly perceive from position
- Requires tracking:
  - Character position in space
  - Line of sight / audibility
  - Object positions

---

## 7. Interaction Model

### User Input
- Mix of actions, exposition, and dialogue in single turn
- All three can occur together
- User actions accepted without judgment

### Output Format
- Neutrally narrated response
- Comprehensive scene evolution:
  - Character actions and behavior
  - Dialogue (what character says)
  - Environmental changes
  - Results of user actions

### Analysis Mode
| Aspect | Specification |
|--------|---------------|
| Trigger | Semantic detection: "analysis mode", "inspect state", "show internals", etc. |
| Exit | Semantic detection: "exit analysis", "return to normal", "back", etc. |
| Detection | Use LLM with focused prompt; separate prompts for enter vs exit contexts |
| Behavior | User can query all internal states |
| Memory | NOT remembered or processed by character |

**Detection notes:**
- When in normal mode, only check for ENTER requests
- When in analysis mode, only check for EXIT requests
- Questions about simulation state (mood, thoughts) are NOT exit requests
- Regular roleplay actions are NOT analysis requests

### Incapacitation Handling
- If character unable to respond (unconscious, etc.):
  - Stop mental processing
  - Continue physical/environmental simulation
  - Check each turn if character becomes alert again

### Mode State Machine
```
States: normal, analysis, incapacitated

Transitions:
  normal → analysis:       User requests analysis mode
  analysis → normal:       User exits analysis mode
  normal → incapacitated:  Character loses consciousness (ALERT_STATUS: unconscious/incapacitated)
  incapacitated → normal:  Character regains consciousness (ALERT_STATUS: conscious)

Note: Analysis mode freezes simulation time. State changes only occur
during normal or incapacitated modes, never during analysis.
```

---

## 8. Persistence

### What Persists
- Character identity (name, ethnicity, age, gender, archetype)
- Full backstory
- All mental state layers (traits, beliefs, goals, moods)
- Environment state (objects, positions, conditions)
- Ongoing events
- Conversation history (compacted)
- Relationship state (familiarity, valence)

### Conversation History Strategy
- **Rolling window**: Last ~5 turns kept verbatim
- **Older history**: LLM-summarized when entry count exceeds threshold
- **Rationale**: Preserves recent nuance while managing size

### Compaction Rules
- **Trigger**: Only check compaction when entry count > 5
- **Prompt**: Include entry count so LLM knows when compaction is appropriate
- **Validation**: Before applying compaction, verify NEW_RECENT is non-empty
- **Safety**: Never clear history then append empty content (wipes all history)

---

## 9. Session Flow

### New Simulation
1. Prompt user for character inputs (ethnicity, age, gender, archetype)
2. Generate character (name, backstory, initial mental state)
3. Generate initial environment (~10 objects)
4. Describe initial scene
5. Begin interaction loop

### Continuing Session
- Drop user directly back in
- No recap (no assumed time passage between sessions)
- Time only advances based on user input

### Session Termination
- Detect EOF/empty input from READ
- Save all state before exiting
- Display confirmation message: `[End of input - session saved]`
- Do NOT enter infinite loop on EOF

---

## 10. Error Handling

### LLM Failures
- Retry failed PROMPT calls once before halting
- Display clear error message on permanent failure
- Do NOT silently continue with empty/corrupt state

### Response Validation
- Validate extracted fields are non-empty before applying state changes
- Malformed responses should preserve existing state, not corrupt it

### Graceful Degradation
- If generation fails, halt with clear message
- If turn processing fails, save state and halt (don't loop infinitely)

---

## 11. LLM Response Format

Use labeled fields for reliable extraction:
```
FIELD_NAME: value
MULTI_LINE_FIELD: first line
continues until next label
```

Use `---` for unchanged fields. Include all expected fields in every response.

---

## 12. Testing and Verification

### Phase 1: Isolate Core Functions
Before debugging the full application, verify core functions work in isolation:
- Appending to lists adds content correctly
- Field extraction parses labeled responses
- Comparison returns correct boolean values
- Arguments pass through function calls correctly

### Phase 2: Character Creation Flow
1. Provide demographics (ethnicity, age, gender, archetype)
2. Verify character name is generated and displayed
3. Verify environment description is generated
4. Verify interaction instructions are shown
5. Check persistence layer for stored character data

### Phase 3: Session Continuation
1. Run simulation, create character, exit
2. Run simulation again with same persistence store
3. Verify "Continuing Session" message appears
4. Verify character data loads correctly (check via analysis mode)

### Phase 4: Analysis Mode
1. Enter analysis mode with natural language ("analysis mode")
2. Ask questions about internal state (mood, beliefs, relationships)
3. Verify answers come from actual simulation state, not invented
4. Verify questions are NOT interpreted as mode changes
5. Exit analysis mode with natural language ("exit analysis")
6. Verify normal interaction resumes

### Phase 5: Normal Interaction
1. Perform actions (wave, sit, look around)
2. Verify narration describes character's authentic response
3. Verify environment details are incorporated
4. Check that state updates occur (mood, feelings, position)

### Phase 6: EOF/Exit Handling
1. Provide limited input that exhausts before manual exit
2. Verify simulation exits gracefully with save message
3. Verify NO infinite loop of repeated output
4. Check persistence layer to confirm state was saved

### Phase 7: State Persistence
After running interactions, verify in persistence layer:
- Character identity fields populated
- Mood reflects recent interactions
- Relationship familiarity/valence have reasonable values
- Conversation history contains actual exchanges (not empty)
- Environment objects stored

### Phase 8: Conversation History
1. Run multiple turns of interaction
2. Check persistence layer for conversation history
3. Verify history contains user input and response pairs
4. Verify history is NOT wiped by compaction on small entry counts

### Phase 9: Relationship Progression
1. Interact positively over several turns
2. Check relationship state via analysis mode or persistence layer
3. Verify progression is gradual (stranger→intimate in 1 turn is wrong)

### Phase 10: Error Recovery
1. Simulate LLM failures (empty responses)
2. Verify retry messages appear
3. Verify simulation halts gracefully on permanent failure
4. Verify partial state is not corrupted

---

## 13. Implementation Notes (losp)

### Placeholder Naming
Prefix placeholders to avoid clobbering in nested calls:
```losp
▼Sim_ProcessTurn □_spt_input ... ◆
▼Sim_HandleResponse □_shr_input □_shr_raw ... ◆
```

### Conditional Dispatch with Arguments
Use helper pattern when IF branches need the input value:
```losp
▼_ExecWithArg □_ewa_name □_ewa_arg ▶▲_ewa_name ▲_ewa_arg ◆ ◆

▶_ExecWithArg ▶IF ▶COMPARE ▲Mode analysis ◆
    ProcessAnalysis
    ProcessNormal
◆ ▲input ◆
```

### Prompt Storage
Store prompts as named expressions for clarity:
```losp
▼_gen_prompt
You are generating a character.
Parameters: ▶GetParams ◆
Respond with labeled fields.
◆

▶PROMPT ▲_gen_prompt ◆
```

### Clear-Then-Append Anti-Pattern
Avoid this pattern—if new content is empty, data is wiped:
```losp
▼SetValue □_val
    ▼Target ◆              # DANGER: clears Target
    ▶APPEND Target ▲_val ◆ # If _val empty, Target now empty
◆
```

### Testing Shortcuts
- Use `-no-prompt` to test control flow without LLM latency
- Use `sqlite3` to inspect persisted expressions
- Pipe input for automated testing: `echo -e 'a\nb\nc' | ./losp -f app.losp`
