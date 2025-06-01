class GameClient {
  constructor() {
    this.canvas = null;
    this.ctx = null;
    this.gameState = null;
    this.tileset = null;
    this.tilesetImage = null;
    this.polling = false;
    this.pollAbortController = null;
    this.version = 0;
    this.inputBuffer = [];
    this.inputStatistics = {
      totalEvents: 0,
      keyEvents: 0,
      mouseEvents: 0,
      lastInputTime: 0
    };
  }

  async init() {
    try {
      this.setupCanvas();
      this.setupEventListeners();
      await this.loadTileset();
      await this.loadInitialState();
      await this.startPolling();
      console.log("Game client initialized successfully");
    } catch (error) {
      console.error("Failed to initialize game client:", error);
      this.showError("Failed to initialize game client: " + error.message);
    }
  }

  setupCanvas() {
    this.canvas = document.getElementById("gameCanvas");
    if (!this.canvas) {
      throw new Error("Canvas element not found");
    }

    this.ctx = this.canvas.getContext("2d");
    if (!this.ctx) {
      throw new Error("Failed to get 2D context");
    }

    // Set initial canvas size
    this.canvas.width = 800;
    this.canvas.height = 600;

    // Ensure crisp pixel rendering
    this.ctx.imageSmoothingEnabled = false;
    this.ctx.textBaseline = "top";

    console.log("Canvas setup complete");
  }

  setupEventListeners() {
    // Keyboard events
    this.canvas.addEventListener("keydown", e => this.handleKeyDown(e));

    // Mouse events for focus
    this.canvas.addEventListener("click", () => {
      this.canvas.focus();
    });

    // Make canvas focusable
    this.canvas.tabIndex = 0;
    this.canvas.focus();

    // Window resize
    window.addEventListener("resize", () => this.resizeCanvas());

    console.log("Event listeners setup complete");
  }

  async rpcCall(method, params = {}) {
    const request = {
      jsonrpc: "2.0",
      method: method,
      params: params,
      id: Date.now()
    };

    try {
      const response = await fetch("/rpc", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify(request)
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result = await response.json();

      if (result.error) {
        throw new Error(
          `RPC Error ${result.error.code}: ${result.error.message}`
        );
      }

      return result.result;
    } catch (error) {
      console.error(`RPC call ${method} failed:`, error);
      throw error;
    }
  }

  async loadTileset() {
    try {
      console.log("Loading tileset configuration...");
      this.tileset = await this.rpcCall("tileset.fetch");

      if (this.tileset && this.tileset.source_image) {
        console.log("Loading tileset image...");
        await this.loadTilesetImage();
      }

      console.log("Tileset loaded:", this.tileset);
    } catch (error) {
      console.warn("Failed to load tileset:", error);
      // Continue without tileset
    }
  }

  async loadTilesetImage() {
    return new Promise((resolve, reject) => {
      const img = new Image();
      img.onload = () => {
        this.tilesetImage = img;
        console.log("Tileset image loaded successfully");
        resolve();
      };
      img.onerror = () => {
        console.warn("Failed to load tileset image");
        reject(new Error("Failed to load tileset image"));
      };
      img.src = "/tileset/image";
    });
  }

  async loadInitialState() {
    try {
      console.log("Loading initial game state...");
      this.gameState = await this.rpcCall("game.getState");
      this.version = this.gameState.version || 0;
      console.log("Initial state loaded, version:", this.version);
      this.render();
    } catch (error) {
      console.error("Failed to load initial state:", error);
      throw error;
    }
  }

  resizeCanvas() {
    if (!this.gameState) return;

    const cellWidth = 12; // Default cell size
    const cellHeight = 16;

    const targetWidth = this.gameState.width * cellWidth;
    const targetHeight = this.gameState.height * cellHeight;

    this.canvas.width = targetWidth;
    this.canvas.height = targetHeight;

    // Re-render after resize
    this.render();
  }

  async startPolling() {
    if (this.polling) return;

    this.polling = true;
    console.log("Starting state polling...");

    while (this.polling) {
      try {
        this.pollAbortController = new AbortController();

        const diff = await this.rpcCall("game.poll", {
          version: this.version,
          timeout: 30
        });

        if (diff && diff.version > this.version) {
          console.log(
            `State update received: v${this.version} -> v${diff.version}`,
            diff
          );
          this.applyChanges(diff);
          this.version = diff.version;
          this.render();
        } else if (diff) {
          console.log("Received diff but no version change:", diff);
        }
      } catch (error) {
        if (error.name !== "AbortError") {
          console.error("Polling error:", error);
          console.error("Error details:", {
            name: error.name,
            message: error.message,
            stack: error.stack
          });
          // Wait before retrying
          await new Promise(resolve => setTimeout(resolve, 1000));
        }
      }
    }
  }

  stopPolling() {
    this.polling = false;
    if (this.pollAbortController) {
      this.pollAbortController.abort();
    }
    console.log("Stopped state polling");
  }

  render() {
    if (!this.gameState || !this.ctx) return;

    const cellWidth = this.canvas.width / this.gameState.width;
    const cellHeight = this.canvas.height / this.gameState.height;

    // Clear canvas with black background
    this.ctx.fillStyle = "#000000";
    this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);

    // Render each cell
    for (let y = 0; y < this.gameState.height; y++) {
      for (let x = 0; x < this.gameState.width; x++) {
        if (this.gameState.buffer[y] && this.gameState.buffer[y][x]) {
          const cell = this.gameState.buffer[y][x];
          this.renderCell(x, y, cell, cellWidth, cellHeight);
        }
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
    let bgColor = this.parseColor(cell.bg_color) || "#000000";
    let fgColor = this.parseColor(cell.fg_color) || "#FFFFFF";

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

    // Parse and validate colors
    const bgColor = this.parseColor(cell.bg_color) || "#000000";
    const fgColor = this.parseColor(cell.fg_color) || "#FFFFFF";

    // Always render background first
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
      // Render using tileset with color support
      this.renderTilesetCell(
        pixelX,
        pixelY,
        cellWidth,
        cellHeight,
        cell,
        fgColor
      );
    } else {
      // Text fallback rendering
      this.renderTextFallback(pixelX, pixelY, cellWidth, cellHeight, cell);
    }
  }

  renderTilesetCell(x, y, width, height, cell, fgColor) {
    const srcX = cell.tile_x * this.tileset.tile_width;
    const srcY = cell.tile_y * this.tileset.tile_height;

    // For colored tileset rendering, apply color tinting if needed
    if (fgColor && fgColor !== "#FFFFFF") {
      // Save the current composite operation
      const oldComposite = this.ctx.globalCompositeOperation;

      // Draw the tile
      this.ctx.drawImage(
        this.tilesetImage,
        srcX,
        srcY,
        this.tileset.tile_width,
        this.tileset.tile_height,
        x,
        y,
        width,
        height
      );

      // Apply color tinting
      this.ctx.globalCompositeOperation = "multiply";
      this.ctx.fillStyle = fgColor;
      this.ctx.fillRect(x, y, width, height);

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
        x,
        y,
        width,
        height
      );
    }
  }

  // Enhanced color parsing method
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
      red: "#800000",
      green: "#008000",
      yellow: "#808000",
      blue: "#000080",
      magenta: "#800080",
      cyan: "#008080",
      white: "#C0C0C0",
      bright_black: "#808080",
      bright_red: "#FF0000",
      bright_green: "#00FF00",
      bright_yellow: "#FFFF00",
      bright_blue: "#0000FF",
      bright_magenta: "#FF00FF",
      bright_cyan: "#00FFFF",
      bright_white: "#FFFFFF",
      // Add more color mappings as needed
      gray: "#808080",
      grey: "#808080",
      dark_red: "#800000",
      dark_green: "#008000",
      dark_blue: "#000080",
      orange: "#FF8000",
      purple: "#800080",
      brown: "#A52A2A",
      pink: "#FFC0CB"
    };

    return namedColors[color.toLowerCase()] || color;
  }

  applyChanges(diff) {
    if (!this.gameState || !diff) return;

    // Update cursor position if provided
    if (diff.cursor_x !== undefined) {
      this.gameState.cursor_x = diff.cursor_x;
    }
    if (diff.cursor_y !== undefined) {
      this.gameState.cursor_y = diff.cursor_y;
    }
    if (diff.version !== undefined) {
      this.gameState.version = diff.version;
    }
    if (diff.timestamp !== undefined) {
      this.gameState.timestamp = diff.timestamp;
    }

    // Apply cell changes if they exist and are iterable
    if (diff.changes && Array.isArray(diff.changes)) {
      for (const change of diff.changes) {
        if (
          change &&
          change.y >= 0 &&
          change.y < this.gameState.height &&
          change.x >= 0 &&
          change.x < this.gameState.width
        ) {
          if (!this.gameState.buffer[change.y]) {
            this.gameState.buffer[change.y] = [];
          }
          this.gameState.buffer[change.y][change.x] = change.cell;
        }
      }
      console.log(`Applied ${diff.changes.length} changes`);
    } else {
      console.log(
        "No changes to apply or changes is not an array:",
        diff.changes
      );
    }
  }

  renderCursor(cellWidth, cellHeight) {
    if (!this.gameState) return;

    const x = this.gameState.cursor_x * cellWidth;
    const y = this.gameState.cursor_y * cellHeight;

    // Draw cursor as a blinking rectangle
    const blink = Math.floor(Date.now() / 500) % 2;
    if (blink) {
      this.ctx.strokeStyle = "#FFFFFF";
      this.ctx.lineWidth = 2;
      this.ctx.strokeRect(x + 1, y + 1, cellWidth - 2, cellHeight - 2);
    }
  }

  handleKeyDown(event) {
    // Prevent default browser behavior for game keys
    const gameKeys = [
      "ArrowUp",
      "ArrowDown",
      "ArrowLeft",
      "ArrowRight",
      "Enter",
      "Escape",
      "Tab",
      "Backspace",
      "Delete",
      " " // Space
    ];

    if (gameKeys.includes(event.key) || event.key.length === 1) {
      event.preventDefault();
    }

    // Create input event
    const inputEvent = {
      type: "key",
      key: event.key,
      code: event.code,
      ctrl: event.ctrlKey,
      alt: event.altKey,
      shift: event.shiftKey,
      meta: event.metaKey,
      timestamp: Date.now()
    };

    // Add to input buffer
    this.inputBuffer.push(inputEvent);
    this.inputStatistics.totalEvents++;
    this.inputStatistics.keyEvents++;
    this.inputStatistics.lastInputTime = Date.now();

    // Send input immediately for better responsiveness
    this.sendInput([inputEvent]);

    // Log input statistics periodically
    if (this.inputStatistics.totalEvents % 10 === 0) {
      this.logInputStatistics();
    }
  }

  logInputStatistics() {
    console.log("Input Statistics:", {
      total: this.inputStatistics.totalEvents,
      keys: this.inputStatistics.keyEvents,
      mouse: this.inputStatistics.mouseEvents,
      lastInput: new Date(
        this.inputStatistics.lastInputTime
      ).toLocaleTimeString()
    });
  }

  async sendInput(events) {
    try {
      await this.rpcCall("game.sendInput", { events });
      console.log(`Sent ${events.length} input event(s)`);
    } catch (error) {
      console.error("Failed to send input:", error);
      this.showError("Failed to send input: " + error.message);
    }
  }

  showError(message) {
    console.error(message);

    // Remove existing error message
    const existingError = document.getElementById("errorMessage");
    if (existingError) {
      existingError.remove();
    }

    // Create error message element
    const errorDiv = document.createElement("div");
    errorDiv.id = "errorMessage";
    errorDiv.style.cssText = `
      position: fixed;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      background: rgba(255, 0, 0, 0.9);
      color: white;
      padding: 20px;
      border-radius: 5px;
      font-family: monospace;
      font-size: 14px;
      max-width: 400px;
      text-align: center;
      z-index: 10000;
      box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3);
    `;

    errorDiv.innerHTML = `
      <strong>Error</strong><br>
      ${message}<br>
      <button onclick="this.parentElement.remove()" style="margin-top: 10px; padding: 5px 10px;">Close</button>
    `;

    document.body.appendChild(errorDiv);

    // Auto-remove after 10 seconds
    setTimeout(() => {
      if (errorDiv.parentElement) {
        errorDiv.remove();
      }
    }, 10000);
  }
}

// Connection status indicator
class ConnectionStatus {
  constructor() {
    this.status = "disconnected";
    this.element = null;
    this.createStatusIndicator();
  }

  createStatusIndicator() {
    this.element = document.createElement("div");
    this.element.id = "connectionStatus";
    this.element.style.cssText = `
      position: fixed;
      top: 10px;
      right: 10px;
      padding: 8px 12px;
      border-radius: 4px;
      font-family: monospace;
      font-size: 12px;
      font-weight: bold;
      z-index: 1000;
      backdrop-filter: blur(4px);
      border: 1px solid rgba(255, 255, 255, 0.2);
    `;

    document.body.appendChild(this.element);
    this.updateStatus("connecting");
  }

  updateStatus(status) {
    this.status = status;

    const statusConfig = {
      connecting: {
        text: "Connecting...",
        color: "#FFA500",
        bg: "rgba(255, 165, 0, 0.1)"
      },
      connected: {
        text: "Connected",
        color: "#00FF00",
        bg: "rgba(0, 255, 0, 0.1)"
      },
      disconnected: {
        text: "Disconnected",
        color: "#FF0000",
        bg: "rgba(255, 0, 0, 0.1)"
      },
      error: { text: "Error", color: "#FF0000", bg: "rgba(255, 0, 0, 0.2)" }
    };

    const config = statusConfig[status] || statusConfig.disconnected;

    this.element.textContent = config.text;
    this.element.style.color = config.color;
    this.element.style.backgroundColor = config.bg;
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
      connectionStatus.updateStatus("connecting");
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

  // Start the game client
  gameClient.init().catch(error => {
    console.error("Failed to start game client:", error);
    connectionStatus.updateStatus("error");
  });
});
