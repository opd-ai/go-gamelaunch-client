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
        
        this.init();
    }
    
    async init() {
        this.setupCanvas();
        this.setupEventListeners();
        
        try {
            await this.loadTileset();
            await this.loadInitialState();
            this.startPolling();
        } catch (error) {
            console.error('Failed to initialize game client:', error);
            this.showError('Failed to connect to game server');
        }
    }
    
    setupCanvas() {
        this.canvas = document.getElementById('gameCanvas');
        if (!this.canvas) {
            // Create canvas if it doesn't exist
            this.canvas = document.createElement('canvas');
            this.canvas.id = 'gameCanvas';
            this.canvas.width = 800;
            this.canvas.height = 600;
            this.canvas.style.border = '1px solid #ccc';
            this.canvas.style.backgroundColor = '#000';
            document.body.appendChild(this.canvas);
        }
        
        this.ctx = this.canvas.getContext('2d');
        this.ctx.imageSmoothingEnabled = false; // Pixel-perfect rendering
    }
    
    setupEventListeners() {
        // Keyboard input
        document.addEventListener('keydown', (e) => {
            this.handleKeyDown(e);
        });
        
        // Canvas focus management
        this.canvas.addEventListener('click', () => {
            this.canvas.focus();
        });
        
        this.canvas.setAttribute('tabindex', '0');
        this.canvas.focus();
        
        // Prevent default browser behavior for game keys
        document.addEventListener('keydown', (e) => {
            if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Space'].includes(e.key)) {
                e.preventDefault();
            }
        });
    }
    
    async rpcCall(method, params = {}) {
        const request = {
            jsonrpc: '2.0',
            method: method,
            params: params,
            id: this.rpcId++
        };
        
        try {
            const response = await fetch('/rpc', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(request)
            });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const result = await response.json();
            
            if (result.error) {
                throw new Error(`RPC Error ${result.error.code}: ${result.error.message}`);
            }
            
            return result.result;
        } catch (error) {
            console.error(`RPC call failed for ${method}:`, error);
            throw error;
        }
    }
    
    async loadTileset() {
        try {
            const result = await this.rpcCall('tileset.fetch');
            
            if (result.tileset) {
                this.tileset = result.tileset;
                
                if (result.image_available) {
                    await this.loadTilesetImage();
                }
            }
        } catch (error) {
            console.warn('Failed to load tileset, falling back to text rendering:', error);
        }
    }
    
    async loadTilesetImage() {
        return new Promise((resolve, reject) => {
            this.tilesetImage = new Image();
            this.tilesetImage.onload = () => {
                console.log('Tileset image loaded successfully');
                resolve();
            };
            this.tilesetImage.onerror = () => {
                console.warn('Failed to load tileset image, using text rendering');
                this.tilesetImage = null;
                resolve(); // Don't reject, just fall back to text
            };
            this.tilesetImage.src = '/tileset/image?' + Date.now(); // Cache busting
        });
    }
    
    async loadInitialState() {
        const result = await this.rpcCall('game.getState');
        
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
        this.ctx = this.canvas.getContext('2d');
        this.ctx.imageSmoothingEnabled = false;
    }
    
    async startPolling() {
        if (this.polling) return;
        this.polling = true;
        
        const poll = async () => {
            if (!this.polling) return;
            
            try {
                const version = this.gameState ? this.gameState.version : 0;
                const result = await this.rpcCall('game.poll', {
                    version: version,
                    timeout: this.pollTimeout
                });
                
                if (result.changes && !result.timeout) {
                    this.applyChanges(result.changes);
                    this.render();
                }
                
                // Continue polling
                setTimeout(poll, 100);
            } catch (error) {
                console.error('Polling error:', error);
                // Retry after delay
                setTimeout(poll, 5000);
            }
        };
        
        poll();
    }
    
    stopPolling() {
        this.polling = false;
    }
    
    applyChanges(diff) {
        if (!this.gameState) {
            // If we don't have initial state, treat as full update
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
                    char: ' ',
                    fg_color: '#FFFFFF',
                    bg_color: '#000000'
                });
            }
            
            this.gameState.buffer[change.y][change.x] = change.cell;
        });
        
        // Update dimensions
        this.gameState.height = this.gameState.buffer.length;
        this.gameState.width = Math.max(...this.gameState.buffer.map(row => row.length));
        
        // Resize canvas if needed
        this.resizeCanvas();
    }
    
    render() {
        if (!this.gameState || !this.ctx) return;
        
        const cellWidth = this.tileset ? this.tileset.tile_width : 12;
        const cellHeight = this.tileset ? this.tileset.tile_height : 16;
        
        // Clear canvas
        this.ctx.fillStyle = '#000000';
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
    
    renderCell(x, y, cell, cellWidth, cellHeight) {
        const pixelX = x * cellWidth;
        const pixelY = y * cellHeight;
        
        // Background
        if (cell.bg_color && cell.bg_color !== '#000000') {
            this.ctx.fillStyle = cell.bg_color;
            this.ctx.fillRect(pixelX, pixelY, cellWidth, cellHeight);
        }
        
        // Character rendering
        if (this.tilesetImage && this.tileset && cell.tile_x !== undefined && cell.tile_y !== undefined) {
            // Render using tileset
            const srcX = cell.tile_x * this.tileset.tile_width;
            const srcY = cell.tile_y * this.tileset.tile_height;
            
            this.ctx.drawImage(
                this.tilesetImage,
                srcX, srcY, this.tileset.tile_width, this.tileset.tile_height,
                pixelX, pixelY, cellWidth, cellHeight
            );
        } else if (cell.char && cell.char !== ' ') {
            // Text fallback - size font to fit tile dimensions
            this.renderTextFallback(pixelX, pixelY, cellWidth, cellHeight, cell);
        }
    }

    renderTextFallback(x, y, width, height, cell) {
        // Calculate optimal font size to fit the tile dimensions
        // Start with a base size and adjust to fit
        let fontSize = Math.min(width * 0.8, height * 0.8);
        
        // Ensure minimum readable size
        fontSize = Math.max(fontSize, 8);
        
        // Set font properties for monospace rendering
        this.ctx.font = `${fontSize}px monospace`;
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';
        
        // Apply text styling
        if (cell.bold) {
            this.ctx.font = `bold ${fontSize}px monospace`;
        }
        
        // Set text color
        this.ctx.fillStyle = cell.fg_color || '#FFFFFF';
        
        // Handle inverse rendering
        if (cell.inverse) {
            // Swap foreground and background
            const bgColor = cell.bg_color || '#000000';
            const fgColor = cell.fg_color || '#FFFFFF';
            
            // Fill background with foreground color
            this.ctx.fillStyle = fgColor;
            this.ctx.fillRect(x, y, width, height);
            
            // Set text color to background color
            this.ctx.fillStyle = bgColor;
        }
        
        // Calculate text position (centered in tile)
        const textX = x + width / 2;
        const textY = y + height / 2;
        
        // Measure text to ensure it fits
        const char = String.fromCharCode(cell.char);
        const metrics = this.ctx.measureText(char);
        
        // If text is too wide, reduce font size
        if (metrics.width > width * 0.9) {
            fontSize = fontSize * (width * 0.9) / metrics.width;
            this.ctx.font = cell.bold ? `bold ${fontSize}px monospace` : `${fontSize}px monospace`;
        }
        
        // Render the character
        this.ctx.fillText(char, textX, textY);
        
        // Handle blinking effect (if needed)
        if (cell.blink) {
            // Could implement blinking by toggling visibility
            // For now, just render normally
        }
    }

    renderCursor(cellWidth, cellHeight) {
        if (this.gameState.cursor_x >= 0 && this.gameState.cursor_y >= 0) {
            const cursorX = this.gameState.cursor_x * cellWidth;
            const cursorY = this.gameState.cursor_y * cellHeight;
            
            // Blinking cursor effect
            const blink = Math.floor(Date.now() / 500) % 2;
            if (blink) {
                this.ctx.strokeStyle = '#FFFFFF';
                this.ctx.lineWidth = 1;
                this.ctx.strokeRect(cursorX, cursorY, cellWidth, cellHeight);
            }
        }
        
        // Schedule next render for cursor blink
        setTimeout(() => this.render(), 500);
    }
    
    handleKeyDown(event) {
        const inputEvent = {
            type: 'keydown',
            key: event.key,
            keyCode: event.keyCode,
            timestamp: Date.now()
        };
        
        this.sendInput([inputEvent]);
    }
    
    async sendInput(events) {
        try {
            await this.rpcCall('game.sendInput', { events: events });
        } catch (error) {
            console.error('Failed to send input:', error);
        }
    }
    
    showError(message) {
        // Create or update error display
        let errorDiv = document.getElementById('errorMessage');
        if (!errorDiv) {
            errorDiv = document.createElement('div');
            errorDiv.id = 'errorMessage';
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
        this.updateStatus('connecting');
    }
    
    createStatusIndicator() {
        this.statusDiv = document.createElement('div');
        this.statusDiv.id = 'connectionStatus';
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
            connecting: { bg: '#ff9500', text: 'Connecting...', color: 'white' },
            connected: { bg: '#00aa00', text: 'Connected', color: 'white' },
            disconnected: { bg: '#aa0000', text: 'Disconnected', color: 'white' },
            error: { bg: '#ff0000', text: 'Connection Error', color: 'white' }
        };
        
        const style = styles[status] || styles.disconnected;
        this.statusDiv.style.backgroundColor = style.bg;
        this.statusDiv.style.color = style.color;
        this.statusDiv.textContent = style.text;
    }
}

// Instructions panel
function createInstructions() {
    const instructions = document.createElement('div');
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
document.addEventListener('DOMContentLoaded', () => {
    const connectionStatus = new ConnectionStatus();
    createInstructions();
    
    const gameClient = new GameClient();
    
    // Update connection status based on client state
    gameClient.originalRpcCall = gameClient.rpcCall;
    gameClient.rpcCall = async function(method, params) {
        try {
            const result = await this.originalRpcCall(method, params);
            connectionStatus.updateStatus('connected');
            return result;
        } catch (error) {
            connectionStatus.updateStatus('error');
            throw error;
        }
    };
    
    // Handle page visibility changes
    document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
            gameClient.stopPolling();
        } else {
            gameClient.startPolling();
        }
    });
    
    // Handle beforeunload
    window.addEventListener('beforeunload', () => {
        gameClient.stopPolling();
    });
});