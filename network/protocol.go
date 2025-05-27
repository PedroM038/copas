package network

import (
	"encoding/json"
)

// MessageType define os tipos de mensagens possíveis no protocolo.
type MessageType string

const (
	MessageToken     MessageType = "TOKEN"
	MessagePlay      MessageType = "PLAY"
	MessageState     MessageType = "STATE"
	MessageDiscovery MessageType = "DISCOVERY"
	MessagePass      MessageType = "PASS"
)

// Message representa a estrutura básica de uma mensagem no protocolo.
type Message struct {
	Type    MessageType     `json:"type"`
	From    int             `json:"from"`
	To      int             `json:"to"`
	Payload json.RawMessage `json:"payload"`
}

// PlayMessagePayload para jogadas de cartas
type PlayMessagePayload struct {
	PlayerID int             `json:"player_id"`
	Card     json.RawMessage `json:"card"` // Será um model.Card serializado
}

// StateMessagePayload para sincronização do estado do jogo
type StateMessagePayload struct {
	GameState json.RawMessage `json:"game_state"` // Será um model.Game serializado
}

// PassMessagePayload para troca de cartas
type PassMessagePayload struct {
	PlayerID int               `json:"player_id"`
	Cards    []json.RawMessage `json:"cards"`  // Array de model.Card
	Target   int               `json:"target"` // ID do jogador alvo
}

// DiscoveryMessagePayload para descoberta de jogadores
type DiscoveryMessagePayload struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name"`
	IsReady    bool   `json:"is_ready"`
}

// EncodeMessage serializa uma mensagem para JSON.
func EncodeMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// DecodeMessage desserializa uma mensagem JSON para struct Message.
func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}

// NewPlayMessage cria uma mensagem de jogada
func NewPlayMessage(from, to, playerID int, card json.RawMessage) (Message, error) {
	payload := PlayMessagePayload{
		PlayerID: playerID,
		Card:     card,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:    MessagePlay,
		From:    from,
		To:      to,
		Payload: json.RawMessage(payloadBytes),
	}, nil
}

// NewStateMessage cria uma mensagem de estado
func NewStateMessage(from, to int, gameState json.RawMessage) (Message, error) {
	payload := StateMessagePayload{
		GameState: gameState,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:    MessageState,
		From:    from,
		To:      to,
		Payload: json.RawMessage(payloadBytes),
	}, nil
}

// NewPassMessage cria uma mensagem de troca de cartas
func NewPassMessage(from, to, playerID, target int, cards []json.RawMessage) (Message, error) {
	payload := PassMessagePayload{
		PlayerID: playerID,
		Cards:    cards,
		Target:   target,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:    MessagePass,
		From:    from,
		To:      to,
		Payload: json.RawMessage(payloadBytes),
	}, nil
}

// NewDiscoveryMessage cria uma mensagem de descoberta
func NewDiscoveryMessage(from, to, playerID int, playerName string, isReady bool) (Message, error) {
	payload := DiscoveryMessagePayload{
		PlayerID:   playerID,
		PlayerName: playerName,
		IsReady:    isReady,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Type:    MessageDiscovery,
		From:    from,
		To:      to,
		Payload: json.RawMessage(payloadBytes),
	}, nil
}
