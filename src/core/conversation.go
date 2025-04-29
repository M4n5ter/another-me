package core

import (
	"strings"

	"github.com/cloudwego/eino/schema"
)

type Conversation struct {
	history []*schema.Message
}

func NewConversation() *Conversation {
	return &Conversation{
		history: make([]*schema.Message, 0),
	}
}

func (c *Conversation) Add(message *schema.Message) {
	c.history = append(c.history, message)
}

func (c *Conversation) ToMessages() []*schema.Message {
	return c.history
}

func (c *Conversation) Set(messages []*schema.Message) {
	c.history = messages
}

func (c *Conversation) String() string {
	var sb strings.Builder
	for _, message := range c.history {
		sb.WriteString(message.String())
	}
	return sb.String()
}
