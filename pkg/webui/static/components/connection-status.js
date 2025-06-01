/**
 * @fileoverview Connection status indicator component for real-time client connection monitoring
 * @module components/connection-status
 * @requires utils/logger
 * @author go-gamelaunch-client
 * @version 1.0.0
 */

import { createLogger, LogLevel } from "../utils/logger.js";

/**
 * @enum {string}
 * @readonly
 * @description Visual status indicator states with color coding
 */
const StatusState = {
  DISCONNECTED: "disconnected",
  CONNECTING: "connecting",
  CONNECTED: "connected",
  AUTHENTICATED: "authenticated",
  PLAYING: "playing",
  ERROR: "error",
  RECONNECTING: "reconnecting"
};

/**
 * @enum {Object}
 * @readonly
 * @description Status display configuration with colors and icons
 */
const StatusConfig = {
  [StatusState.DISCONNECTED]: {
    color: "#6c757d",
    backgroundColor: "#f8f9fa",
    icon: "●",
    text: "Disconnected",
    description: "Not connected to game server"
  },
  [StatusState.CONNECTING]: {
    color: "#ffc107",
    backgroundColor: "#fff3cd",
    icon: "◐",
    text: "Connecting",
    description: "Establishing connection to server"
  },
  [StatusState.CONNECTED]: {
    color: "#17a2b8",
    backgroundColor: "#d1ecf1",
    icon: "◑",
    text: "Connected",
    description: "Connected, initializing session"
  },
  [StatusState.AUTHENTICATED]: {
    color: "#28a745",
    backgroundColor: "#d4edda",
    icon: "◒",
    text: "Authenticated",
    description: "Session authenticated and ready"
  },
  [StatusState.PLAYING]: {
    color: "#28a745",
    backgroundColor: "#d4edda",
    icon: "●",
    text: "Playing",
    description: "Active game session"
  },
  [StatusState.ERROR]: {
    color: "#dc3545",
    backgroundColor: "#f8d7da",
    icon: "✕",
    text: "Error",
    description: "Connection error occurred"
  },
  [StatusState.RECONNECTING]: {
    color: "#fd7e14",
    backgroundColor: "#ffeaa7",
    icon: "↻",
    text: "Reconnecting",
    description: "Attempting to restore connection"
  }
};

/**
 * @class ConnectionHistory
 * @description Tracks connection history and statistics for status display
 */
class ConnectionHistory {
  /**
   * Creates a new ConnectionHistory instance
   * @param {number} [maxEntries=50] - Maximum number of history entries to keep
   */
  constructor(maxEntries = 50) {
    this.logger = createLogger("ConnectionHistory", LogLevel.DEBUG);
    this.maxEntries = maxEntries;
    this.entries = [];
    this.currentSession = null;

    this.logger.info(
      "constructor",
      `Connection history initialized with max entries: ${maxEntries}`
    );
  }

  /**
   * Records a connection state change
   * @param {string} state - New connection state
   * @param {string} [reason] - Optional reason for state change
   * @param {Object} [metadata] - Additional metadata about the change
   */
  recordStateChange(state, reason = null, metadata = {}) {
    const entry = {
      timestamp: Date.now(),
      state: state,
      reason: reason,
      metadata: { ...metadata },
      duration: 0 // Will be calculated when state changes
    };

    // Calculate duration of previous state
    if (this.entries.length > 0) {
      const lastEntry = this.entries[this.entries.length - 1];
      lastEntry.duration = entry.timestamp - lastEntry.timestamp;
    }

    this.entries.push(entry);

    // Maintain size limit
    if (this.entries.length > this.maxEntries) {
      this.entries.shift();
    }

    // Track current session
    if (
      state === StatusState.CONNECTED ||
      state === StatusState.AUTHENTICATED
    ) {
      this.currentSession = {
        startTime: entry.timestamp,
        initialState: state
      };
    } else if (
      state === StatusState.DISCONNECTED ||
      state === StatusState.ERROR
    ) {
      if (this.currentSession) {
        this.currentSession.endTime = entry.timestamp;
        this.currentSession.duration =
          this.currentSession.endTime - this.currentSession.startTime;
        this.currentSession = null;
      }
    }

    this.logger.debug("recordStateChange", `State change recorded: ${state}`, {
      reason: reason,
      totalEntries: this.entries.length
    });
  }

  /**
   * Gets connection statistics and metrics
   * @returns {Object} Connection statistics
   */
  getStatistics() {
    if (this.entries.length === 0) {
      return {
        totalSessions: 0,
        totalUptime: 0,
        averageSessionDuration: 0,
        connectionSuccessRate: 0,
        lastConnection: null,
        currentSession: this.currentSession
      };
    }

    const now = Date.now();
    const firstEntry = this.entries[0];
    const lastEntry = this.entries[this.entries.length - 1];

    // Calculate session statistics
    let totalSessions = 0;
    let totalUptime = 0;
    let connectionAttempts = 0;
    let successfulConnections = 0;

    for (const entry of this.entries) {
      if (entry.state === StatusState.CONNECTING) {
        connectionAttempts++;
      } else if (
        entry.state === StatusState.CONNECTED ||
        entry.state === StatusState.AUTHENTICATED
      ) {
        successfulConnections++;
        totalSessions++;
      } else if (entry.state === StatusState.PLAYING && entry.duration > 0) {
        totalUptime += entry.duration;
      }
    }

    // Add current session uptime if active
    if (this.currentSession) {
      totalUptime += now - this.currentSession.startTime;
    }

    const stats = {
      totalSessions: totalSessions,
      totalUptime: totalUptime,
      averageSessionDuration:
        totalSessions > 0 ? totalUptime / totalSessions : 0,
      connectionSuccessRate:
        connectionAttempts > 0
          ? successfulConnections / connectionAttempts * 100
          : 0,
      lastConnection: lastEntry ? lastEntry.timestamp : null,
      currentSession: this.currentSession,
      historyPeriod: lastEntry.timestamp - firstEntry.timestamp,
      totalStateChanges: this.entries.length
    };

    this.logger.debug("getStatistics", "Retrieved connection statistics", {
      totalSessions: stats.totalSessions,
      successRate: `${stats.connectionSuccessRate.toFixed(1)}%`
    });

    return stats;
  }

  /**
   * Gets recent connection history entries
   * @param {number} [limit=10] - Maximum number of entries to return
   * @returns {Array} Recent history entries
   */
  getRecentHistory(limit = 10) {
    const recent = this.entries.slice(-limit).map(entry => ({
      ...entry,
      relativeTime: this._formatRelativeTime(Date.now() - entry.timestamp)
    }));

    this.logger.debug(
      "getRecentHistory",
      `Retrieved ${recent.length} recent entries`
    );
    return recent;
  }

  /**
   * Formats a time duration in milliseconds to human-readable string
   * @param {number} ms - Duration in milliseconds
   * @returns {string} Formatted time string
   * @private
   */
  _formatRelativeTime(ms) {
    if (ms < 60000) {
      // Less than 1 minute
      return `${Math.floor(ms / 1000)}s ago`;
    } else if (ms < 3600000) {
      // Less than 1 hour
      return `${Math.floor(ms / 60000)}m ago`;
    } else if (ms < 86400000) {
      // Less than 1 day
      return `${Math.floor(ms / 3600000)}h ago`;
    } else {
      return `${Math.floor(ms / 86400000)}d ago`;
    }
  }

  /**
   * Clears all connection history
   */
  clear() {
    const entriesCleared = this.entries.length;
    this.entries = [];
    this.currentSession = null;

    this.logger.info("clear", `Cleared ${entriesCleared} history entries`);
  }
}

/**
 * @class StatusIndicator
 * @description Individual status indicator element with animation and tooltip support
 */
class StatusIndicator {
  /**
   * Creates a new StatusIndicator instance
   * @param {Object} [options={}] - Indicator configuration options
   * @param {boolean} [options.showIcon=true] - Whether to show status icon
   * @param {boolean} [options.showText=true] - Whether to show status text
   * @param {boolean} [options.showTooltip=true] - Whether to show detailed tooltip
   * @param {boolean} [options.animate=true] - Whether to animate state transitions
   * @param {string} [options.size='medium'] - Indicator size (small, medium, large)
   */
  constructor(options = {}) {
    this.logger = createLogger("StatusIndicator", LogLevel.DEBUG);

    this.options = {
      showIcon: options.showIcon !== false,
      showText: options.showText !== false,
      showTooltip: options.showTooltip !== false,
      animate: options.animate !== false,
      size: options.size || "medium",
      ...options
    };

    this.element = null;
    this.iconElement = null;
    this.textElement = null;
    this.tooltipElement = null;

    this.currentState = StatusState.DISCONNECTED;
    this.animationTimeout = null;

    this._createElement();

    this.logger.debug("constructor", "Status indicator created", this.options);
  }

  /**
   * Creates the DOM element structure for the indicator
   * @private
   */
  _createElement() {
    this.element = document.createElement("div");
    this.element.className = `status-indicator status-indicator--${
      this.options.size
    }`;

    // Apply base styles
    this._applyBaseStyles();

    // Create icon element
    if (this.options.showIcon) {
      this.iconElement = document.createElement("span");
      this.iconElement.className = "status-indicator__icon";
      this.element.appendChild(this.iconElement);
    }

    // Create text element
    if (this.options.showText) {
      this.textElement = document.createElement("span");
      this.textElement.className = "status-indicator__text";
      this.element.appendChild(this.textElement);
    }

    // Create tooltip element
    if (this.options.showTooltip) {
      this.tooltipElement = document.createElement("div");
      this.tooltipElement.className = "status-indicator__tooltip";
      this.element.appendChild(this.tooltipElement);

      // Add hover event listeners for tooltip
      this.element.addEventListener("mouseenter", () => this._showTooltip());
      this.element.addEventListener("mouseleave", () => this._hideTooltip());
    }

    // Set initial state
    this.updateState(this.currentState);

    this.logger.debug("_createElement", "DOM element structure created");
  }

  /**
   * Applies base CSS styles to the indicator element
   * @private
   */
  _applyBaseStyles() {
    const sizeMap = {
      small: { padding: "4px 8px", fontSize: "12px" },
      medium: { padding: "6px 12px", fontSize: "14px" },
      large: { padding: "8px 16px", fontSize: "16px" }
    };

    const size = sizeMap[this.options.size] || sizeMap.medium;

    Object.assign(this.element.style, {
      display: "inline-flex",
      alignItems: "center",
      gap: "6px",
      padding: size.padding,
      fontSize: size.fontSize,
      fontFamily: "monospace",
      borderRadius: "4px",
      border: "1px solid #dee2e6",
      position: "relative",
      cursor: "default",
      userSelect: "none",
      transition: this.options.animate ? "all 0.3s ease" : "none"
    });

    // Size-specific icon styles
    if (this.iconElement) {
      Object.assign(this.iconElement.style, {
        fontSize: `${parseInt(size.fontSize) + 2}px`,
        lineHeight: "1",
        transition: this.options.animate ? "transform 0.2s ease" : "none"
      });
    }

    // Tooltip styles
    if (this.tooltipElement) {
      Object.assign(this.tooltipElement.style, {
        position: "absolute",
        bottom: "100%",
        left: "50%",
        transform: "translateX(-50%)",
        marginBottom: "5px",
        padding: "8px 12px",
        backgroundColor: "#000",
        color: "#fff",
        fontSize: "12px",
        borderRadius: "4px",
        whiteSpace: "nowrap",
        opacity: "0",
        visibility: "hidden",
        transition: this.options.animate
          ? "opacity 0.2s ease, visibility 0.2s ease"
          : "none",
        zIndex: "1000",
        pointerEvents: "none"
      });

      // Tooltip arrow
      const arrow = document.createElement("div");
      Object.assign(arrow.style, {
        position: "absolute",
        top: "100%",
        left: "50%",
        transform: "translateX(-50%)",
        width: "0",
        height: "0",
        borderLeft: "5px solid transparent",
        borderRight: "5px solid transparent",
        borderTop: "5px solid #000"
      });
      this.tooltipElement.appendChild(arrow);
    }
  }

  /**
   * Updates the indicator state and visual appearance
   * @param {string} state - New state from StatusState enum
   * @param {Object} [metadata] - Additional state metadata
   */
  updateState(state, metadata = {}) {
    if (!StatusConfig[state]) {
      this.logger.warn("updateState", `Invalid state: ${state}`);
      return;
    }

    const previousState = this.currentState;
    this.currentState = state;
    const config = StatusConfig[state];

    this.logger.debug(
      "updateState",
      `State updated: ${previousState} -> ${state}`
    );

    // Clear any pending animation
    if (this.animationTimeout) {
      clearTimeout(this.animationTimeout);
      this.animationTimeout = null;
    }

    // Apply visual changes
    this._applyStateStyles(config);
    this._updateContent(config, metadata);

    // Add state-specific animations
    if (this.options.animate) {
      this._animateStateChange(state, previousState);
    }

    // Update CSS class for external styling
    this.element.className = `status-indicator status-indicator--${
      this.options.size
    } status-indicator--${state}`;
  }

  /**
   * Applies visual styles for the current state
   * @param {Object} config - State configuration object
   * @private
   */
  _applyStateStyles(config) {
    // Update element colors
    Object.assign(this.element.style, {
      color: config.color,
      backgroundColor: config.backgroundColor,
      borderColor: config.color
    });

    // Update icon
    if (this.iconElement) {
      this.iconElement.textContent = config.icon;
      this.iconElement.style.color = config.color;
    }

    // Update text
    if (this.textElement) {
      this.textElement.textContent = config.text;
    }
  }

  /**
   * Updates indicator content including tooltip
   * @param {Object} config - State configuration object
   * @param {Object} metadata - Additional metadata for display
   * @private
   */
  _updateContent(config, metadata) {
    if (!this.tooltipElement) {
      return;
    }

    let tooltipContent = config.description;

    // Add metadata to tooltip if available
    if (metadata.reason) {
      tooltipContent += `\nReason: ${metadata.reason}`;
    }
    if (metadata.attempts !== undefined) {
      tooltipContent += `\nAttempt: ${metadata.attempts}`;
    }
    if (metadata.latency !== undefined) {
      tooltipContent += `\nLatency: ${metadata.latency}ms`;
    }
    if (metadata.uptime !== undefined) {
      tooltipContent += `\nUptime: ${this._formatDuration(metadata.uptime)}`;
    }

    this.tooltipElement.firstChild.textContent = tooltipContent;
  }

  /**
   * Formats duration in milliseconds to human-readable string
   * @param {number} ms - Duration in milliseconds
   * @returns {string} Formatted duration
   * @private
   */
  _formatDuration(ms) {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  }

  /**
   * Animates state transition effects
   * @param {string} newState - New state
   * @param {string} previousState - Previous state
   * @private
   */
  _animateStateChange(newState, previousState) {
    // Pulse animation for important state changes
    if (
      (previousState === StatusState.CONNECTING &&
        newState === StatusState.CONNECTED) ||
      (previousState === StatusState.RECONNECTING &&
        newState === StatusState.CONNECTED)
    ) {
      this._pulseAnimation();
    }

    // Rotate icon for connecting/reconnecting states
    if (
      newState === StatusState.CONNECTING ||
      newState === StatusState.RECONNECTING
    ) {
      this._rotateAnimation();
    }

    // Shake animation for errors
    if (newState === StatusState.ERROR) {
      this._shakeAnimation();
    }
  }

  /**
   * Applies pulse animation effect
   * @private
   */
  _pulseAnimation() {
    if (!this.iconElement) return;

    this.iconElement.style.transform = "scale(1.2)";
    this.animationTimeout = setTimeout(() => {
      if (this.iconElement) {
        this.iconElement.style.transform = "scale(1)";
      }
    }, 200);
  }

  /**
   * Applies rotation animation for loading states
   * @private
   */
  _rotateAnimation() {
    if (!this.iconElement) return;

    let rotation = 0;
    const rotate = () => {
      if (
        this.currentState === StatusState.CONNECTING ||
        this.currentState === StatusState.RECONNECTING
      ) {
        rotation = (rotation + 45) % 360;
        this.iconElement.style.transform = `rotate(${rotation}deg)`;
        this.animationTimeout = setTimeout(rotate, 200);
      } else {
        this.iconElement.style.transform = "rotate(0deg)";
      }
    };
    rotate();
  }

  /**
   * Applies shake animation for error states
   * @private
   */
  _shakeAnimation() {
    if (!this.element) return;

    const originalTransform = this.element.style.transform;
    let shakeCount = 0;

    const shake = () => {
      if (shakeCount < 6) {
        const offset = shakeCount % 2 === 0 ? "2px" : "-2px";
        this.element.style.transform = `translateX(${offset})`;
        shakeCount++;
        this.animationTimeout = setTimeout(shake, 50);
      } else {
        this.element.style.transform = originalTransform;
      }
    };
    shake();
  }

  /**
   * Shows the tooltip
   * @private
   */
  _showTooltip() {
    if (this.tooltipElement) {
      this.tooltipElement.style.opacity = "1";
      this.tooltipElement.style.visibility = "visible";
    }
  }

  /**
   * Hides the tooltip
   * @private
   */
  _hideTooltip() {
    if (this.tooltipElement) {
      this.tooltipElement.style.opacity = "0";
      this.tooltipElement.style.visibility = "hidden";
    }
  }

  /**
   * Gets the DOM element for this indicator
   * @returns {HTMLElement} The indicator element
   */
  getElement() {
    return this.element;
  }

  /**
   * Destroys the indicator and cleans up resources
   */
  destroy() {
    if (this.animationTimeout) {
      clearTimeout(this.animationTimeout);
      this.animationTimeout = null;
    }

    if (this.element && this.element.parentNode) {
      this.element.parentNode.removeChild(this.element);
    }

    this.element = null;
    this.iconElement = null;
    this.textElement = null;
    this.tooltipElement = null;

    this.logger.debug("destroy", "Status indicator destroyed");
  }
}

/**
 * @class ConnectionStatus
 * @description Main connection status component managing multiple indicators and history
 */
class ConnectionStatus {
  /**
   * Creates a new ConnectionStatus instance
   * @param {Object} [options={}] - Component configuration options
   * @param {HTMLElement} [options.container] - Container element for status display
   * @param {Object} [options.indicator] - Status indicator options
   * @param {boolean} [options.showHistory=false] - Whether to show connection history
   * @param {boolean} [options.showStatistics=false] - Whether to show connection statistics
   * @param {number} [options.updateInterval=1000] - Update interval for statistics in milliseconds
   */
  constructor(options = {}) {
    this.logger = createLogger("ConnectionStatus", LogLevel.INFO);

    this.options = {
      showHistory: options.showHistory === true,
      showStatistics: options.showStatistics === true,
      updateInterval: options.updateInterval || 1000,
      ...options
    };

    this.container = options.container || null;
    this.indicator = new StatusIndicator(options.indicator);
    this.history = new ConnectionHistory();

    // DOM elements
    this.element = null;
    this.historyElement = null;
    this.statisticsElement = null;

    // Update management
    this.updateTimer = null;
    this.lastUpdateTime = 0;

    this._createElement();
    this._startUpdates();

    this.logger.info("constructor", "Connection status component initialized", {
      showHistory: this.options.showHistory,
      showStatistics: this.options.showStatistics
    });
  }

  /**
   * Creates the main component DOM structure
   * @private
   */
  _createElement() {
    this.element = document.createElement("div");
    this.element.className = "connection-status";

    // Apply base styles
    Object.assign(this.element.style, {
      display: "flex",
      flexDirection: "column",
      gap: "8px",
      padding: "8px",
      backgroundColor: "#f8f9fa",
      border: "1px solid #dee2e6",
      borderRadius: "4px",
      fontFamily: "monospace",
      fontSize: "12px"
    });

    // Add main indicator
    const indicatorContainer = document.createElement("div");
    indicatorContainer.className = "connection-status__indicator";
    indicatorContainer.appendChild(this.indicator.getElement());
    this.element.appendChild(indicatorContainer);

    // Add statistics section if enabled
    if (this.options.showStatistics) {
      this.statisticsElement = document.createElement("div");
      this.statisticsElement.className = "connection-status__statistics";
      Object.assign(this.statisticsElement.style, {
        fontSize: "11px",
        color: "#6c757d",
        borderTop: "1px solid #dee2e6",
        paddingTop: "8px",
        marginTop: "4px"
      });
      this.element.appendChild(this.statisticsElement);
    }

    // Add history section if enabled
    if (this.options.showHistory) {
      this.historyElement = document.createElement("div");
      this.historyElement.className = "connection-status__history";
      Object.assign(this.historyElement.style, {
        fontSize: "11px",
        color: "#6c757d",
        borderTop: "1px solid #dee2e6",
        paddingTop: "8px",
        marginTop: "4px",
        maxHeight: "120px",
        overflowY: "auto"
      });
      this.element.appendChild(this.historyElement);
    }

    // Add to container if provided
    if (this.container) {
      this.container.appendChild(this.element);
    }

    this.logger.debug("_createElement", "Component DOM structure created");
  }

  /**
   * Starts periodic updates for statistics and history
   * @private
   */
  _startUpdates() {
    if (this.options.showStatistics || this.options.showHistory) {
      this.updateTimer = setInterval(() => {
        this._updateDisplay();
      }, this.options.updateInterval);

      this.logger.debug(
        "_startUpdates",
        `Periodic updates started with ${
          this.options.updateInterval
        }ms interval`
      );
    }
  }

  /**
   * Updates the connection status display
   * @param {string} state - New connection state
   * @param {string} [reason] - Optional reason for state change
   * @param {Object} [metadata] - Additional metadata about the connection
   */
  updateStatus(state, reason = null, metadata = {}) {
    this.logger.debug("updateStatus", `Status update: ${state}`, {
      reason,
      metadata
    });

    // Update indicator
    this.indicator.updateState(state, metadata);

    // Record in history
    this.history.recordStateChange(state, reason, metadata);

    // Update display elements
    this._updateDisplay();

    this.lastUpdateTime = Date.now();
  }

  /**
   * Updates statistics and history display elements
   * @private
   */
  _updateDisplay() {
    if (this.statisticsElement) {
      this._updateStatistics();
    }

    if (this.historyElement) {
      this._updateHistory();
    }
  }

  /**
   * Updates the statistics display
   * @private
   */
  _updateStatistics() {
    const stats = this.history.getStatistics();

    const statisticsHTML = `
      <div><strong>Connection Statistics</strong></div>
      <div>Sessions: ${stats.totalSessions}</div>
      <div>Success Rate: ${stats.connectionSuccessRate.toFixed(1)}%</div>
      <div>Total Uptime: ${this._formatDuration(stats.totalUptime)}</div>
      ${
        stats.currentSession
          ? `<div>Current Session: ${this._formatDuration(
              Date.now() - stats.currentSession.startTime
            )}</div>`
          : ""
      }
      <div>Last Update: ${new Date(
        this.lastUpdateTime
      ).toLocaleTimeString()}</div>
    `;

    this.statisticsElement.innerHTML = statisticsHTML;
  }

  /**
   * Updates the history display
   * @private
   */
  _updateHistory() {
    const recentHistory = this.history.getRecentHistory(5);

    if (recentHistory.length === 0) {
      this.historyElement.innerHTML =
        "<div><strong>Connection History</strong></div><div>No history available</div>";
      return;
    }

    const historyHTML = recentHistory
      .map(entry => {
        const config = StatusConfig[entry.state] || {};
        return `
        <div style="margin: 2px 0; padding: 2px 4px; background: ${config.backgroundColor ||
          "#f8f9fa"}; border-radius: 2px;">
          <span style="color: ${config.color || "#000"};">${config.icon ||
          "●"}</span>
          ${config.text || entry.state} - ${entry.relativeTime}
          ${entry.reason ? ` (${entry.reason})` : ""}
        </div>
      `;
      })
      .join("");

    this.historyElement.innerHTML = `
      <div><strong>Recent History</strong></div>
      ${historyHTML}
    `;
  }

  /**
   * Formats duration in milliseconds to human-readable string
   * @param {number} ms - Duration in milliseconds
   * @returns {string} Formatted duration
   * @private
   */
  _formatDuration(ms) {
    if (ms < 60000) {
      return `${Math.floor(ms / 1000)}s`;
    } else if (ms < 3600000) {
      return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
    } else {
      return `${Math.floor(ms / 3600000)}h ${Math.floor(
        (ms % 3600000) / 60000
      )}m`;
    }
  }

  /**
   * Gets the main DOM element for this component
   * @returns {HTMLElement} The component element
   */
  getElement() {
    return this.element;
  }

  /**
   * Gets current connection statistics
   * @returns {Object} Connection statistics and status information
   */
  getStatistics() {
    return {
      currentState: this.indicator.currentState,
      lastUpdate: this.lastUpdateTime,
      history: this.history.getStatistics(),
      options: this.options
    };
  }

  /**
   * Clears connection history and resets statistics
   */
  reset() {
    this.logger.info("reset", "Resetting connection status component");

    this.history.clear();
    this.indicator.updateState(StatusState.DISCONNECTED);
    this.lastUpdateTime = Date.now();
    this._updateDisplay();
  }

  /**
   * Destroys the component and cleans up resources
   */
  destroy() {
    this.logger.enter("destroy");

    // Stop updates
    if (this.updateTimer) {
      clearInterval(this.updateTimer);
      this.updateTimer = null;
    }

    // Destroy indicator
    this.indicator.destroy();

    // Remove from DOM
    if (this.element && this.element.parentNode) {
      this.element.parentNode.removeChild(this.element);
    }

    // Clear references
    this.element = null;
    this.historyElement = null;
    this.statisticsElement = null;
    this.container = null;

    this.logger.info("destroy", "Connection status component destroyed");
  }
}

// Export public interface
export {
  ConnectionStatus,
  StatusIndicator,
  ConnectionHistory,
  StatusState,
  StatusConfig
};

console.log(
  "[ConnectionStatus] Connection status component module loaded successfully"
);
