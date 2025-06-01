package network

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// Tipos de mensagem no protocolo
const (
	MSG_TOKEN     = "TOKEN"     // Bastão
	MSG_DATA      = "DATA"      // Mensagem de dados
	MSG_GAME      = "GAME"      // Mensagem específica do jogo
	MSG_HEARTBEAT = "HEARTBEAT" // Para manter a rede ativa
	MSG_BROADCAST = "BROADCAST" // Mensagem de broadcast
	MSG_ERROR     = "ERROR"     // Mensagem de erro
	MSG_ACK       = "ACK"       // Confirmação de recebimento
)

// Prioridades de mensagem
const (
	PRIORITY_LOW    = 0
	PRIORITY_NORMAL = 1
	PRIORITY_HIGH   = 2
)

// Estrutura principal da mensagem
type Message struct {
	Type      string      `json:"type"`
	From      int         `json:"from"`
	To        int         `json:"to"`
	Content   interface{} `json:"content"`
	Hops      int         `json:"hops"`
	MessageID string      `json:"message_id"`
	Timestamp int64       `json:"timestamp"`
	Priority  int         `json:"priority"` // 0=baixa, 1=normal, 2=alta
}

// Estrutura para conteúdo específico do jogo
type GameContent struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// Estrutura para mensagens de erro
type ErrorContent struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Details     string `json:"details"`
}

// Estrutura para ACK
type AckContent struct {
	OriginalMessageID string `json:"original_message_id"`
	Status            string `json:"status"` // "success" ou "error"
}

// Gera um ID único para a mensagem
func generateMessageID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// Cria uma nova mensagem
func NewMessage(msgType string, from, to int, content interface{}, priority int) *Message {
	return &Message{
		Type:      msgType,
		From:      from,
		To:        to,
		Content:   content,
		Hops:      0,
		MessageID: generateMessageID(),
		Timestamp: time.Now().Unix(),
		Priority:  priority,
	}
}

// Cria uma mensagem de jogo
func NewGameMessage(from, to int, action string, data interface{}) *Message {
	content := GameContent{
		Action: action,
		Data:   data,
	}
	return NewMessage(MSG_GAME, from, to, content, PRIORITY_NORMAL)
}

// Cria uma mensagem de erro
func NewErrorMessage(from, to int, code int, description, details string) *Message {
	content := ErrorContent{
		Code:        code,
		Description: description,
		Details:     details,
	}
	return NewMessage(MSG_ERROR, from, to, content, PRIORITY_HIGH)
}

// Cria uma mensagem de ACK
func NewAckMessage(from, to int, originalMessageID, status string) *Message {
	content := AckContent{
		OriginalMessageID: originalMessageID,
		Status:            status,
	}
	return NewMessage(MSG_ACK, from, to, content, PRIORITY_HIGH)
}

// Cria uma mensagem de broadcast
func NewBroadcastMessage(from int, content interface{}) *Message {
	return NewMessage(MSG_BROADCAST, from, -1, content, PRIORITY_NORMAL)
}

// Codifica a mensagem para JSON
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// Decodifica a mensagem de JSON
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar mensagem: %v", err)
	}
	return &msg, nil
}

// Incrementa o número de hops
func (m *Message) IncrementHops() {
	m.Hops++
}

// Verifica se a mensagem deve ser descartada por excesso de hops
func (m *Message) ShouldDiscard(maxHops int) bool {
	return m.Hops >= maxHops
}

// Verifica se a mensagem é válida
func (m *Message) IsValid() bool {
	if m.Type == "" || m.MessageID == "" {
		return false
	}

	validTypes := map[string]bool{
		MSG_TOKEN:     true,
		MSG_DATA:      true,
		MSG_GAME:      true,
		MSG_HEARTBEAT: true,
		MSG_BROADCAST: true,
		MSG_ERROR:     true,
		MSG_ACK:       true,
	}

	return validTypes[m.Type]
}

// Verifica se a mensagem é para este nó
func (m *Message) IsForNode(nodeID int) bool {
	return m.To == nodeID || m.To == -1 // -1 para broadcast
}

// Cria uma cópia da mensagem
func (m *Message) Copy() *Message {
	return &Message{
		Type:      m.Type,
		From:      m.From,
		To:        m.To,
		Content:   m.Content,
		Hops:      m.Hops,
		MessageID: m.MessageID,
		Timestamp: m.Timestamp,
		Priority:  m.Priority,
	}
}

// Converte o conteúdo da mensagem para o tipo esperado
func (m *Message) GetGameContent() (*GameContent, error) {
	if m.Type != MSG_GAME {
		return nil, fmt.Errorf("mensagem não é do tipo GAME")
	}

	data, err := json.Marshal(m.Content)
	if err != nil {
		return nil, err
	}

	var gameContent GameContent
	err = json.Unmarshal(data, &gameContent)
	if err != nil {
		return nil, err
	}

	return &gameContent, nil
}

// Converte o conteúdo da mensagem para ErrorContent
func (m *Message) GetErrorContent() (*ErrorContent, error) {
	if m.Type != MSG_ERROR {
		return nil, fmt.Errorf("mensagem não é do tipo ERROR")
	}

	data, err := json.Marshal(m.Content)
	if err != nil {
		return nil, err
	}

	var errorContent ErrorContent
	err = json.Unmarshal(data, &errorContent)
	if err != nil {
		return nil, err
	}

	return &errorContent, nil
}

// Converte o conteúdo da mensagem para AckContent
func (m *Message) GetAckContent() (*AckContent, error) {
	if m.Type != MSG_ACK {
		return nil, fmt.Errorf("mensagem não é do tipo ACK")
	}

	data, err := json.Marshal(m.Content)
	if err != nil {
		return nil, err
	}

	var ackContent AckContent
	err = json.Unmarshal(data, &ackContent)
	if err != nil {
		return nil, err
	}

	return &ackContent, nil
}
