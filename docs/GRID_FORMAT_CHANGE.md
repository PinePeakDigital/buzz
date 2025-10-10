# Grid Goal Format Change

## Overview

Updated the goal grid cell format to use a consistent 16-character inner width with stakes displayed on the first line alongside the goal slug.

## Before (Old Format)

Each cell had a 16-character inner width:

```
┌──────────────────┐
│ the_slug         │
│ $5 | +2 in 3 days │
└──────────────────┘
```

Format:
- Line 1: Slug (truncated to 16 chars)
- Line 2: `$pledge | deltaValue in timeframe`

## After (New Format)

Each cell now has a 16-character inner width (20 total with borders and padding):

```
┌──────────────────┐
│ the_slug      $5 │
│ +2 in 3 days     │
└──────────────────┘
```

Format:
- Line 1: `slug` + expanding spaces + `$pledge`
- Line 2: `deltaValue in timeframe`

## Examples

### Various Scenarios

**Short slug with small pledge:**

```text
┌──────────────────┐
│ the_slug      $5 │
│ +2 in 3 days     │
└──────────────────┘
```

**Long slug (truncated with ellipsis):**

```text
┌──────────────────┐
│ a_very_lon... $5 │
│ +10 in 5 days    │
└──────────────────┘
```

**Large pledge amount:**

```text
┌──────────────────┐
│ exercise    $270 │
│ +1.5 in 12 hrs   │
└──────────────────┘
```

**Short timeframe:**

```text
┌──────────────────┐
│ my_goal      $10 │
│ 0 in today       │
└──────────────────┘
```

**Decimal delta value:**

```text
┌──────────────────┐
│ short         $5 │
│ +1.315464 in 5 d │
└──────────────────┘
```

## Implementation Details

### New Functions

- `formatGoalFirstLine(slug string, pledge float64) string`
  - Formats the first line with slug and stakes
  - Ensures exactly 16 characters
  - Truncates slug with ellipsis if needed
  - Adds expanding spaces between slug and pledge

- `formatGoalSecondLine(deltaValue string, timeframe string) string`
  - Formats the second line with delta and timeframe
  - Ensures exactly 16 characters
  - Truncates with ellipsis if combined string is too long

### Changes to Existing Functions

- `calculateColumns()`: Updated to use 20 chars per cell (16 content + 2 borders + 2 padding)
- `RenderGrid()`: Updated to use new formatting functions

### Tests

Added comprehensive tests for both new formatting functions covering:
- Short and long slugs
- Various pledge amounts
- Truncation scenarios
- Edge cases (empty strings, exact lengths)

## Benefits

1. **Consistency**: All cells have the same width
2. **Cleaner Display**: Stakes integrated into first line looks cleaner
3. **Compact Layout**: Narrower cells (16 vs original 18 chars) provide a more compact display
4. **Better Truncation**: Clear ellipsis handling for both lines
