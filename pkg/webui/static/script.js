class GameClient {
  constructor() {
    this.rpcId = 1;
    this.gameState = null;
    this.tileset = null;
    this.pollTimeout = 30000; // 30 seconds
    this.polling = false;
    this.canvas = null;
    this.ctx = null;
    this.tilesetImage = null;
    this.debugInput = true; // Add debug flag

    this.init();

    // Add keyboard shortcut to toggle input debugging (Ctrl+Shift+D)
    document.addEventListener("keydown", e => {
      if (e.ctrlKey && e.shiftKey && e.key === "D") {
        this.debugInput = !this.debugInput;
        console.log("Input debugging:", this.debugInput ? "ON" : "OFF");
        if (this.debugInput && this.inputStats) {
          this.logInputStatistics();
        }
      }
    });
  }

  async init() {
    this.setupCanvas();
    this.setupEventListeners();

    try {
      await this.loadTileset();
      await this.loadInitialState();
      this.startPolling();
    } catch (error) {
      console.error("Failed to initialize game client:", error);
      this.showError("Failed to connect to game server");
    }
  }

  setupCanvas() {
    this.canvas = document.getElementById("gameCanvas");
    if (!this.canvas) {
      // Create canvas if it doesn't exist
      this.canvas = document.createElement("canvas");
      this.canvas.id = "gameCanvas";
      this.canvas.width = 800;
      this.canvas.height = 600;
      this.canvas.style.border = "1px solid #ccc";
      this.canvas.style.backgroundColor = "#000";
      document.body.appendChild(this.canvas);
    }

    this.ctx = this.canvas.getContext("2d");
    this.ctx.imageSmoothingEnabled = false; // Pixel-perfect rendering
  }

  setupEventListeners() {
    // Keyboard input
    document.addEventListener("keydown", e => {
      this.handleKeyDown(e);
    });

    // Canvas focus management
    this.canvas.addEventListener("click", () => {
      this.canvas.focus();
    });

    this.canvas.setAttribute("tabindex", "0");
    this.canvas.focus();

    // Prevent default browser behavior for game keys
    document.addEventListener("keydown", e => {
      if (
        ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Space"].includes(
          e.key
        )
      ) {
        e.preventDefault();
      }
    });
  }

  async rpcCall(method, params = {}) {
    const request = {
      jsonrpc: "2.0",
      method: method,
      params: params,
      id: this.rpcId++
    };

    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 35000); // 35 second timeout

      const response = await fetch("/rpc", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify(request),
        signal: controller.signal
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const responseText = await response.text();

      let result;
      try {
        result = JSON.parse(responseText);
      } catch (parseError) {
        console.error(
          `Failed to parse JSON response for ${method}:`,
          parseError
        );
        console.error("Raw response:", responseText);
        throw new Error(`Invalid JSON response: ${parseError.message}`);
      }

      if (result.error) {
        throw new Error(
          `RPC Error ${result.error.code}: ${result.error.message}`
        );
      }

      return result.result;
    } catch (error) {
      if (error.name === "AbortError") {
        throw new Error(`Request timeout for ${method}`);
      }
      console.error(`RPC call failed for ${method}:`, error);
      throw error;
    }
  }

  async loadTileset() {
    try {
      const result = await this.rpcCall("tileset.fetch");

      if (result.tileset) {
        this.tileset = result.tileset;

        if (result.image_available) {
          await this.loadTilesetImage();
        }
      }
    } catch (error) {
      console.warn(
        "Failed to load tileset, falling back to text rendering:",
        error
      );
    }
  }

  async loadTilesetImage() {
    return new Promise((resolve, reject) => {
      this.tilesetImage = new Image();
      this.tilesetImage.onload = () => {
        console.log("Tileset image loaded successfully");
        resolve();
      };
      this.tilesetImage.onerror = () => {
        console.warn("Failed to load tileset image, using text rendering");
        this.tilesetImage = null;
        resolve(); // Don't reject, just fall back to text
      };
      this.tilesetImage.src = "/tileset/image?" + Date.now(); // Cache busting
    });
  }

  async loadInitialState() {
    const result = await this.rpcCall("game.getState");

    if (result.state) {
      this.gameState = result.state;
      this.resizeCanvas();
      this.render();
    }
  }

  resizeCanvas() {
    if (!this.gameState) return;

    const cellWidth = this.tileset ? this.tileset.tile_width : 12;
    const cellHeight = this.tileset ? this.tileset.tile_height : 16;

    this.canvas.width = this.gameState.width * cellWidth;
    this.canvas.height = this.gameState.height * cellHeight;

    // Re-get context after resize
    this.ctx = this.canvas.getContext("2d");
    this.ctx.imageSmoothingEnabled = false;
  }

  async startPolling() {
    if (this.polling) return;
    this.polling = true;

    let consecutiveErrors = 0;
    const maxConsecutiveErrors = 5;

    const poll = async () => {
      if (!this.polling) return;

      try {
        const version = this.gameState ? this.gameState.version : 0;
        console.log(`Polling for changes since version ${version}`);

        const result = await this.rpcCall("game.poll", {
          version: version,
          timeout: 25000
        });

        consecutiveErrors = 0; // Reset error counter on success

        if (result.changes && !result.timeout) {
          console.log(
            `Received ${result.changes.length} changes, version ${
              result.version
            }`
          );
          this.applyChanges(result.changes);
          this.render();
        } else if (result.timeout) {
          console.log("Poll timeout - no changes");
        }

        setTimeout(poll, 100);
      } catch (error) {
        consecutiveErrors++;
        console.error(
          `Polling error (${consecutiveErrors}/${maxConsecutiveErrors}):`,
          error
        );

        if (consecutiveErrors >= maxConsecutiveErrors) {
          console.error(
            "Too many consecutive polling errors, stopping polling"
          );
          this.polling = false;
          this.showError("Connection lost - too many errors");
          return;
        }

        // Determine retry delay based on error type
        let retryDelay = 5000; // Default 5 seconds

        if (error.message.includes("timeout")) {
          retryDelay = 100; // Quick retry for timeouts
        } else if (
          error.message.includes("NetworkError") ||
          error.message.includes("Failed to fetch")
        ) {
          retryDelay = Math.min(5000 * consecutiveErrors, 30000); // Exponential backoff up to 30s
        }

        setTimeout(poll, retryDelay);
      }
    };

    poll();
  }

  stopPolling() {
    this.polling = false;
  }

  render() {
    if (!this.gameState || !this.ctx) return;

    const cellWidth = this.tileset ? this.tileset.tile_width : 12;
    const cellHeight = this.tileset ? this.tileset.tile_height : 16;

    // Clear canvas
    this.ctx.fillStyle = "#000000";
    this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);

    // Render cells
    for (let y = 0; y < this.gameState.buffer.length; y++) {
      const row = this.gameState.buffer[y];
      for (let x = 0; x < row.length; x++) {
        const cell = row[x];
        this.renderCell(x, y, cell, cellWidth, cellHeight);
      }
    }

    // Render cursor
    this.renderCursor(cellWidth, cellHeight);
  }

  renderTextFallback(x, y, width, height, cell) {
    // Calculate optimal font size to fit the tile dimensions
    let fontSize = Math.min(width * 0.8, height * 0.8);
    fontSize = Math.max(fontSize, 8);

    this.ctx.font = `${fontSize}px monospace`;
    this.ctx.textAlign = "center";
    this.ctx.textBaseline = "middle";

    if (cell.bold) {
      this.ctx.font = `bold ${fontSize}px monospace`;
    }

    // Handle background color first
    let bgColor = cell.bg_color || "#000000";
    let fgColor = cell.fg_color || "#FFFFFF";

    // Handle inverse rendering
    if (cell.inverse) {
      // Swap foreground and background colors
      [bgColor, fgColor] = [fgColor, bgColor];
    }

    // Always render background if it's not the default black
    if (bgColor !== "#000000") {
      this.ctx.fillStyle = bgColor;
      this.ctx.fillRect(x, y, width, height);
    }

    // Set text color
    this.ctx.fillStyle = fgColor;

    // Calculate text position (centered in tile)
    const textX = x + width / 2;
    const textY = y + height / 2;

    // Handle character rendering - support both char codes and strings
    let char;
    if (typeof cell.char === "number") {
      char = String.fromCharCode(cell.char);
    } else {
      char = cell.char || " ";
    }

    if (char !== " ") {
      // Measure text to ensure it fits
      const metrics = this.ctx.measureText(char);

      // If text is too wide, reduce font size
      if (metrics.width > width * 0.9) {
        fontSize = fontSize * (width * 0.9) / metrics.width;
        this.ctx.font = cell.bold
          ? `bold ${fontSize}px monospace`
          : `${fontSize}px monospace`;
      }

      // Add text shadow for better readability on colored backgrounds
      if (bgColor !== "#000000") {
        this.ctx.shadowColor = "rgba(0, 0, 0, 0.5)";
        this.ctx.shadowOffsetX = 1;
        this.ctx.shadowOffsetY = 1;
        this.ctx.shadowBlur = 1;
      }

      // Render the character
      this.ctx.fillText(char, textX, textY);

      // Reset shadow
      this.ctx.shadowColor = "transparent";
      this.ctx.shadowOffsetX = 0;
      this.ctx.shadowOffsetY = 0;
      this.ctx.shadowBlur = 0;
    }

    // Handle blinking effect
    if (cell.blink) {
      const blink = Math.floor(Date.now() / 500) % 2;
      if (!blink) {
        // Hide character on blink
        this.ctx.fillStyle = bgColor;
        this.ctx.fillRect(x, y, width, height);
      }
    }
  }

  renderCell(x, y, cell, cellWidth, cellHeight) {
    const pixelX = x * cellWidth;
    const pixelY = y * cellHeight;

    // Always render background first
    const bgColor = cell.bg_color || "#000000";
    if (bgColor !== "#000000") {
      this.ctx.fillStyle = bgColor;
      this.ctx.fillRect(pixelX, pixelY, cellWidth, cellHeight);
    }

    // Character rendering with tileset or text fallback
    if (
      this.tilesetImage &&
      this.tileset &&
      cell.tile_x !== undefined &&
      cell.tile_y !== undefined
    ) {
      // Render using tileset
      const srcX = cell.tile_x * this.tileset.tile_width;
      const srcY = cell.tile_y * this.tileset.tile_height;

      // For colored tileset rendering, we might need to apply color tinting
      if (cell.fg_color && cell.fg_color !== "#FFFFFF") {
        // Save the current composite operation
        const oldComposite = this.ctx.globalCompositeOperation;

        // Draw the tile
        this.ctx.drawImage(
          this.tilesetImage,
          srcX,
          srcY,
          this.tileset.tile_width,
          this.tileset.tile_height,
          pixelX,
          pixelY,
          cellWidth,
          cellHeight
        );

        // Apply color tinting
        this.ctx.globalCompositeOperation = "multiply";
        this.ctx.fillStyle = cell.fg_color;
        this.ctx.fillRect(pixelX, pixelY, cellWidth, cellHeight);

        // Restore composite operation
        this.ctx.globalCompositeOperation = oldComposite;
      } else {
        // Normal tileset rendering
        this.ctx.drawImage(
          this.tilesetImage,
          srcX,
          srcY,
          this.tileset.tile_width,
          this.tileset.tile_height,
          pixelX,
          pixelY,
          cellWidth,
          cellHeight
        );
      }
    } else {
      // Text fallback rendering
      this.renderTextFallback(pixelX, pixelY, cellWidth, cellHeight, cell);
    }
  }

  // Add helper method for color parsing and validation
  parseColor(color) {
    // Handle various color formats
    if (!color) return "#000000";

    // If it's already a hex color, return as-is
    if (color.startsWith("#")) {
      return color;
    }

    // Handle RGB values
    if (color.startsWith("rgb")) {
      return color;
    }

    // Handle named colors or convert from other formats
    const namedColors = {
      black: "#000000",
      red: "#FF0000",
      green: "#00FF00",
      yellow: "#FFFF00",
      blue: "#0000FF",
      magenta: "#FF00FF",
      cyan: "#00FFFF",
      white: "#FFFFFF",
      bright_black: "#808080",
      bright_red: "#FF8080",
      bright_green: "#80FF80",
      bright_yellow: "#FFFF80",
      bright_blue: "#8080FF",
      bright_magenta: "#FF80FF",
      bright_cyan: "#80FFFF",
      bright_white: "#FFFFFF"
    };

    return namedColors[color.toLowerCase()] || color;
  }

  applyChanges(diff) {
    if (!this.gameState) {
      this.gameState = {
        buffer: [],
        width: 0,
        height: 0,
        cursor_x: diff.cursor_x || 0,
        cursor_y: diff.cursor_y || 0,
        version: diff.version
      };
    }

    // Update cursor position
    this.gameState.cursor_x = diff.cursor_x;
    this.gameState.cursor_y = diff.cursor_y;
    this.gameState.version = diff.version;

    // Apply cell changes
    diff.changes.forEach(change => {
      // Ensure buffer is large enough
      while (this.gameState.buffer.length <= change.y) {
        this.gameState.buffer.push([]);
      }

      while (this.gameState.buffer[change.y].length <= change.x) {
        this.gameState.buffer[change.y].push({
          char: " ",
          fg_color: "#FFFFFF",
          bg_color: "#000000"
        });
      }

      // Parse and normalize colors
      const cell = { ...change.cell };
      if (cell.fg_color) {
        cell.fg_color = this.parseColor(cell.fg_color);
      }
      if (cell.bg_color) {
        cell.bg_color = this.parseColor(cell.bg_color);
      }

      this.gameState.buffer[change.y][change.x] = cell;
    });

    // Update dimensions
    this.gameState.height = this.gameState.buffer.length;
    this.gameState.width = Math.max(
      ...this.gameState.buffer.map(row => row.length)
    );

    // Resize canvas if needed
    this.resizeCanvas();
  }

  renderCursor(cellWidth, cellHeight) {
    if (this.gameState.cursor_x >= 0 && this.gameState.cursor_y >= 0) {
      const cursorX = this.gameState.cursor_x * cellWidth;
      const cursorY = this.gameState.cursor_y * cellHeight;

      // Blinking cursor effect
      const blink = Math.floor(Date.now() / 500) % 2;
      if (blink) {
        this.ctx.strokeStyle = "#FFFFFF";
        this.ctx.lineWidth = 1;
        this.ctx.strokeRect(cursorX, cursorY, cellWidth, cellHeight);
      }
    }

    // Schedule next render for cursor blink
    setTimeout(() => this.render(), 500);
  }

  handleKeyDown(event) {
    // Prevent default behavior for specific game keys
    if (
      [
        "ArrowUp",
        "ArrowDown",
        "ArrowLeft",
        "ArrowRight",
        "Space",
        "Enter",
        "Escape"
      ].includes(event.key)
    ) {
      event.preventDefault();
    }

    const inputEvent = {
      type: "keydown",
      key: event.key,
      keyCode: event.keyCode,
      code: event.code,
      ctrlKey: event.ctrlKey,
      shiftKey: event.shiftKey,
      altKey: event.altKey,
      metaKey: event.metaKey,
      repeat: event.repeat,
      timestamp: Date.now()
    };

    // Enhanced logging with more details
    console.log("Keyboard input:", {
      key: event.key,
      keyCode: event.keyCode,
      code: event.code,
      modifiers: {
        ctrl: event.ctrlKey,
        shift: event.shiftKey,
        alt: event.altKey,
        meta: event.metaKey
      },
      repeat: event.repeat,
      timestamp: inputEvent.timestamp,
      preventDefault: [
        "ArrowUp",
        "ArrowDown",
        "ArrowLeft",
        "ArrowRight",
        "Space",
        "Enter",
        "Escape"
      ].includes(event.key)
    });

    this.sendInput([inputEvent]);
  }

  logInputStatistics() {
    if (!this.inputStats) {
      this.inputStats = {
        totalInputs: 0,
        keyPresses: new Map(),
        errors: 0,
        averageLatency: 0,
        latencies: []
      };
    }

    console.group("Input Statistics");
    console.log("Total inputs sent:", this.inputStats.totalInputs);
    console.log("Input errors:", this.inputStats.errors);
    console.log(
      "Average latency:",
      this.inputStats.averageLatency.toFixed(2) + "ms"
    );
    console.log(
      "Most pressed keys:",
      [...this.inputStats.keyPresses.entries()]
        .sort((a, b) => b[1] - a[1])
        .slice(0, 5)
    );
    console.groupEnd();
  }

  async sendInput(events) {
    try {
      if (!this.inputStats) {
        this.inputStats = {
          totalInputs: 0,
          keyPresses: new Map(),
          errors: 0,
          averageLatency: 0,
          latencies: []
        };
      }

      console.log(
        "Sending input events to server:",
        events.map(e => ({
          type: e.type,
          key: e.key,
          keyCode: e.keyCode,
          timestamp: e.timestamp
        }))
      );

      const startTime = performance.now();
      await this.rpcCall("game.sendInput", { events: events });
      const endTime = performance.now();

      const latency = endTime - startTime;

      // Update statistics
      this.inputStats.totalInputs++;
      this.inputStats.latencies.push(latency);
      if (this.inputStats.latencies.length > 100) {
        this.inputStats.latencies.shift(); // Keep only last 100 measurements
      }
      this.inputStats.averageLatency =
        this.inputStats.latencies.reduce((a, b) => a + b, 0) /
        this.inputStats.latencies.length;

      events.forEach(event => {
        const count = this.inputStats.keyPresses.get(event.key) || 0;
        this.inputStats.keyPresses.set(event.key, count + 1);
      });

      console.log(
        `Input sent successfully in ${latency.toFixed(
          2
        )}ms (avg: ${this.inputStats.averageLatency.toFixed(2)}ms)`
      );

      // Log statistics every 20 inputs
      if (this.inputStats.totalInputs % 20 === 0) {
        this.logInputStatistics();
      }
    } catch (error) {
      if (this.inputStats) {
        this.inputStats.errors++;
      }
      console.error("Failed to send input:", error);
      console.error("Failed input events:", events);
      this.showError("Failed to send input to server");
    }
  }

  showError(message) {
    // Create or update error display
    let errorDiv = document.getElementById("errorMessage");
    if (!errorDiv) {
      errorDiv = document.createElement("div");
      errorDiv.id = "errorMessage";
      errorDiv.style.cssText = `
                position: fixed;
                top: 10px;
                right: 10px;
                background: #ff4444;
                color: white;
                padding: 10px;
                border-radius: 5px;
                z-index: 1000;
                font-family: monospace;
            `;
      document.body.appendChild(errorDiv);
    }

    errorDiv.textContent = message;

    // Auto-hide after 5 seconds
    setTimeout(() => {
      if (errorDiv.parentNode) {
        errorDiv.parentNode.removeChild(errorDiv);
      }
    }, 5000);
  }
}

// Connection status indicator
class ConnectionStatus {
  constructor() {
    this.createStatusIndicator();
    this.updateStatus("connecting");
  }

  createStatusIndicator() {
    this.statusDiv = document.createElement("div");
    this.statusDiv.id = "connectionStatus";
    this.statusDiv.style.cssText = `
            position: fixed;
            top: 10px;
            left: 10px;
            padding: 5px 10px;
            border-radius: 3px;
            font-family: monospace;
            font-size: 12px;
            z-index: 1000;
        `;
    document.body.appendChild(this.statusDiv);
  }

  updateStatus(status) {
    const styles = {
      connecting: { bg: "#ff9500", text: "Connecting...", color: "white" },
      connected: { bg: "#00aa00", text: "Connected", color: "white" },
      disconnected: { bg: "#aa0000", text: "Disconnected", color: "white" },
      error: { bg: "#ff0000", text: "Connection Error", color: "white" }
    };

    const style = styles[status] || styles.disconnected;
    this.statusDiv.style.backgroundColor = style.bg;
    this.statusDiv.style.color = style.color;
    this.statusDiv.textContent = style.text;
  }
}

// Instructions panel
function createInstructions() {
  const instructions = document.createElement("div");
  instructions.style.cssText = `
        position: fixed;
        bottom: 10px;
        left: 10px;
        background: rgba(0, 0, 0, 0.8);
        color: white;
        padding: 10px;
        border-radius: 5px;
        font-family: monospace;
        font-size: 12px;
        max-width: 300px;
        z-index: 1000;
    `;

  instructions.innerHTML = `
        <strong>Game Controls:</strong><br>
        • Arrow keys: Move<br>
        • Enter: Confirm<br>
        • Escape: Cancel/Menu<br>
        • Click canvas to focus
    `;

  document.body.appendChild(instructions);
}

// Initialize when page loads
document.addEventListener("DOMContentLoaded", () => {
  const connectionStatus = new ConnectionStatus();
  createInstructions();

  const gameClient = new GameClient();

  // Update connection status based on client state
  gameClient.originalRpcCall = gameClient.rpcCall;
  gameClient.rpcCall = async function(method, params) {
    try {
      const result = await this.originalRpcCall(method, params);
      connectionStatus.updateStatus("connected");
      return result;
    } catch (error) {
      connectionStatus.updateStatus("error");
      throw error;
    }
  };

  // Handle page visibility changes
  document.addEventListener("visibilitychange", () => {
    if (document.hidden) {
      gameClient.stopPolling();
    } else {
      gameClient.startPolling();
    }
  });

  // Handle beforeunload
  window.addEventListener("beforeunload", () => {
    gameClient.stopPolling();
  });
});
