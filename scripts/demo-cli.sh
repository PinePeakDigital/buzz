#!/usr/bin/env bash

# Buzz CLI Demo Recording Script
# Demonstrates buzz next, today, and add commands with asciinema

set -euo pipefail

# Get script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

# Configuration
BUZZ_BIN="$REPO_DIR/buzz"
DEMOS_DIR="$REPO_DIR/demos"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RECORDING_FILE="$DEMOS_DIR/demo-buzz-cli-$TIMESTAMP.cast"
TEMP_EXPECT_FILE="/tmp/buzz_demo_$$.expect"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v asciinema >/dev/null 2>&1; then
        log_error "asciinema not found. Install with: apt-get install asciinema"
        exit 1
    fi
    
    if ! command -v expect >/dev/null 2>&1; then
        log_error "expect not found. Install with: apt-get install expect"
        exit 1
    fi
    
    if [[ ! -x "$BUZZ_BIN" ]]; then
        log_info "Building buzz binary..."
        cd "$REPO_DIR"
        if go build -o "$BUZZ_BIN"; then
            log_info "Built buzz binary successfully ✓"
        else
            log_error "Failed to build buzz binary"
            exit 1
        fi
    fi
    
    log_info "All dependencies found ✓"
}

# Create expect script for automated demo
create_expect_script() {
    log_info "Creating expect script..."
    
    cat > "$TEMP_EXPECT_FILE" << EOF
#!/usr/bin/expect -f

set timeout 10
set send_slow {1 0.05}

# Set consistent terminal environment - disable colors and interactions
set env(TERM) "dumb"
set env(COLORTERM) ""
set env(NO_COLOR) "1"
set env(FORCE_COLOR) "0"

# Start bash with consistent environment
spawn env TERM=dumb COLORTERM= NO_COLOR=1 FORCE_COLOR=0 bash --noprofile --norc
expect "$ "

# Change to repo directory
send -s "cd $REPO_DIR\r"
expect "$ "

# Set simple PS1 to avoid complex prompts
send -s "export PS1='\$ '\r"
expect "$ "

# Clear screen and start demo
send -s "clear\r"
expect "$ "

# Add some initial context
send -s "# Buzz CLI Demo - Beeminder Terminal Interface\r"
expect "$ "
sleep 1

send -s "# Let's check our most urgent goal\r"
expect "$ "
sleep 0.5

# Demo 1: buzz next
send -s "./buzz next\r"
expect "$ "
sleep 3

send -s "# That shows: goalslug limsum timeframe\r"
expect "$ "
sleep 1

send -s "# Now let's see all goals due today\r"
expect "$ "
sleep 0.5

# Demo 2: buzz today  
send -s "./buzz today\r"
expect "$ "
sleep 3

send -s "# Finally, let's add a datapoint to a goal\r"
expect "$ "
sleep 0.5

# Demo 3: buzz add (using a test value)
send -s "./buzz add p3 0.1 \"Demo datapoint\"\r"
expect "$ "
sleep 2

send -s "# That's the buzz CLI! Use 'buzz --help' for more options\r"
expect "$ "
sleep 1

send -s "./buzz --help\r"
expect "$ "
sleep 3

send -s "exit\r"
expect eof
EOF

    chmod +x "$TEMP_EXPECT_FILE"
    log_info "Expect script created ✓"
}

# Record the demo
record_demo() {
    log_info "Starting asciinema recording..."
    log_info "Recording to: $RECORDING_FILE"
    
    # Ensure directory exists
    mkdir -p "$DEMOS_DIR"
    
    # Record with specific dimensions for better display
    if env TERM=dumb NO_COLOR=1 FORCE_COLOR=0 asciinema rec \
        --cols 80 \
        --rows 24 \
        --title "Buzz CLI Demo" \
        --env="TERM,NO_COLOR,FORCE_COLOR" \
        --quiet \
        "$RECORDING_FILE" \
        --command "expect -f $TEMP_EXPECT_FILE" 2>/dev/null; then
        log_info "Recording completed successfully ✓"
    else
        log_error "Recording failed"
        return 1
    fi
}

# Validate the recording
validate_recording() {
    log_info "Validating recording..."
    
    if [[ ! -f "$RECORDING_FILE" ]]; then
        log_error "Recording file not created"
        return 1
    fi
    
    # Check file size (should be > 0)
    if [[ ! -s "$RECORDING_FILE" ]]; then
        log_error "Recording file is empty"
        return 1
    fi
    
    # Try to get duration info
    local file_size
    file_size=$(du -h "$RECORDING_FILE" | cut -f1)
    log_info "Recording size: $file_size"
    
    # Quick play test (just check if it loads)
    if timeout 2 asciinema play --speed 10 "$RECORDING_FILE" >/dev/null 2>&1; then
        log_info "Recording validation passed ✓"
    else
        log_warn "Recording validation inconclusive (but file exists)"
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    [[ -f "$TEMP_EXPECT_FILE" ]] && rm -f "$TEMP_EXPECT_FILE"
}

# Main execution
main() {
    log_info "Starting Buzz CLI Demo Recording"
    log_info "================================"
    
    trap cleanup EXIT
    
    check_dependencies
    create_expect_script
    record_demo
    validate_recording
    
    log_info ""
    log_info "Demo recording completed successfully!"
    log_info "File: $RECORDING_FILE"
    log_info ""
    log_info "To play the recording:"
    log_info "  asciinema play '$RECORDING_FILE'"
    log_info ""
    log_info "To upload to asciinema.org:"
    log_info "  asciinema upload '$RECORDING_FILE'"
    log_info ""
    log_info "To convert to GIF (requires agg):"
    log_info "  agg '$RECORDING_FILE' '${RECORDING_FILE%.cast}.gif'"
}

# Run main function
main "$@"