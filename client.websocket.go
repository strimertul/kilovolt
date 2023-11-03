package kv

import (
	"bytes"
	"context"
	"time"

	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512000
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type WebsocketClient struct {
	hub *Hub

	// Unique ID
	uid int64

	// The websocket connection.
	conn *websocket.Conn
	addr string

	// Context with timeouts
	ctx context.Context

	// Buffered channel of outbound messages.
	send chan []byte

	options ClientOptions
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *WebsocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.CloseNow()
	}()
	c.conn.SetReadLimit(maxMessageSize)

	for c.ctx.Err() == nil {
		if err := c.readNext(); err != nil {
			return
		}
	}
}

func (c *WebsocketClient) readNext() error {
	ctx, cancel := context.WithTimeout(c.ctx, pongWait)
	defer cancel()
	_, message, err := c.conn.Read(ctx)
	if err != nil {
		c.hub.logger.Debug("read error", zap.Error(err), zap.String("client", c.addr))
		return err
	}
	message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
	c.hub.incoming <- Message{c, message}
	return nil
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WebsocketClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.CloseNow()
	}()
	for c.ctx.Err() == nil {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.conn.Close(websocket.StatusNormalClosure, "bye")
				return
			}

			c.write(message)
		case <-ticker.C:
			if err := c.conn.Ping(c.ctx); err != nil {
				return
			}
		}
	}
}

func (c *WebsocketClient) write(message []byte) {
	ctx, cancel := context.WithTimeout(c.ctx, writeWait)
	defer cancel()

	w, err := c.conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		return
	}
	w.Write(message)

	// Read other queued messages
	n := len(c.send)
	for i := 0; i < n; i++ {
		w.Write(newline)
		w.Write(<-c.send)
	}
	w.Close()
}

func (c *WebsocketClient) SetUID(uid int64) {
	c.uid = uid
}

func (c *WebsocketClient) UID() int64 {
	return c.uid
}

func (c *WebsocketClient) SendJSON(data any) {
	msg, _ := json.Marshal(data)
	c.send <- msg
}

func (c *WebsocketClient) SendMessage(data []byte) {
	c.send <- data
}

func (c *WebsocketClient) Options() ClientOptions {
	return c.options
}

func (c *WebsocketClient) Close() {
	close(c.send)
}
