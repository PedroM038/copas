package network

import (
    "encoding/json"
)

// MessageType define os tipos de mensagens possíveis no protocolo.
type MessageType string

const (
    MessageToken MessageType = "TOKEN"
    MessagePlay  MessageType = "PLAY"
    MessageState MessageType = "STATE"
)

// Message representa a estrutura básica de uma mensagem no protocolo.
type Message struct {
    Type    MessageType `json:"type"`
    From    int         `json:"from"`
    To      int         `json:"to"`
    Payload interface{} `json:"payload"`
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