import { Page } from '@playwright/test';

export interface WebSocketMessage {
  type: string;
  [key: string]: unknown;
}

// Wait for WebSocket connection to be established
export async function waitForWebSocketConnection(page: Page, timeout = 10000): Promise<void> {
  await page.waitForFunction(
    () => {
      const ws = (window as any).ws;
      return ws && ws.readyState === WebSocket.OPEN;
    },
    { timeout }
  );
}

// Wait for a specific WebSocket message type
export async function waitForWebSocketMessage(
  page: Page,
  messageType: string,
  timeout = 5000
): Promise<WebSocketMessage> {
  return await page.evaluate(
    ({ messageType, timeout }) => {
      return new Promise<WebSocketMessage>((resolve, reject) => {
        const ws = (window as any).ws;
        if (!ws) {
          reject(new Error('WebSocket not found'));
          return;
        }

        const timeoutId = setTimeout(() => {
          reject(new Error(`Timeout waiting for message type: ${messageType}`));
        }, timeout);

        const originalOnMessage = ws.onmessage;
        ws.onmessage = (event: MessageEvent) => {
          const data = JSON.parse(event.data);
          if (data.type === messageType) {
            clearTimeout(timeoutId);
            ws.onmessage = originalOnMessage;
            resolve(data);
          } else if (originalOnMessage) {
            originalOnMessage.call(ws, event);
          }
        };
      });
    },
    { messageType, timeout }
  );
}

// Send a WebSocket message
export async function sendWebSocketMessage(page: Page, message: WebSocketMessage): Promise<void> {
  await page.evaluate((msg) => {
    const ws = (window as any).ws;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg));
    } else {
      throw new Error('WebSocket not connected');
    }
  }, message);
}

// Simulate WebSocket disconnect
export async function disconnectWebSocket(page: Page): Promise<void> {
  await page.evaluate(() => {
    const ws = (window as any).ws;
    if (ws) {
      ws.close();
    }
  });
}

// Check WebSocket connection state
export async function getWebSocketState(page: Page): Promise<number> {
  return await page.evaluate(() => {
    const ws = (window as any).ws;
    return ws ? ws.readyState : -1;
  });
}

// WebSocket ready states for reference
export const WebSocketStates = {
  CONNECTING: 0,
  OPEN: 1,
  CLOSING: 2,
  CLOSED: 3,
};

// Monitor WebSocket messages for debugging
export async function captureWebSocketMessages(
  page: Page,
  duration: number
): Promise<WebSocketMessage[]> {
  return await page.evaluate((duration) => {
    return new Promise<WebSocketMessage[]>((resolve) => {
      const messages: WebSocketMessage[] = [];
      const ws = (window as any).ws;

      if (!ws) {
        resolve([]);
        return;
      }

      const originalOnMessage = ws.onmessage;
      ws.onmessage = (event: MessageEvent) => {
        const data = JSON.parse(event.data);
        messages.push(data);
        if (originalOnMessage) {
          originalOnMessage.call(ws, event);
        }
      };

      setTimeout(() => {
        ws.onmessage = originalOnMessage;
        resolve(messages);
      }, duration);
    });
  }, duration);
}
