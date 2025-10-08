# Refactoring: UI Components and Business Logic Separation

## Overview

This document describes the refactoring effort to separate UI components from business logic in the buzz application, making the codebase more maintainable and easier to extend.

## Changes Made

### File Structure

The refactoring reorganized code from a single large `main.go` file (618 lines) into multiple focused files:

1. **main.go** (217 lines) - Core Bubble Tea orchestration
   - `Init()` - Initialize the application
   - `Update()` - Handle state transitions
   - `updateApp()` - Process application messages
   - `View()` - Render the UI
   - `viewApp()` - Render the application view
   - `main()` - Entry point
   - `handleNextCommand()` - CLI command handler

2. **model.go** (96 lines) - State definitions and initialization
   - `appModel` struct - Application state
   - `model` struct - Top-level model
   - `initialAppModel()` - Create initial app state
   - `initialModel()` - Create initial model
   - `filterGoals()` - Filter goals by search query
   - `getDisplayGoals()` - Get goals to display

3. **handlers.go** (408 lines) - Input handling logic
   - `handleKeyPress()` - Main keyboard input router
   - Individual handlers for each key action:
     - `handleEscapeKey()` - Exit/close actions
     - `handleAddDatapoint()` - Enter datapoint input mode
     - `handleTabKey()` - Tab navigation
     - `handleBackspace()` - Text deletion
     - `handleEnterKey()` - Submit forms or open modals
     - `handleNavigation*()` - Arrow key navigation
     - `handleScroll*()` - Page scrolling
     - `handleRefresh()` - Manual refresh
     - `handleToggleRefresh()` - Auto-refresh toggle
     - `handleEnterSearch()` - Enter search mode

4. **grid.go** (245 lines) - UI rendering (unchanged)
   - Grid, modal, and footer rendering functions

5. **Other files** - Unchanged
   - `auth.go`, `beeminder.go`, `config.go`, `messages.go`, `styles.go`, `utils.go`

## Benefits

### 1. Improved Code Organization
- **Single Responsibility**: Each file has a clear, focused purpose
- **Easier Navigation**: Developers can quickly find relevant code
- **Reduced Cognitive Load**: Smaller files are easier to understand

### 2. Better Maintainability
- **Isolated Changes**: Input handling changes don't affect rendering
- **Clear Dependencies**: State, handlers, and views are separated
- **Easier Testing**: Smaller, focused functions are easier to test

### 3. Enhanced Extensibility
- **New Features**: Adding new keyboard shortcuts is straightforward
- **New States**: Model changes are isolated in model.go
- **New UI Components**: Rendering is already organized in grid.go

## Design Decisions

### Why Not Separate Packages?

We chose to keep everything in the `main` package rather than creating separate packages (e.g., `state/`, `handlers/`, `ui/`) because:

1. **Minimal Changes**: Reduces the scope of refactoring
2. **Simpler Imports**: No cross-package imports needed
3. **Type Access**: All types remain accessible without export requirements
4. **Incremental Approach**: Can be further refactored into packages later if needed

### Handler Pattern

Each keyboard input has a dedicated handler function that:
- Takes the current model as input
- Returns the updated model and optional command
- Has a clear, descriptive name (e.g., `handleNavigationLeft`)
- Contains all logic for that specific input

This pattern makes it easy to:
- Add new keyboard shortcuts
- Modify existing behavior
- Test individual handlers
- Understand what each key does

### State Management

The `model.go` file centralizes all state-related code:
- Model struct definitions
- State initialization
- State query methods (e.g., `getDisplayGoals()`)

This makes it clear where to look for state-related changes.

## Migration Notes

### For Developers

When working on this codebase:

1. **Adding New Keyboard Shortcuts**
   - Add handler function in `handlers.go`
   - Call handler from `handleKeyPress()` switch statement

2. **Modifying State**
   - Update struct definition in `model.go`
   - Update initialization in `initialAppModel()`

3. **Changing UI Rendering**
   - Modify functions in `grid.go`
   - Keep styles in `styles.go`

4. **Adding New Features**
   - State in `model.go`
   - Input handling in `handlers.go`
   - Rendering in `grid.go`
   - Business logic in appropriate files (e.g., `beeminder.go`)

## Future Improvements

Potential next steps for further refactoring:

1. **Extract to Packages**: Move to `state/`, `handlers/`, `ui/` packages
2. **Unit Tests**: Add tests for handlers and state management
3. **Component Interfaces**: Define interfaces for testability
4. **View Models**: Separate display logic from state
5. **Command Pattern**: Centralize command creation

## Metrics

- **Lines Reduced in main.go**: 401 lines (65% reduction)
- **New Files Created**: 2 (model.go, handlers.go)
- **Build Status**: ✅ Successful
- **Functionality**: ✅ Preserved (no regressions)

## Conclusion

This refactoring successfully separated UI components from business logic while maintaining all existing functionality. The codebase is now more maintainable, organized, and ready for future enhancements.
