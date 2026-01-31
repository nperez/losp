# Lonely Robot - Game Requirements

A post-apocalyptic robot survival game using losp for LLM-driven narrative and probability.

## Concept

You are a robot that has just awakened in a post-apocalyptic wasteland. You have fragmented memories and uncertain purpose. Your goal is simple: survive as many days as possible while navigating the dangers and mysteries of the wasteland.

All interactions are through natural language. The LLM determines outcomes based on probability embedded in the narrative - you never see explicit odds, just the story of what happens.

## Robot State

### Battery
Fuzzy categories representing power reserves:
- **fully charged** - Optimal power, all systems functional
- **moderate** - Comfortable reserves, normal operation
- **low** - Concerning levels, should find power soon
- **critical** - Emergency reserves, immediate danger
- **depleted** - Game over

### Physical Condition
Fuzzy categories representing structural integrity:
- **pristine** - Factory fresh, no damage
- **functional** - Minor wear, fully operational
- **damaged** - Significant issues, reduced capability
- **critical** - Major damage, barely operational
- **destroyed** - Game over

### Game Over
Occurs when either:
- Battery reaches "depleted"
- Condition reaches "destroyed"

## Inventory System

- Simple item names stored as newline-separated list
- Items consumed based on LLM judgment from action context
- No explicit durability or quantity tracking
- Used through natural language (e.g., "use the repair kit", "eat the power cell")

Example inventory:
```
rusty screwdriver
cracked solar panel
mysterious data chip
half-empty coolant canister
length of copper wire
```

## World Model

### Abstract Wandering
The robot is always "somewhere in the wasteland" - no discrete location grid. The environment is described narratively based on context.

### Landmarks
Notable locations at varying distances:
- **near** - Can reach with minimal effort (< 1 hour)
- **close** - Reachable with some travel (1-2 hours)
- **distant** - Significant journey (half a day)
- **far** - Major expedition (full day or more)

Landmarks can:
- Be approached or avoided
- Have unique encounters
- Change distance as robot moves
- Be discovered during exploration

Example landmarks:
```
Collapsed Radio Tower - near
Rusted Convoy Graveyard - close
Sunken Shopping Mall - distant
Flickering Lighthouse - far
```

## Encounter System

### Encounter Types
- **Combat** - Hostile entities (raiders, rogue robots, mutant creatures)
- **Scavenging** - Opportunities to find supplies
- **Environmental** - Weather, terrain hazards, natural phenomena
- **Social** - Other survivors, friendly robots, traders
- **Weird** - Aliens, time loops, dimensional anomalies, inexplicable events

### Encounter Generation
- Random check each turn based on context
- Time of day affects probability (night more dangerous)
- Robot condition affects encounters (damaged attracts scavengers)
- Recent events influence what happens next
- Landmarks may trigger specific encounters

### Resolution
- Single-shot resolution per turn
- Probability embedded in narrative, never shown explicitly
- Outcomes affect battery, condition, inventory, landmarks
- LLM determines hours passed

## Probability System

Probability is embedded in narrative rather than shown:

**Instead of:** "You have a 30% chance to succeed"

**Show:** "The rusted lock looks formidable, and your manipulators aren't at their best..."

The LLM considers:
- Current battery level (low power = worse outcomes)
- Physical condition (damaged = reduced capability)
- Relevant inventory items (tools help)
- Environmental context (time of day, weather)
- Recent history (accumulated exhaustion)

## Time System

### Hours per Action
The LLM estimates hours passed for each action:
- Quick actions: 1-2 hours
- Standard actions: 2-4 hours
- Complex actions: 4-6 hours
- Major undertakings: 6-8 hours

### Day Advancement
- Each day is 24 hours
- Time accumulates across turns
- New day triggers when 24+ hours pass
- Day number is the primary survival metric

### Starting Time
- Day 1 begins at awakening
- Starting time of day varies (morning/afternoon/evening)

## Starting State

Random generation by LLM:
- **Battery**: Random from {fully charged, moderate, low} - never critical
- **Condition**: Random from {pristine, functional, damaged} - never critical
- **Inventory**: 3-5 random items (survival gear, junk, maybe something useful)
- **Situation**: Robot just awakened, unique circumstances each time
- **Landmarks**: 2-3 initial landmarks at varying distances

## History Management

### Recent History
Last 3 turns kept in full detail:
```
Turn 5: Searched the abandoned vehicle → Found rusty screwdriver, disturbed nest of rad-rats
Turn 6: Fought off the rad-rats → Minor scratches, battery drain from combat
Turn 7: Continued toward radio tower → Reached the base, found locked entrance
```

### Summary History
Older turns compressed by LLM:
```
Days 1-2: Awakened in crater, found initial supplies, survived first night in collapsed structure. Encountered and fled from raider patrol.
```

### Compaction
- When recent history exceeds 3 turns
- LLM summarizes oldest turns into summary
- Preserves important events and consequences
- Maintains narrative continuity

## Input Handling

### All Natural Language
No commands, no menus. Player types what they want:
- "Look around for anything useful"
- "Try to pry open the door with the screwdriver"
- "Hide and wait for the patrol to pass"
- "What's my battery level?"

### Input Classification
LLM classifies each input:

**QUESTION** - Asking about status or situation
- "How am I doing?"
- "What do I have?"
- "What's nearby?"
- Does NOT advance time
- Returns information, then prompts for next action

**ACTION** - Attempting to do something
- "Search the wreckage"
- "Approach the tower carefully"
- "Try to repair myself using the toolkit"
- Advances time
- Resolves with probability
- Updates state

## Persistence

### Auto-Save
After each turn:
- All WL_ state variables persisted
- History saved
- Can resume from any point

### Resume Flow
On startup:
- Check for existing WL_Day
- If exists: load state, generate recap, continue
- If empty: new game flow

### Seamless Continue
Resume experience:
```
=== Resuming Wasteland ===

You last played as a damaged maintenance robot on Day 3.
Battery: low | Condition: damaged
You were near the Collapsed Radio Tower, having just...
[LLM-generated recap of situation]

What do you do?
>
```

## Game Over

When battery=depleted or condition=destroyed:

1. Final narrative describing the end
2. Summary of the run:
   - Days survived
   - Key moments (major discoveries, close calls, strange encounters)
   - Final status
3. Clean exit (no auto-restart)

Example:
```
=== SHUTDOWN ===

Your power reserves finally give out. The last thing you
register is the setting sun casting long shadows across
the wasteland. Your optical sensors dim, then go dark.

DAYS SURVIVED: 7

KEY MOMENTS:
- Awakened in the crater of an old impact site
- Discovered the underground bunker on Day 2
- Survived the electromagnetic storm on Day 4
- Met the wandering merchant, traded for solar cells
- Final stand against the scavenger pack...

Perhaps another robot will find your chassis someday.
```

## State Variables

All prefixed with `WL_`:
```
WL_Day              - Current day number
WL_TimeOfDay        - Hours into current day (0-23)
WL_Battery          - Battery category
WL_Condition        - Physical condition category
WL_Inventory        - Newline-separated item list
WL_Landmarks        - Landmarks with distances
WL_History_Recent   - Last 3 turns in detail
WL_History_Summary  - Compressed older history
WL_TurnCount        - Total turns played
```

## Design Principles

1. **Narrative over mechanics** - Everything is described, nothing is displayed as numbers
2. **Probability through prose** - Likelihood hints embedded naturally
3. **Emergent storytelling** - Each run is unique based on LLM responses
4. **Simple state, rich outcomes** - Few variables, infinite possibilities
5. **Graceful degradation** - Game remains playable as condition worsens
6. **Mystery preserved** - Weird encounters add spice without explanation
