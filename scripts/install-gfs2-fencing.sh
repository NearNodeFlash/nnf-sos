#!/bin/bash

# GFS2 Fencing Agent Installation Script
# This script installs the GFS2 fencing agent system

set -euo pipefail

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/gfs2-fencing}"
LOG_DIR="${LOG_DIR:-/var/log}"
DOC_DIR="${DOC_DIR:-/usr/local/share/doc/gfs2-fencing}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"

# Script directory (where this script is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_deps=()
    
    # Check for required commands
    for cmd in kubectl jq bash; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done
    
    # Check for optional but recommended commands
    for cmd in systemctl; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            log_warn "Optional dependency missing: $cmd (systemd service will not be available)"
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again"
        exit 1
    fi
    
    log_success "All required prerequisites are available"
}

# Function to create directories
create_directories() {
    log_info "Creating directories..."
    
    for dir in "$INSTALL_DIR" "$CONFIG_DIR" "$DOC_DIR"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log_info "Created directory: $dir"
        fi
    done
    
    # Create log directory if it doesn't exist
    if [[ ! -d "$LOG_DIR" ]]; then
        mkdir -p "$LOG_DIR"
        log_info "Created log directory: $LOG_DIR"
    fi
    
    log_success "Directories created successfully"
}

# Function to install scripts
install_scripts() {
    log_info "Installing scripts..."
    
    local scripts=(
        "gfs2-compute-fence-agent.sh"
        "k8s-gfs2-fence-agent.sh"
        "fence-event-controller.sh"
        "gfs2-fence-integration-example.sh"
    )
    
    for script in "${scripts[@]}"; do
        local src_path="$PROJECT_ROOT/scripts/$script"
        local dst_path="$INSTALL_DIR/$script"
        
        if [[ -f "$src_path" ]]; then
            cp "$src_path" "$dst_path"
            chmod +x "$dst_path"
            log_info "Installed: $script"
        else
            log_error "Source script not found: $src_path"
            exit 1
        fi
    done
    
    log_success "Scripts installed successfully"
}

# Function to install Kubernetes manifests
install_k8s_manifests() {
    log_info "Installing Kubernetes manifests..."
    
    local manifests_dir="$CONFIG_DIR/kubernetes"
    mkdir -p "$manifests_dir"
    
    local manifests=(
        "computenodefenceevent-crd.yaml"
        "gfs2-fence-controller.yaml"
    )
    
    for manifest in "${manifests[@]}"; do
        local src_path="$PROJECT_ROOT/config/crd/$manifest"
        local dst_path="$manifests_dir/$manifest"
        
        if [[ -f "$src_path" ]]; then
            cp "$src_path" "$dst_path"
            log_info "Installed K8s manifest: $manifest"
        else
            # Try alternative locations
            src_path="$PROJECT_ROOT/config/manager/$manifest"
            if [[ -f "$src_path" ]]; then
                cp "$src_path" "$dst_path"
                log_info "Installed K8s manifest: $manifest"
            else
                log_warn "Kubernetes manifest not found: $manifest"
            fi
        fi
    done
    
    log_success "Kubernetes manifests installed"
}

# Function to install systemd service
install_systemd_service() {
    if ! command -v systemctl >/dev/null 2>&1; then
        log_warn "systemctl not available, skipping systemd service installation"
        return 0
    fi
    
    log_info "Installing systemd service..."
    
    local service_file="$PROJECT_ROOT/config/systemd/gfs2-fence-monitor.service"
    local dst_path="$SYSTEMD_DIR/gfs2-fence-monitor.service"
    
    if [[ -f "$service_file" ]]; then
        cp "$service_file" "$dst_path"
        systemctl daemon-reload
        log_info "Systemd service installed: gfs2-fence-monitor.service"
        log_info "To enable and start: systemctl enable --now gfs2-fence-monitor"
    else
        log_warn "Systemd service file not found: $service_file"
    fi
    
    log_success "Systemd service installation completed"
}

# Function to install documentation
install_documentation() {
    log_info "Installing documentation..."
    
    local docs=(
        "GFS2-Fencing-Agent.md"
        "../Readme.md"
    )
    
    for doc in "${docs[@]}"; do
        local src_path="$PROJECT_ROOT/scripts/$doc"
        local dst_name="$(basename "$doc")"
        local dst_path="$DOC_DIR/$dst_name"
        
        if [[ -f "$src_path" ]]; then
            cp "$src_path" "$dst_path"
            log_info "Installed documentation: $dst_name"
        else
            log_warn "Documentation not found: $src_path"
        fi
    done
    
    log_success "Documentation installed"
}

# Function to create configuration template
create_config_template() {
    log_info "Creating configuration template..."
    
    local config_file="$CONFIG_DIR/gfs2-fencing.conf"
    
    cat > "$config_file" << 'EOF'
# GFS2 Fencing Agent Configuration
# Copy this file and customize for your environment

# Kubernetes configuration
KUBECONFIG="/root/.kube/config"
KUBERNETES_NAMESPACE="nnf-system"

# Fence agent configuration
FENCE_TIMEOUT="300"
FENCE_RETRIES="3"
FENCE_RETRY_DELAY="30"

# Logging configuration
LOG_LEVEL="INFO"
LOG_FILE="/var/log/gfs2-fence-integration.log"
LOG_MAX_SIZE="100M"
LOG_ROTATE_COUNT="5"

# Notification configuration (customize for your environment)
ENABLE_EMAIL_ALERTS="false"
EMAIL_RECIPIENTS="admin@example.com"
ENABLE_SLACK_ALERTS="false"
SLACK_WEBHOOK_URL=""

# Monitoring configuration
HEALTH_CHECK_INTERVAL="60"
LOCK_CONTENTION_THRESHOLD="300"
NODE_RESPONSE_TIMEOUT="30"

# GFS2 specific settings
GFS2_MOUNT_PATHS="/gfs2"
GFS2_LOCK_DUMP_TIMEOUT="30"

# Advanced settings
ENABLE_DRY_RUN="false"
ENABLE_DETAILED_LOGGING="false"
FENCE_COOLDOWN_PERIOD="600"
EOF

    chmod 600 "$config_file"
    log_info "Configuration template created: $config_file"
    log_success "Configuration template installation completed"
}

# Function to set up RBAC for Kubernetes
setup_k8s_rbac() {
    log_info "Setting up Kubernetes RBAC..."
    
    local rbac_file="$CONFIG_DIR/kubernetes/gfs2-fence-rbac.yaml"
    
    cat > "$rbac_file" << 'EOF'
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gfs2-fence-agent
  namespace: nnf-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gfs2-fence-agent
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: ["nnf.cray.hpe.com"]
  resources: ["*"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: ["dataworkflowservices.github.io"]
  resources: ["*"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch", "create", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gfs2-fence-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gfs2-fence-agent
subjects:
- kind: ServiceAccount
  name: gfs2-fence-agent
  namespace: nnf-system
EOF

    log_info "RBAC configuration created: $rbac_file"
    log_success "Kubernetes RBAC setup completed"
}

# Function to validate installation
validate_installation() {
    log_info "Validating installation..."
    
    local errors=0
    
    # Check scripts
    local scripts=(
        "$INSTALL_DIR/gfs2-compute-fence-agent.sh"
        "$INSTALL_DIR/k8s-gfs2-fence-agent.sh"
        "$INSTALL_DIR/fence-event-controller.sh"
        "$INSTALL_DIR/gfs2-fence-integration-example.sh"
    )
    
    for script in "${scripts[@]}"; do
        if [[ ! -x "$script" ]]; then
            log_error "Script not executable: $script"
            ((errors++))
        fi
    done
    
    # Check configuration
    if [[ ! -f "$CONFIG_DIR/gfs2-fencing.conf" ]]; then
        log_error "Configuration file missing: $CONFIG_DIR/gfs2-fencing.conf"
        ((errors++))
    fi
    
    # Test basic functionality
    if ! "$INSTALL_DIR/k8s-gfs2-fence-agent.sh" --help >/dev/null 2>&1; then
        log_error "Main fence agent script test failed"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log_success "Installation validation passed"
        return 0
    else
        log_error "Installation validation failed with $errors errors"
        return 1
    fi
}

# Function to show post-installation instructions
show_post_install_instructions() {
    log_success "GFS2 Fencing Agent installation completed successfully!"
    
    echo
    echo "Next steps:"
    echo "1. Review and customize the configuration:"
    echo "   $CONFIG_DIR/gfs2-fencing.conf"
    echo
    echo "2. Deploy the Kubernetes components:"
    echo "   kubectl apply -f $CONFIG_DIR/kubernetes/"
    echo
    echo "3. Test the fence agent:"
    echo "   $INSTALL_DIR/k8s-gfs2-fence-agent.sh list"
    echo
    echo "4. (Optional) Enable the monitoring service:"
    echo "   systemctl enable --now gfs2-fence-monitor"
    echo
    echo "5. Check the documentation:"
    echo "   $DOC_DIR/GFS2-Fencing-Agent.md"
    echo
    echo "For troubleshooting, check the logs:"
    echo "   tail -f $LOG_DIR/gfs2-fence-integration.log"
    echo "   journalctl -u gfs2-fence-monitor -f"
}

# Function to uninstall
uninstall() {
    log_info "Uninstalling GFS2 Fencing Agent..."
    
    # Stop and disable systemd service
    if command -v systemctl >/dev/null 2>&1; then
        if systemctl is-active --quiet gfs2-fence-monitor; then
            systemctl stop gfs2-fence-monitor
            log_info "Stopped gfs2-fence-monitor service"
        fi
        
        if systemctl is-enabled --quiet gfs2-fence-monitor; then
            systemctl disable gfs2-fence-monitor
            log_info "Disabled gfs2-fence-monitor service"
        fi
        
        if [[ -f "$SYSTEMD_DIR/gfs2-fence-monitor.service" ]]; then
            rm -f "$SYSTEMD_DIR/gfs2-fence-monitor.service"
            systemctl daemon-reload
            log_info "Removed systemd service file"
        fi
    fi
    
    # Remove scripts
    local scripts=(
        "$INSTALL_DIR/gfs2-compute-fence-agent.sh"
        "$INSTALL_DIR/k8s-gfs2-fence-agent.sh"
        "$INSTALL_DIR/fence-event-controller.sh"
        "$INSTALL_DIR/gfs2-fence-integration-example.sh"
    )
    
    for script in "${scripts[@]}"; do
        if [[ -f "$script" ]]; then
            rm -f "$script"
            log_info "Removed script: $(basename "$script")"
        fi
    done
    
    # Remove configuration and documentation
    if [[ -d "$CONFIG_DIR" ]]; then
        rm -rf "$CONFIG_DIR"
        log_info "Removed configuration directory: $CONFIG_DIR"
    fi
    
    if [[ -d "$DOC_DIR" ]]; then
        rm -rf "$DOC_DIR"
        log_info "Removed documentation directory: $DOC_DIR"
    fi
    
    # Note: We don't remove log files as they might be needed for troubleshooting
    log_warn "Log files preserved in $LOG_DIR"
    
    log_success "GFS2 Fencing Agent uninstalled successfully"
}

# Usage function
usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

COMMANDS:
    install     - Install the GFS2 fencing agent system (default)
    uninstall   - Remove the GFS2 fencing agent system
    validate    - Validate existing installation
    help        - Show this help message

OPTIONS:
    --install-dir DIR     - Installation directory for scripts (default: $INSTALL_DIR)
    --config-dir DIR      - Configuration directory (default: $CONFIG_DIR)
    --doc-dir DIR         - Documentation directory (default: $DOC_DIR)
    --log-dir DIR         - Log directory (default: $LOG_DIR)

EXAMPLES:
    # Standard installation
    $0 install

    # Custom installation directories
    $0 install --install-dir /opt/gfs2-fence --config-dir /etc/gfs2

    # Uninstall
    $0 uninstall

    # Validate installation
    $0 validate

ENVIRONMENT VARIABLES:
    INSTALL_DIR    - Override default installation directory
    CONFIG_DIR     - Override default configuration directory
    DOC_DIR        - Override default documentation directory
    LOG_DIR        - Override default log directory
EOF
}

# Parse command line arguments
COMMAND="install"
while [[ $# -gt 0 ]]; do
    case $1 in
        install|uninstall|validate|help)
            COMMAND="$1"
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --config-dir)
            CONFIG_DIR="$2"
            shift 2
            ;;
        --doc-dir)
            DOC_DIR="$2"
            shift 2
            ;;
        --log-dir)
            LOG_DIR="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
case "$COMMAND" in
    "install")
        check_root
        check_prerequisites
        create_directories
        install_scripts
        install_k8s_manifests
        install_systemd_service
        install_documentation
        create_config_template
        setup_k8s_rbac
        validate_installation
        show_post_install_instructions
        ;;
    "uninstall")
        check_root
        uninstall
        ;;
    "validate")
        validate_installation
        ;;
    "help")
        usage
        ;;
    *)
        log_error "Unknown command: $COMMAND"
        usage
        exit 1
        ;;
esac