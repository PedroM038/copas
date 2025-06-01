package network

import (
	"encoding/json"
	"fmt"
	"time"
)

// Estrutura do token (bastão)
type Token struct {
	ID        string                 `json:"id"`
	Owner     int                    `json:"owner"`
	Timestamp int64                  `json:"timestamp"`
	Sequence  uint64                 `json:"sequence"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Estados do token
const (
	TOKEN_STATE_NONE    = "NONE"    // Nó não possui token
	TOKEN_STATE_WAITING = "WAITING" // Nó está esperando token
	TOKEN_STATE_HOLDING = "HOLDING" // Nó possui o token
	TOKEN_STATE_PASSING = "PASSING" // Nó está passando o token
)

// Controlador de token
type TokenController struct {
	nodeID       int
	state        string
	token        *Token
	lastSeen     time.Time
	sequence     uint64
	waitingQueue []*Message // Fila de mensagens esperando o token
}

// Cria um novo controlador de token
func NewTokenController(nodeID int) *TokenController {
	return &TokenController{
		nodeID:       nodeID,
		state:        TOKEN_STATE_NONE,
		token:        nil,
		lastSeen:     time.Now(),
		sequence:     0,
		waitingQueue: make([]*Message, 0),
	}
}

// Cria um novo token (usado apenas pelo nó inicial)
func NewToken(ownerID int) *Token {
	return &Token{
		ID:        generateMessageID(),
		Owner:     ownerID,
		Timestamp: time.Now().Unix(),
		Sequence:  1,
		Data:      make(map[string]interface{}),
	}
}

// Verifica se o nó possui o token
func (tc *TokenController) HasToken() bool {
	return tc.state == TOKEN_STATE_HOLDING && tc.token != nil
}

// Verifica se o nó está esperando o token
func (tc *TokenController) IsWaiting() bool {
	return tc.state == TOKEN_STATE_WAITING
}

// Recebe o token
func (tc *TokenController) ReceiveToken(token *Token) error {
	if token == nil {
		return fmt.Errorf("token inválido")
	}

	tc.token = token
	tc.token.Owner = tc.nodeID
	tc.token.Timestamp = time.Now().Unix()
	tc.token.Sequence++
	tc.state = TOKEN_STATE_HOLDING
	tc.lastSeen = time.Now()

	return nil
}

// Passa o token para o próximo nó
func (tc *TokenController) PassToken(nextNodeID int) (*Message, error) {
	if !tc.HasToken() {
		return nil, fmt.Errorf("nó não possui o token")
	}

	tc.state = TOKEN_STATE_PASSING
	tc.token.Owner = nextNodeID
	tc.token.Timestamp = time.Now().Unix()

	// Cria mensagem de token
	msg := NewMessage(MSG_TOKEN, tc.nodeID, nextNodeID, tc.token, PRIORITY_HIGH)

	// Remove o token local após um pequeno delay (será removido após confirmação)
	tc.scheduleTokenRemoval()

	return msg, nil
}

// Agenda a remoção do token local
func (tc *TokenController) scheduleTokenRemoval() {
	go func() {
		time.Sleep(100 * time.Millisecond) // Pequeno delay
		tc.token = nil
		tc.state = TOKEN_STATE_NONE
	}()
}

// Confirma que o token foi passado com sucesso
func (tc *TokenController) ConfirmTokenPassed() {
	tc.token = nil
	tc.state = TOKEN_STATE_NONE
}

// Solicita o token (coloca em estado de espera)
func (tc *TokenController) RequestToken() {
	if tc.state == TOKEN_STATE_NONE {
		tc.state = TOKEN_STATE_WAITING
	}
}

// Adiciona mensagem à fila de espera
func (tc *TokenController) AddToWaitingQueue(msg *Message) {
	tc.waitingQueue = append(tc.waitingQueue, msg)
}

// Obtém mensagens da fila de espera
func (tc *TokenController) GetWaitingMessages() []*Message {
	messages := make([]*Message, len(tc.waitingQueue))
	copy(messages, tc.waitingQueue)
	tc.waitingQueue = tc.waitingQueue[:0] // Limpa a fila
	return messages
}

// Verifica se tem mensagens esperando
func (tc *TokenController) HasWaitingMessages() bool {
	return len(tc.waitingQueue) > 0
}

// Obtém o estado atual do token
func (tc *TokenController) GetState() string {
	return tc.state
}

// Obtém informações do token atual
func (tc *TokenController) GetTokenInfo() map[string]interface{} {
	info := map[string]interface{}{
		"state":              tc.state,
		"node_id":            tc.nodeID,
		"last_seen":          tc.lastSeen.Unix(),
		"waiting_queue_size": len(tc.waitingQueue),
	}

	if tc.token != nil {
		info["token"] = map[string]interface{}{
			"id":       tc.token.ID,
			"owner":    tc.token.Owner,
			"sequence": tc.token.Sequence,
		}
	}

	return info
}

// Verifica se o token está perdido (timeout)
func (tc *TokenController) IsTokenLost(timeout time.Duration) bool {
	return time.Since(tc.lastSeen) > timeout && tc.state != TOKEN_STATE_HOLDING
}

// Reinicia o controlador de token
func (tc *TokenController) Reset() {
	tc.token = nil
	tc.state = TOKEN_STATE_NONE
	tc.waitingQueue = tc.waitingQueue[:0]
	tc.lastSeen = time.Now()
}

// Codifica o token para ser enviado na mensagem
func (t *Token) Encode() ([]byte, error) {
	return json.Marshal(t)
}

// Decodifica o token da mensagem
func DecodeToken(data []byte) (*Token, error) {
	var token Token
	err := json.Unmarshal(data, &token)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar token: %v", err)
	}
	return &token, nil
}

// Valida se o token é válido
func (t *Token) IsValid() bool {
	return t.ID != "" && t.Owner >= 0 && t.Sequence > 0
}

// Obtém token da mensagem
func GetTokenFromMessage(msg *Message) (*Token, error) {
	if msg.Type != MSG_TOKEN {
		return nil, fmt.Errorf("mensagem não é do tipo TOKEN")
	}

	data, err := json.Marshal(msg.Content)
	if err != nil {
		return nil, err
	}

	return DecodeToken(data)
}

// Define dados customizados no token
func (t *Token) SetData(key string, value interface{}) {
	if t.Data == nil {
		t.Data = make(map[string]interface{})
	}
	t.Data[key] = value
}

// Obtém dados customizados do token
func (t *Token) GetData(key string) (interface{}, bool) {
	if t.Data == nil {
		return nil, false
	}
	value, exists := t.Data[key]
	return value, exists
}
