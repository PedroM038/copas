package network

import "encoding/json"

// Token representa o bastão de controle de acesso à rede.
type Token struct {
	HolderID int // ID do jogador que possui o token
}

// TokenMessagePayload é o payload usado na mensagem de token.
type TokenMessagePayload struct {
	Token Token `json:"token"`
}

// NewToken cria um novo token para o jogador inicial.
func NewToken(holderID int) Token {
	return Token{HolderID: holderID}
}

// NewTokenMessage cria uma mensagem de passagem de token.
func NewTokenMessage(from, to, holderID int) (Message, error) {
	payload := TokenMessagePayload{
		Token: Token{HolderID: holderID},
	}

	// Serializa o payload para json.RawMessage com tratamento de erro
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Type:    MessageToken,
		From:    from,
		To:      to,
		Payload: json.RawMessage(payloadBytes),
	}, nil
}

// ParseTokenPayload extrai o token de um payload de mensagem
func ParseTokenPayload(payload json.RawMessage) (Token, error) {
	var tokenPayload TokenMessagePayload
	err := json.Unmarshal(payload, &tokenPayload)
	if err != nil {
		return Token{}, err
	}
	return tokenPayload.Token, nil
}
